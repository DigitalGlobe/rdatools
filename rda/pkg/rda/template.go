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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
)

// Template contains methods for interacting with the RDA template APIs.
type Template struct {
	templateID  string
	queryParams url.Values
	window      TileWindow

	client *retryablehttp.Client

	numParallel  int
	progressFunc func() int
}

// NewTemplate returns a configured Template.
func NewTemplate(templateID string, client *retryablehttp.Client, options ...TemplateOption) *Template {
	t := &Template{
		templateID:  templateID,
		queryParams: make(url.Values),

		client: client,

		numParallel:  4 * runtime.NumCPU(),
		progressFunc: func() int { return 0 },
	}

	// Apply any options provided.
	for _, opt := range options {
		opt(t)
	}
	return t
}

// TemplateOption sets options on a Template
type TemplateOption func(*Template)

// NumParallel lets you set the max concurrency used when accessing
// RDA template API endpoints.  This is primarily for controlling how
// many tiles to concurrently download from RDA when realizing a
// template.
func NumParallel(val int) TemplateOption {
	return func(t *Template) {
		if val > 0 {
			t.numParallel = val
		}
	}
}

// AddParameter populates the template parameter named by key with val.
func AddParameter(key, val string) TemplateOption {
	return func(t *Template) {
		t.queryParams.Add(key, val)
	}
}

// WithWindow adds a TileWindow to use when realizing imagery from RDA.
func WithWindow(window TileWindow) TemplateOption {
	return func(t *Template) {
		t.window = window
	}
}

// WithProgressFunc will set progressFunc to be called everytime a tile is downloaded during realization.
func WithProgressFunc(progressFunc func() int) TemplateOption {
	return func(t *Template) {
		t.progressFunc = progressFunc
	}
}

// Describe returns a description of the RDA template.
func (t *Template) Describe() (*Graph, error) {
	ep := urls.describeURL(t.templateID)

	res, err := t.client.Get(ep)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to form GET for fetching template description")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, ResponseToError(res.Body, fmt.Sprintf("failed fetching template description from %s, HTTP Status: %s", ep, res.Status))
	}

	return NewGraphFromAPI(res.Body)
}

// Upload uploads the graph g to the RDA API, returning the RDA template ID associated with it.
func (t *Template) Upload(g *Graph) (string, error) {
	body, err := json.Marshal(g)
	if err != nil {
		return "", errors.Wrap(err, "failed forming request body for RDA template upload")
	}

	res, err := t.client.Post(urls.uploadURL(), "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", errors.Wrap(err, "failed posting template to RDA")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", ResponseToError(res.Body, fmt.Sprintf("failed posting RDA template, HTTP Status: %s", res.Status))
	}

	// Decode the response body; should be a Graph with the id filled in.
	resp := rdaGraph{}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return "", errors.Wrap(err, "failed decoding RDA API response after posting an rda graph")
	}

	return resp.ID, nil
}

// Metadata returns the RDA metadata describing the template.
func (t *Template) Metadata() (*Metadata, error) {
	ep, err := urls.metadataURL(t.templateID, t.queryParams)
	if err != nil {
		return nil, err
	}

	res, err := t.client.Get(ep)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to form GET for fetching metadata")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, ResponseToError(res.Body, fmt.Sprintf("failed fetching metadata from %s, HTTP Status: %s", ep, res.Status))
	}

	md := Metadata{}
	if err := json.NewDecoder(res.Body).Decode(&md); err != nil {
		return nil, errors.Wrap(err, "failed parsing template metadata from response")
	}
	md.setTileGeoreferencing()

	return &md, nil
}

