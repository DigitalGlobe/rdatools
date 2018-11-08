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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

func TestTemplateMetadata(t *testing.T) {

	var testFunc func(r *http.Request)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		testFunc(r)

		md := Metadata{}
		if err := json.NewEncoder(w).Encode(md); err != nil {
			t.Fatal("test server failed to encode response", err)
		}

	}))
	defer ts.Close()
	urls = newEndpoints(ts.URL)

	tests := []struct {
		name     string
		template *Template
		testFunc func(r *http.Request)
	}{
		{"noopts",
			NewTemplate("tID", retryablehttp.NewClient()),
			func(r *http.Request) {
				if !strings.Contains(r.URL.Path, "tID") {
					t.Fatalf("tID should have been in the url %s", r.URL)
				}
			}},
		{"params",
			NewTemplate("tID", retryablehttp.NewClient(), AddParameter("param1", "val1"), AddParameter("param2", "val2")),
			func(r *http.Request) {
				if r.FormValue("param1") != "val1" || r.FormValue("param2") != "val2" {
					t.Fatal("request is missing query parameters param1=val1 and/or param2=val2")
				}
			}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFunc = tc.testFunc
			if _, err := tc.template.Metadata(); err != nil {
				t.Fatal(err)
			}
		})
	}

}

func TestTemplateBatchRealize(t *testing.T) {

	var testFunc func(r *http.Request)

	tw := TileWindow{
		NumXTiles: 1,
		NumYTiles: 1,
		MinTileX:  0,
		MinTileY:  0,
		MaxTileX:  1,
		MaxTileY:  1,
		tileGeoTransform: ImageGeoreferencing{
			ScaleX: 1,
			ScaleY: -1,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		testFunc(r)

		md := BatchResponse{}
		if err := json.NewEncoder(w).Encode(md); err != nil {
			t.Fatal("test server failed to encode response", err)
		}

	}))
	defer ts.Close()
	urls = newEndpoints(ts.URL)

	tests := []struct {
		name     string
		template *Template
		testFunc func(r *http.Request)
	}{
		{"noopts",
			NewTemplate("tID", retryablehttp.NewClient()),
			func(r *http.Request) {
				brExp := BatchRequest{
					ImageReference: ImageReference{TemplateID: "tID"},
				}
				br := BatchRequest{}

				if err := json.NewDecoder(r.Body).Decode(&br); err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(brExp, br); diff != "" {
					t.Fatal(diff)
				}
			}},
		{"with-params",
			NewTemplate("tID", retryablehttp.NewClient(), AddParameter("nodeId", "nID"), AddParameter("param1", "val1"), AddParameter("param2", "val2")),
			func(r *http.Request) {
				brExp := BatchRequest{
					ImageReference: ImageReference{
						TemplateID: "tID",
						NodeID:     "nID",
						Parameters: map[string]string{"param1": "val1", "param2": "val2"},
					},
				}
				br := BatchRequest{}

				if err := json.NewDecoder(r.Body).Decode(&br); err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(brExp, br); diff != "" {
					t.Fatal(diff)
				}
			}},
		{"with-window",
			NewTemplate("tID", retryablehttp.NewClient(), WithWindow(tw)),
			func(r *http.Request) {
				brExp := BatchRequest{
					ImageReference: ImageReference{
						TemplateID: "tID",
					},
					CropGeometryWKT: tw.wkt(),
				}
				br := BatchRequest{}
				if err := json.NewDecoder(r.Body).Decode(&br); err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(brExp, br); diff != "" {
					t.Fatal(diff)
				}
			}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFunc = tc.testFunc
			if _, err := tc.template.BatchRealize(context.Background(), Tif); err != nil {
				t.Fatal(err)
			}
		})
	}
}
