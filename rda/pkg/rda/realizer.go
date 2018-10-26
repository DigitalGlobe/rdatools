// Copyright © 2018 DigitalGlobe
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package rda

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"

	"os"

	"path/filepath"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
)

// TileInfo holds information about rda tiles that are local on disk.
type TileInfo struct {
	// FilePath is where this tile is located on disk.
	FilePath string

	// XTile is the x coordinate of this tile in reference to the TileWindow it came from.
	XTile int

	// YTile is the y coordinate of this tile in reference to the TileWindow it came from.
	YTile int
}

// Realizer realizes tiles out of RDA.
type Realizer struct {
	// Client is the http client to use when realizing.
	Client *retryablehttp.Client

	// NumParallel is how many tile requests to have going
	// concurrently.  By default we use 4 times the number of CPUs
	// detected.
	NumParallel int
}

// RealizeGraph will retrieve all the tiles from in the RDA
// graphID/nodeID combo as desribed by tileWindow.
//
// graphID and nodeID are the RDA graph and node you are trying to
// realize.  All tiles described in tileWindow will be downloaded, so
// modify this to suite your needs.  outDir is where the tiles will be
// placed, and this directory will be created for you if not present.
// onFinished is called whenever a tile is finished downloading, nil
// can be provided for this argument.
func (r *Realizer) RealizeGraph(ctx context.Context, graphID, nodeID string, tileWindow TileWindow, outDir string, onFinished func() int) ([]TileInfo, error) {
	tileURL := fmt.Sprintf(graphTileEndpoint, graphID, nodeID)
	return r.realize(ctx, tileURL, nil, tileWindow, outDir, onFinished)
}

// RealizeTemplate will retrieve all the tiles the Realizer knows about. If
// tiles already exist, they are not downloaded again.
//
// graphID and nodeID are the RDA graph and node you are trying to
// realize.  All tiles described in tileWindow will be downloaded, so
// modify this to suite your needs.  outDir is where the tiles will be
// placed, and this directory will be created for you if not present.
// onFinished is called whenever a tile is finished downloading, nil
// can be provided for this argument.
func (r *Realizer) RealizeTemplate(ctx context.Context, templateID string, qp url.Values, tileWindow TileWindow, outDir string, onFinished func() int) ([]TileInfo, error) {
	tileURL := fmt.Sprintf(templateTileEnpoint, templateID)
	return r.realize(ctx, tileURL, qp, tileWindow, outDir, onFinished)
}

type realizeError struct {
	errors []error
}

func (r *realizeError) addError(err error) *realizeError {
	// Note that this is safe to call on a nil reciever.
	if r == nil {
		return &realizeError{errors: []error{err}}
	}
	r.errors = append(r.errors, err)
	return r
}

func (r *realizeError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d error(s) during realization:\n", len(r.errors))
	for i, err := range r.errors {
		fmt.Fprintf(&sb, "\terror %d: %v\n", i+1, err)
	}
	return sb.String()
}