// BatchRealize asks RDA's batch materialization to generate the imagery described by the template and its parameters.
func (t *Template) BatchRealize(ctx context.Context, format BatchFormat) (*BatchResponse, error) {
	// Make the request.
	reqBody := BatchRequest{
		ImageReference: ImageReference{
			TemplateID: t.templateID,
		},
		OutputFormat:    format,
		CropGeometryWKT: t.window.wkt(),
	}

	// Parse out the template's query parameters to where they need to be in the batch request body.
	tp := make(map[string]string)
	for key, val := range map[string][]string(t.queryParams) {
		switch {
		case len(val) != 1:
			continue
		case key == "nodeId":
			reqBody.ImageReference.NodeID = val[0]
		case len(val) == 1:
			tp[key] = val[0]
		}
	}
	reqBody.ImageReference.Parameters = tp

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "failed forming request body for batch materialization")
	}

	res, err := t.client.Post(urls.batchURL(), "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.Wrap(err, "failed posting batch materialization request")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, ResponseToError(res.Body, fmt.Sprintf("failed posting batch materialization request, HTTP Status: %s", res.Status))
	}

	// Decode the response body.
	resBody := BatchResponse{}
	if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
		return nil, errors.Wrap(err, "batch materialization response failed to decode as json")
	}
	return &resBody, nil
}

// Realize downloads all the tiles from RDA described by the template and its parameters to tileDir.
func (t *Template) Realize(ctx context.Context, tileDir string) ([]TileInfo, error) {
	if err := os.MkdirAll(tileDir, 0775); err != nil {
		return nil, errors.Wrap(err, "couldn't make directory to realize tiles into")
	}

	return t.realize(ctx, tileDir)
}

func (t *Template) realize(ctx context.Context, tileDir string) ([]TileInfo, error) {
	wg := sync.WaitGroup{}
	jobsIn := make(chan realizeJob)
	jobsOut := make(chan realizeJob)

	// Spin up some workers. Note these workers will only shut
	// down once jobsIn is closed and jobsOut is drained.
	for i := 0; i < t.numParallel; i++ {
		wg.Add(1)
		go func(jobsIn <-chan realizeJob, jobsOut chan<- realizeJob) {
			defer wg.Done()
			for job := range jobsIn {
				t.processJob(ctx, job, jobsOut)
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

		for x := t.window.MinTileX; x <= t.window.MaxTileX; x++ {
			for y := t.window.MinTileY; y <= t.window.MaxTileY; y++ {
				rj := realizeJob{
					filePath: filepath.Join(tileDir, fmt.Sprintf("tile_%d_%d.tif", x, y)),
					xTile:    x,
					yTile:    y,
				}

				// Note that if the rj.err is set, we expect it to be handled by the consumer.
				rj.url, rj.err = urls.tileURL(t.templateID, x, y, t.queryParams)
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
	var jobserr *rdaErrors
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

func (t *Template) processJob(ctx context.Context, job realizeJob, jobsOut chan<- realizeJob) {
	// Note we always send our input jobs to the output channel, adding an error to job if one occurred.
	defer func() { jobsOut <- job }()
	defer t.progressFunc()

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
	req.Header.Set("Accept", "image/tiff")
	req = req.WithContext(ctx)

	res, err := t.client.Do(req)
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

// TileInfo holds information about rda tiles that are local on disk.
type TileInfo struct {
	// FilePath is where this tile is located on disk.
	FilePath string

	// XTile is the x coordinate of this tile in reference to the TileWindow it came from.
	XTile int

	// YTile is the y coordinate of this tile in reference to the TileWindow it came from.
	YTile int
}

type rdaErrors struct {
	errors []error
}

func (r *rdaErrors) addError(err error) *rdaErrors {
	// Don't bother reporting context cancellation as an error.
	if errors.Cause(err).Error() == "context canceled" {
		return r
	}

	if r == nil {
		return &rdaErrors{errors: []error{err}}
	}
	r.errors = append(r.errors, err)
	return r
}

func (r *rdaErrors) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d error(s) during realization:\n", len(r.errors))
	for i, err := range r.errors {
		fmt.Fprintf(&sb, "\terror %d: %v\n", i+1, err)
	}
	return sb.String()
}

type realizeJob struct {
	url      string
	filePath string
	xTile    int
	yTile    int
	err      error
}
