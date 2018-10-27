package rda

import (
	"encoding/json"
	"testing"
)

var (
	respBodyNotFinished = `{
  "jobId": "e08e1dd0-7366-451a-9cb3-d942827aeb96",
  "request": {
    "imageReference": {
      "templateId": "DigitalGlobeStrip",
      "nodeId": null,
      "parameters": {
        "GSD": "15",
        "bandSelection": "RGB",
        "bands": "MS",
        "catalogId": "103001000EBC3C00",
        "correctionType": "DN",
        "crs": "UTM",
        "draType": "HistogramDRA"
      }
    },
    "outputFormat": "TIF",
    "formatOptions": {},
    "callbackUrl": null,
    "cropGeometryWKT": null,
    "accountId": "b265b97f-30f2-48bd-9bc5-84c7c7eb0e06",
    "emailAddress": "patrick.young@digitalglobe.com"
  },
  "status": {
    "internalJobId": "8f37d137-29b2-4a17-85e8-b13c4fa279c3",
    "jobStatus": "processing",
    "startTime": 1540583795477,
    "endTime": null,
    "elapsedTime": null,
    "statusMessage": null
  }
}`
	respBodyFinished = `
{
    "jobId": "eaa6de92-4d6f-409f-8522-c5cf66857965",
    "request": {
        "imageReference": {
            "templateId": "DigitalGlobeStrip",
            "nodeId": null,
            "parameters": {
                "GSD": "15",
                "bandSelection": "RGB",
                "bands": "MS",
                "catalogId": "103001000EBC3C00",
                "correctionType": "DN",
                "crs": "UTM",
                "draType": "HistogramDRA"
            }
        },
        "outputFormat": "TIF",
        "formatOptions": {},
        "callbackUrl": null,
        "cropGeometryWKT": null,
        "accountId": "b265b97f-30f2-48bd-9bc5-84c7c7eb0e06",
        "emailAddress": "patrick.young@digitalglobe.com"
    },
    "status": {
        "internalJobId": "ce858655-19e6-4300-90ee-e46891919a4d",
        "jobStatus": "complete",
        "startTime": 1540580617754,
        "endTime": 1540580773435,
        "elapsedTime": 155681,
        "statusMessage": null
    }
}`
)

func TestParseBatchResponse(t *testing.T) {
	for i, tc := range [][]byte{
		[]byte(respBodyNotFinished),
		[]byte(respBodyFinished),
	} {
		resp := BatchResponse{}
		if err := json.Unmarshal([]byte(tc), &resp); err != nil {
			t.Fatalf("failed parsing for case %d response, err: %+v", i, err)
		}
	}
}
