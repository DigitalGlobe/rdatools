package rda

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"path"
	"reflect"
	"testing"
	"time"

	"sort"

	"github.com/google/go-cmp/cmp"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

func setTime(unixMS int) *EpochTime {
	t := EpochTime(time.Unix(0, int64(unixMS*1e6)))
	return &t
}

func TestParseBatchResponse(t *testing.T) {
	tests := []struct {
		in  []byte
		out BatchResponse
	}{
		{
			in: []byte(`{"jobId":"e08e1dd0-7366-451a-9cb3-d942827aeb96","request":{"imageReference":{"templateId":"DigitalGlobeStrip","nodeId":null,"parameters":{"GSD":"15","bandSelection":"RGB","bands":"MS","catalogId":"103001000EBC3C00","correctionType":"DN","crs":"UTM","draType":"HistogramDRA"}},"outputFormat":"TIF","formatOptions":{},"callbackUrl":null,"cropGeometryWKT":null,"accountId":"b265b97f-30f2-48bd-9bc5-84c7c7eb0e06","emailAddress":"patrick.young@digitalglobe.com"},"status":{"internalJobId":"8f37d137-29b2-4a17-85e8-b13c4fa279c3","jobStatus":"processing","startTime":1540583795477,"endTime":null,"elapsedTime":null,"statusMessage":null}}`),
			out: BatchResponse{
				JobID: "e08e1dd0-7366-451a-9cb3-d942827aeb96",
				Request: BatchRequest{
					ImageReference: ImageReference{
						TemplateID: "DigitalGlobeStrip",
						Parameters: map[string]string{"correctionType": "DN",
							"crs":           "UTM",
							"draType":       "HistogramDRA",
							"GSD":           "15",
							"bandSelection": "RGB",
							"bands":         "MS",
							"catalogId":     "103001000EBC3C00",
						},
					},
					FormatOptions: map[string]string{},
					OutputFormat:  Tif,
					AccountID:     "b265b97f-30f2-48bd-9bc5-84c7c7eb0e06",
					EmailAddress:  "patrick.young@digitalglobe.com",
				},
				Status: BatchStatus{
					InternalJobID: "8f37d137-29b2-4a17-85e8-b13c4fa279c3",
					Status:        "processing",
					StartTime:     setTime(1540583795477),
				},
			},
		},
		{
			in: []byte(`{"jobId":"eaa6de92-4d6f-409f-8522-c5cf66857965","request":{"imageReference":{"templateId":"DigitalGlobeStrip","nodeId":null,"parameters":{"GSD":"15","bandSelection":"RGB","bands":"MS","catalogId":"103001000EBC3C00","correctionType":"DN","crs":"UTM","draType":"HistogramDRA"}},"outputFormat":"TIF","formatOptions":{},"callbackUrl":null,"cropGeometryWKT":null,"accountId":"b265b97f-30f2-48bd-9bc5-84c7c7eb0e06","emailAddress":"patrick.young@digitalglobe.com"},"status":{"internalJobId":"ce858655-19e6-4300-90ee-e46891919a4d","jobStatus":"complete","startTime":1540580617754,"endTime":1540580773435,"elapsedTime":155681,"statusMessage":null}}`),
			out: BatchResponse{
				JobID: "eaa6de92-4d6f-409f-8522-c5cf66857965",
				Request: BatchRequest{
					ImageReference: ImageReference{
						TemplateID: "DigitalGlobeStrip",
						Parameters: map[string]string{"correctionType": "DN",
							"crs":           "UTM",
							"draType":       "HistogramDRA",
							"GSD":           "15",
							"bandSelection": "RGB",
							"bands":         "MS",
							"catalogId":     "103001000EBC3C00",
						},
					},
					OutputFormat:  Tif,
					FormatOptions: map[string]string{},
					AccountID:     "b265b97f-30f2-48bd-9bc5-84c7c7eb0e06",
					EmailAddress:  "patrick.young@digitalglobe.com"},
				Status: BatchStatus{
					InternalJobID: "ce858655-19e6-4300-90ee-e46891919a4d",
					Status:        "complete",
					StartTime:     setTime(1540580617754),
					EndTime:       setTime(1540580773435),
					ElapsedTime:   155681000000,
				},
			},
		},
	}

	for i, tc := range tests {
		resp := BatchResponse{}
		if err := json.Unmarshal([]byte(tc.in), &resp); err != nil {
			t.Fatalf("failed parsing for case %d response, err: %+v", i, err)
		}
		if !reflect.DeepEqual(resp, tc.out) {
			t.Fatalf("expected:\n%+v\ngot:\n%+v", tc.out, resp)
		}
		t.Log(i)
	}
}

func TestUnmarshalBatchFormat(t *testing.T) {
	tests := []struct {
		in  []byte
		out BatchFormat
	}{
		{[]byte("TIF"), Tif},
		{[]byte("TILE_STREAM"), TileStream},
		{[]byte("TMS"), TMS},
		{[]byte("VECTOR"), Vector},
		{[]byte("VECTOR_TILE"), VectorTile},
	}

	for _, tc := range tests {
		var bTest BatchFormat
		if err := bTest.UnmarshalText(tc.in); err != nil {
			t.Fatal(err)
		}
		if bTest != tc.out {
			t.Fatalf("expected %#v, got:%+v", tc.out, bTest)
		}
	}
}

func TestUnmarshalBatchFormatFail(t *testing.T) {
	tests := [][]byte{
		[]byte("NOT-A-FORMAT"),
		nil,
	}

	for _, tc := range tests {
		var bTest BatchFormat
		if err := bTest.UnmarshalText(tc); err == nil {
			t.Fatal(err)
		}
	}
}

func TestFetchBatchStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)

		jobID := path.Base(r.URL.Path)
		if err := json.NewEncoder(w).Encode(BatchResponse{JobID: jobID}); err != nil {
			t.Fatal("test server failed to encode response", err)
		}
	}))
	defer ts.Close()

	// Map endpoints to the test server.
	urls = newEndpoints(ts.URL)

	jobs := []string{}
	for i := 0; i < 1000; i++ {
		jobs = append(jobs, fmt.Sprintf("job-%d", i))
	}

	jobStats, err := FetchBatchStatus(context.Background(), retryablehttp.NewClient(), jobs...)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobStats) != len(jobs) {
		t.Fatalf("expected %d jobs, got %d", len(jobs), len(jobStats))
	}

	outJobs := []string{}
	for _, j := range jobStats {
		outJobs = append(outJobs, j.JobID)
	}

	sort.Strings(jobs)
	sort.Strings(outJobs)
	if diff := cmp.Diff(jobs, outJobs); diff != "" {
		t.Fatal(diff)
	}
}
