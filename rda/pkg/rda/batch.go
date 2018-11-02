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
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"encoding/json"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
)

// BatchFormat are the types of output RDA's batch materialization can output.
type BatchFormat int

const (
	// Tif specifies that you'll get cloud optimized geotiff output format.
	Tif BatchFormat = iota

	// TileStream will produce a stream of tiles as an output format.
	TileStream

	// TMS will produce a TMS tile stack for you.
	TMS

	// Vector will produce geojson output; a binary image is required.
	Vector

	// VectorTile will produce mapbox vectortile output; a binary image is required.
	VectorTile
)

func (b BatchFormat) String() string {
	switch b {
	case Tif:
		return "TIF"
	case TileStream:
		return "TILE_STREAM"
	case TMS:
		return "TMS"
	case Vector:
		return "VECTOR"
	case VectorTile:
		return "VECTOR_TILE"
	default:
		return "UNKNOWN"
	}
}

// MarshalText writes BatchFormat in JSON that RDA expects.
func (b BatchFormat) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

// MarshalText reads BatchFormat from byte slices as returned from RDA.
func (b *BatchFormat) UnmarshalText(buf []byte) error {
	val := strings.ToUpper(string(buf))
	switch val {
	case "TIF":
		*b = Tif
	case "TILE_STREAM":
		*b = TileStream
	case "TMS":
		*b = TMS
	case "VECTOR":
		*b = Vector
	case "VECTOR_TILE":
		*b = VectorTile
	default:
		return errors.Errorf("Unknown BatchFormat = %s", buf)
	}
	return nil
}

// BatchRequest is the HTTP body expected by RDA when POSTing a
// batch materialization request.
type BatchRequest struct {
	ImageReference  ImageReference    `json:"imageReference"`
	OutputFormat    BatchFormat       `json:"outputFormat"`
	FormatOptions   map[string]string `json:"formatOptions,omitempty"`
	CallbackURL     string            `json:"callbackUrl,omitempty"`
	CropGeometryWKT string            `json:"cropGeometryWKT,omitempty"`
	AccountID       string            `json:"accountId,omitempty"`
	EmailAddress    string            `json:"emailAddress,omitempty"`
}

// ImageReference hold the portion of RDA's batch materialization POST
// describing the template we're trying to render.
type ImageReference struct {
	TemplateID string            `json:"templateId"`
	NodeID     string            `json:"nodeId,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// BatchResponse is the HTTP body returned by RDA when POSTing a
// batch materialization request.
type BatchResponse struct {
	JobID   string       `json:"jobId"`
	Request BatchRequest `json:"request"`
	Status  BatchStatus  `json:"status"`
}

// BatchStatus holds the status of an RDA batch materialization request.
type BatchStatus struct {
	InternalJobID string        `json:"internalJobId"`
	Status        string        `json:"jobStatus"`
	StartTime     *EpochTime    `json:"startTime"`
	EndTime       *EpochTime    `json:"endTime,omitempty"`
	ElapsedTime   EpochDuration `json:"elapsedTime,omitempty"`
	StatusMessage string        `json:"statusMessage,omitempty"`
}

// EpochTime is a time.Time but able to unmarshal from an epoch representation in millisconds.
type EpochTime time.Time

func (et EpochTime) String() string {
	return time.Time(et).String()
}

// UnmarshalJSON lets us unmarshal a unix time stamped field from RDA as a EpochTime
func (et *EpochTime) UnmarshalJSON(b []byte) (err error) {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	epoch, err := strconv.Atoi(string(b))
	if err != nil {
		return errors.Wrap(err, "couldn't unmarshal epoch time")
	}
	*et = EpochTime(time.Unix(int64(epoch/1e3), 0))
	return nil
}

// MarshalText lets us marshal a unix time stamped field from RDA
func (et EpochTime) MarshalText() ([]byte, error) {
	t := time.Time(et)
	if t.Unix() <= 0 {
		return nil, nil
	}
	return []byte(t.String()), nil
}

// EpochDuration is a time.Time but able to unmarshal from a duration in millisconds.
type EpochDuration time.Duration

// String lets us pretty print our EpochDuration values.
func (et EpochDuration) String() string {
	return time.Duration(et).String()
}

// MarshalText lets us marshal a unix time stamped field from RDA
func (et EpochDuration) MarshalText() ([]byte, error) {
	t := time.Duration(et)
	if t <= 0 {
		return nil, nil
	}
	return []byte(t.String()), nil
}

// UnmarshalJSON lets us unmarshal a unix time stamped field from RDA as a EpochTime
func (et *EpochDuration) UnmarshalJSON(b []byte) (err error) {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	dur, err := strconv.Atoi(string(b))
	if err != nil {
		return errors.Wrap(err, "couldn't unmarshal epoch duration")
	}
	*et = EpochDuration(int64(1e6 * dur))
	return nil
}

type batchStatusResponse struct {
	resp *BatchResponse
	err  error
}

// FetchBatchStatus returns the status of RDA batch materialization jobs.
func FetchBatchStatus(ctx context.Context, client *retryablehttp.Client, jobIDs ...string) ([]*BatchResponse, error) {
	numParallel := 4 * runtime.NumCPU()
	if len(jobIDs) < numParallel {
		numParallel = len(jobIDs)
	}
	jobIDsIn := make(chan string)
	jobsOut := make(chan *batchStatusResponse)

	// Send the job ids along the channel to be processed.
	go func(jobIDsIn chan<- string) {
		defer close(jobIDsIn)
		for _, job := range jobIDs {
			select {
			case jobIDsIn <- job:
			case <-ctx.Done():
				return
			}
		}
	}(jobIDsIn)

	// Start the workers that are processing the job ids.
	wg := sync.WaitGroup{}
	for i := 0; i < numParallel; i++ {
		wg.Add(1)
		go func(jobIDsIn <-chan string, jobsOut chan<- *batchStatusResponse) {
			defer wg.Done()
			for jobID := range jobIDsIn {
				resp, err := batchStatusJob(ctx, client, jobID)
				jobsOut <- &batchStatusResponse{resp: resp, err: err}
			}
		}(jobIDsIn, jobsOut)
	}

	// When the workers wrap up, close the output channel.
	go func() {
		wg.Wait()
		close(jobsOut)
	}()

	// Read all the responses from the processed job ids.
	jobs := []*BatchResponse{}
	var jobserr *rdaErrors
	for jobResp := range jobsOut {
		if jobResp.err != nil {
			jobserr.addError(jobResp.err)
		} else {
			jobs = append(jobs, jobResp.resp)
		}
	}
	if jobserr != nil {
		return nil, jobserr
	}
	return jobs, nil
}

func batchStatusJob(ctx context.Context, client *retryablehttp.Client, jobID string) (*BatchResponse, error) {
	ep := fmt.Sprintf(templateJobEndpoint, jobID)
	req, err := retryablehttp.NewRequest("GET", ep, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed forming request for batch job id %s", ep)
	}
	req = req.WithContext(ctx)

	res, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to form GET for fetching job status")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, ResponseToError(res.Body, fmt.Sprintf("failed fetching job status from %s, HTTP Status: %s", ep, res.Status))
	}

	br := BatchResponse{}
	if err := json.NewDecoder(res.Body).Decode(&br); err != nil {
		return nil, errors.Wrap(err, "batch materialization response failed to decode as json")
	}
	return &br, nil
}
