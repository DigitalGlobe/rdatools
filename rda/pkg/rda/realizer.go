package rda

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"sync"

	"os"

	"path/filepath"

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
	Client      Client
	numParallel int
}

// Realize will retrieve all the tiles the Realizer knows about. If
// tiles already exist, they are not downloaded again.
//
// graphID and nodeID are the RDA graph and node you are trying to
// realize.  All tiles described in tileWindow will be downloaded, so
// modify this to suite your needs.  outDir is where the tiles will be
// placed, and this directory will be created for you if not present.
// onFinished is called whenever a tile is finished downloading, nil
// can be provided for this argument.
func (r *Realizer) Realize(ctx context.Context, graphID, nodeID string, tileWindow TileWindow, outDir string, onFinished func() int) ([]TileInfo, error) {
	tileURL := fmt.Sprintf(rdaTileEndpoint, graphID, nodeID)
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
	np := r.numParallel
	if r.numParallel < 1 {
		np = 4 * runtime.NumCPU()
	}
	for i := 0; i < np; i++ {
		wg.Add(1)
		go func(jobsIn <-chan realizeJob, jobsOut chan<- realizeJob) {
			defer wg.Done()
			for job := range jobsIn {
				if err := r.processJob(job, jobsOut, onFinished); err != nil {
					log.Printf("%+v\n", err)
				}
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
					url:      fmt.Sprintf(tileURL, x, y),
					filePath: filepath.Join(outDir, fmt.Sprintf("tile_%d_%d.tif", x, y)),
					xTile:    x,
					yTile:    y,
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
	for job := range jobsOut {
		completedTiles = append(completedTiles, TileInfo{FilePath: job.filePath, XTile: job.xTile, YTile: job.yTile})
	}
	return completedTiles, nil
}

// processJob does the actual download of a tile and writing of it to
// disk.  This should be safe to run concurrently.
func (r *Realizer) processJob(job realizeJob, jobsOut chan<- realizeJob, onFinished func() int) error {
	defer onFinished()
	// If tile is already present, don't download it.
	if _, err := os.Stat(job.filePath); !os.IsNotExist(err) {
		jobsOut <- job
		return nil
	}

	// Download the tile.
	res, err := r.Client.Get(job.url)
	if err != nil {
		return errors.Wrapf(err, "failed requesting tile at %s, err: %v", job.url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.Errorf("failed requesting tile at %s, status: %d %s", job.url, res.StatusCode, res.Status)
	}

	f, err := os.Create(job.filePath)
	if err != nil {
		return errors.Wrapf(err, "failed creating file for tile at %s, err: %v", job.url)
	}
	if _, err := io.Copy(f, res.Body); err != nil {
		err = errors.Wrapf(err, "failed copying tile at %s to disk, err: %v", job.url)
		if nerr := f.Close(); nerr != nil {
			err = errors.WithMessagef(err, "failed closing partially downloaded tile at %s, err: %v", job.filePath, nerr)
		}
		if nerr := os.Remove(job.filePath); nerr != nil {
			err = errors.WithMessagef(err, "failed removing file for partially downloaded tile at %s, err: %v", job.filePath, nerr)
		}
		return err
	}
	if err := f.Close(); err != nil {
		err = errors.Wrapf(err, "failed closing file %s for downloaded tile", job.filePath)
		if nerr := os.Remove(job.filePath); nerr != nil {
			err = errors.WithMessagef(err, "failed removing file for downloaded tile at %s, err: %v", job.filePath, nerr)
		}
		return err
	}
	jobsOut <- job
	return nil
}

type realizeJob struct {
	url      string
	filePath string
	xTile    int
	yTile    int
}