func (r *Realizer) realize(ctx context.Context, tileURL string, qp url.Values, tileWindow TileWindow, outDir string, onFinished func() int) ([]TileInfo, error) {
	if err := os.MkdirAll(outDir, 0777); err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}
	jobsIn := make(chan realizeJob)
	jobsOut := make(chan realizeJob)

	if onFinished == nil {
		onFinished = func() int { return 0 }
	}

	// Spin up some workers. Note these workers will only shut
	// down once jobsIn is closed and jobsOut is drained.
	np := r.NumParallel
	if r.NumParallel < 1 {
		np = 4 * runtime.NumCPU()
	}
	for i := 0; i < np; i++ {
		wg.Add(1)
		go func(jobsIn <-chan realizeJob, jobsOut chan<- realizeJob) {
			defer wg.Done()
			for job := range jobsIn {
				r.processJob(ctx, job, jobsOut, onFinished)
			}
		}(jobsIn, jobsOut)
	}

	// Launch tile requests. Note here is where we watch ctx for
	// signals and if we get one, we close the jobsIn.  This in turn
	// will let the workers finish and shut down gracefully.
	wg.Add(1)
	go func(jobsIn chan<- realizeJob) {
		defer close(jobsIn)
		defer wg.Done()

		for x := tileWindow.MinTileX; x <= tileWindow.MaxTileX; x++ {
			for y := tileWindow.MinTileY; y <= tileWindow.MaxTileY; y++ {
				rj := realizeJob{
					filePath: filepath.Join(outDir, fmt.Sprintf("tile_%d_%d.tif", x, y)),
					xTile:    x,
					yTile:    y,
				}

				tURL := fmt.Sprintf(tileURL, x, y)
				u, err := url.Parse(tURL)
				if err != nil {
					rj.err = errors.Wrapf(err, "failed parsing %s during tile realization", tURL)
				} else {
					u.RawQuery = qp.Encode()
					rj.url = u.String()
				}
				select {
				case jobsIn <- rj:
				case <-ctx.Done():
					return
				}
			}
		}
	}(jobsIn)

	// Close jobsOut once workers are finished.  This will let our
	// main routine drain the output channel and return all
	// successfully downloaded tiles.
	go func() {
		defer close(jobsOut)
		wg.Wait()
	}()

	// Processed successfully finished tiles.  By design this will
	// wait until all works shut down, so we should nab all
	// successfully downloaded tiles before returning.
	completedTiles := []TileInfo{}
	var jobserr *realizeError
	for job := range jobsOut {
		if job.err != nil {
			jobserr = jobserr.addError(job.err)
		} else {
			completedTiles = append(completedTiles, TileInfo{FilePath: job.filePath, XTile: job.xTile, YTile: job.yTile})
		}
	}
	if jobserr != nil {
		return completedTiles, jobserr
	}
	return completedTiles, nil
}

// processJob does the actual download of a tile and writing of it to
// disk.  This should be safe to run concurrently.
func (r *Realizer) processJob(ctx context.Context, job realizeJob, jobsOut chan<- realizeJob, onFinished func() int) {
	// Note we always send our input jobs to the output channel, adding an error to job if one occurred.
	defer func() { jobsOut <- job }()
	defer onFinished()

	// Already errored, so just pass the message along.
	if job.err != nil {
		return
	}

	// If tile is already present, don't download it.
	if _, err := os.Stat(job.filePath); !os.IsNotExist(err) {
		return
	}

	// Download the tile from RDA and dump it to disk.
	req, err := retryablehttp.NewRequest("GET", job.url, nil)
	if err != nil {
		job.err = errors.Wrapf(err, "failed forming request for tile at %s", job.url)
		return
	}
	req = req.WithContext(ctx)

	res, err := r.Client.Do(req)
	if err != nil {
		job.err = errors.Wrapf(err, "failed requesting tile at %s", job.url)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		job.err = ResponseToError(res.Body, fmt.Sprintf("failed requesting tile at %s, status: %d %s", job.url, res.StatusCode, res.Status))
		return
	}

	f, err := os.Create(job.filePath)
	if err != nil {
		job.err = errors.Wrapf(err, "failed creating file for tile at %s", job.url)
		return
	}
	if _, err := io.Copy(f, res.Body); err != nil {
		err = errors.Wrapf(err, "failed copying tile at %s to disk", job.url)
		if nerr := f.Close(); nerr != nil {
			err = errors.WithMessagef(err, "failed closing partially downloaded tile at %s: %v", job.filePath, nerr)
		}
		if nerr := os.Remove(job.filePath); nerr != nil {
			err = errors.WithMessagef(err, "failed removing file for partially downloaded tile at %s, err: %v", job.filePath, nerr)
		}
		job.err = err
		return
	}
	if err := f.Close(); err != nil {
		err = errors.Wrapf(err, "failed closing file %s for downloaded tile", job.filePath)
		if nerr := os.Remove(job.filePath); nerr != nil {
			err = errors.WithMessagef(err, "failed removing file for downloaded tile at %s: %v", job.filePath, nerr)
		}
		job.err = err
	}
}

type realizeJob struct {
	url      string
	filePath string
	xTile    int
	yTile    int
	err      error
}
