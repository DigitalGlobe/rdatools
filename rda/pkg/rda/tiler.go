package rda

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"sync"

	"os"

	"path/filepath"

	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/oauth2"
)

type Retriever struct {
	client      *retryablehttp.Client
	token       *oauth2.Token
	numParallel int

	metadata Metadata

	tileURL string
	outDir  string
}

func NewRetriever(graphID, nodeID string, metadata Metadata, outDir string, gbdxToken *oauth2.Token) (*Retriever, error) {
	r := &Retriever{
		client:   retryablehttp.NewClient(),
		token:    gbdxToken,
		metadata: metadata,
		tileURL:  fmt.Sprintf("https://rda.geobigdata.io/v1/tile/%s/%s/%%d/%%d.tif", graphID, nodeID),
		outDir:   outDir,
	}
	r.client.Logger = nil

	if r.numParallel == 0 {
		r.numParallel = 4 * runtime.NumCPU()
	}

	return r, os.MkdirAll(r.outDir, 0777)
}

type retrieveJob struct {
	url      string
	filePath string
}

func (r *Retriever) Retrieve(onFinished func() int) map[string]string {
	wg := sync.WaitGroup{}

	jobChan := make(chan retrieveJob)

	if onFinished == nil {
		onFinished = func() int { return 0 }
	}

	// Spin up some workers.
	for i := 0; i < r.numParallel; i++ {
		wg.Add(1)
		go func(jobChan <-chan retrieveJob) {
			defer wg.Done()
			for job := range jobChan {

				func() {
					req, err := retryablehttp.NewRequest("GET", job.url, nil)
					if err != nil {
						log.Fatal("failed building tile request")
					}

					req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.token.AccessToken))
					res, err := r.client.Do(req)
					if err != nil {
						log.Fatalf("failed requesting tile at %s, err: %v\n", job.url, err)
					}
					defer res.Body.Close()

					if res.StatusCode != http.StatusOK {
						log.Fatalf("failed requesting tile at %s, status: %d %s\n", job.url, res.StatusCode, res.Status)
					}

					f, err := os.Create(job.filePath)
					if err != nil {
						log.Fatal("failed opening file on disk")
					}
					defer f.Close()
					if _, err := io.Copy(f, res.Body); err != nil {
						log.Fatal("failed copying tile to disk")
					}

					onFinished()
				}()
			}
		}(jobChan)
	}

	// Launch tile requests.
	tileMap := make(map[string]string)
	for x := r.metadata.ImageMetadata.MinTileX; x < r.metadata.ImageMetadata.NumXTiles; x++ {
		for y := r.metadata.ImageMetadata.MinTileY; y < r.metadata.ImageMetadata.NumYTiles; y++ {
			rj := retrieveJob{
				url:      fmt.Sprintf(r.tileURL, x, y),
				filePath: filepath.Join(r.outDir, fmt.Sprintf("tile_%d_%d.tif", x, y)),
			}
			tileMap[fmt.Sprintf("%d/%d", x, y)] = rj.filePath
			jobChan <- rj
		}
	}
	close(jobChan)

	wg.Wait()
	return tileMap
}
