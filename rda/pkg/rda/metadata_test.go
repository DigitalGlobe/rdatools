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
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"sort"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

func TestImageGeoreferencingApply(t *testing.T) {
	gt := ImageGeoreferencing{
		TranslateX: 10.0,
		ScaleX:     0.1,
		ShearX:     0.0,
		TranslateY: 20.0,
		ShearY:     0.0,
		ScaleY:     -1.0,
	}
	xGeo, yGeo := gt.Apply(10.0, 1.0)
	if xGeo != 11.0 || yGeo != 19.0 {
		t.Fatalf("Expected Apply(10.0, 1.0) = (11.0, 19.0), got (%f, %f)", xGeo, yGeo)
	}
}

func TestImageGeoreferencingInvert(t *testing.T) {
	gt := ImageGeoreferencing{
		TranslateX: 10.0,
		ScaleX:     0.1,
		ShearX:     0.0,
		TranslateY: 20.0,
		ShearY:     0.0,
		ScaleY:     -1.0,
	}

	igt := ImageGeoreferencing{
		TranslateX: -100.0,
		ScaleX:     10.0,
		ShearX:     0.0,
		TranslateY: 20.0,
		ShearY:     0.0,
		ScaleY:     -1.0,
	}

	igtc, err := gt.Invert()
	if err != nil {
		t.Fatalf("failed to invert, err: %+v", err)
	}
	if igtc != igt {
		t.Fatalf("bad inverse, %+v != %+v", igtc, igt)
	}

	igtc, err = gt.hardInvert()
	if err != nil {
		t.Fatalf("failed to hard invert, err: %+v", err)
	}
	if igtc != igt {
		t.Fatalf("bad inverse on hard invert, %+v != %+v", igtc, igt)
	}
}

func TestUnmarshalGeoreferencingJSON(t *testing.T) {
	tests := []struct {
		in  []byte
		out ImageGeoreferencing
	}{
		{[]byte(`{"spatialReferenceSystemCode":"EPSG:32723","scaleX":15,"scaleY":-15,"translateX":333540.0423765521,"translateY":7458901.487530498,"shearX":0,"shearY":0}`),
			ImageGeoreferencing{SpatialReferenceSystemCode: "EPSG:32723", ScaleX: 15, TranslateX: 333540.0423765521, ScaleY: -15, TranslateY: 7458901.487530498}},
		{[]byte("{}"), ImageGeoreferencing{ScaleX: 1, ScaleY: -1}},
	}

	for _, tc := range tests {
		gt := ImageGeoreferencing{}
		if err := json.Unmarshal(tc.in, &gt); err != nil {
			t.Fatal(err)
		}
		if gt != tc.out {
			t.Fatalf("expected %+v, got %+v", tc.out, gt)
		}
	}
}

func loadMetadata(t *testing.T, file string) *Metadata {
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	md := Metadata{}
	if err := json.NewDecoder(f).Decode(&md); err != nil {
		t.Fatal(err)
	}
	md.setTileGeoreferencing()

	return &md
}

func getTileWindow(xoff, yoff, xsize, ysize int) TileWindow {
	return TileWindow{MinTileX: xoff, MinTileY: yoff, NumXTiles: xsize, NumYTiles: ysize, MaxTileX: xoff + xsize - 1, MaxTileY: yoff + ysize - 1}
}

func TestMetadataSubset(t *testing.T) {
	md := loadMetadata(t, "test-fixtures/metadata/1040010038952900.json") // A 1000x1000 image, 10x10 blocks, 100x100 tiles

	// Tests we expect to succeed.
	testsNoError := []struct {
		xoff, yoff, xsize, ysize int
		tw                       TileWindow
	}{
		{0, 0, 0, 0, md.ImageMetadata.TileWindow}, // Window should be the entire scene in special case of all zero inputs
		{-1000, -1000, 3000, 3000, md.ImageMetadata.TileWindow},

		{0, 0, 1, 1, getTileWindow(0, 0, 1, 1)},
		{0, 0, 10, 10, getTileWindow(0, 0, 1, 1)},
		{0, 0, 10, 1, getTileWindow(0, 0, 1, 1)},
		{0, 0, 1, 10, getTileWindow(0, 0, 1, 1)},
		{-10, -10, 11, 11, getTileWindow(0, 0, 1, 1)},

		{10, 10, 10, 10, getTileWindow(1, 1, 1, 1)},
		{10, 9, 1, 1, getTileWindow(1, 0, 1, 1)},
		{9, 10, 1, 1, getTileWindow(0, 1, 1, 1)},

		{990, 990, 10, 10, getTileWindow(99, 99, 1, 1)},
		{989, 990, 10, 10, getTileWindow(98, 99, 2, 1)},
		{990, 989, 10, 10, getTileWindow(99, 98, 1, 2)},
		{990, 990, 100, 100, getTileWindow(99, 99, 1, 1)},
	}

	for _, tc := range testsNoError {
		tw, err := md.Subset(tc.xoff, tc.yoff, tc.xsize, tc.ysize)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(tw, &tc.tw, cmpopts.IgnoreUnexported(TileWindow{})); diff != "" {
			t.Errorf("Unexpected tile window:\n%s", diff)
		}
	}

	// Tests we expect to error.
	testsError := []struct {
		xoff, yoff, xsize, ysize int
	}{
		{-1, -1, 1, 1},
		{0, -1, 1, 1},
		{-1, 0, 1, 1},

		{1000, 1000, 1, 1},
		{1000, 0, 1, 1},
		{0, 1000, 1, 1},

		{10, 10, 0, 1},
		{10, 10, 1, 0},
	}

	for _, tc := range testsError {
		if win, err := md.Subset(tc.xoff, tc.yoff, tc.xsize, tc.ysize); err == nil {
			t.Fatalf("expected (xoff, yoff, xsize, ysize) = (%d, %d, %d, %d) to error, but didn't and got a window %+v", tc.xoff, tc.yoff, tc.xsize, tc.ysize, win)
		}
	}
}

func TestOperatorInfo(t *testing.T) {
	fakeOps := []map[string]string{
		map[string]string{"name": "op1"},
		map[string]string{"name": "op2"},
		map[string]string{"name": "op3"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)

		opName := path.Base(r.URL.Path)
		for _, op := range fakeOps {
			if op["name"] == opName {
				if err := json.NewEncoder(w).Encode(op); err != nil {
					t.Fatal("test server failed to encode response", err)
				}
			}
		}

		if err := json.NewEncoder(w).Encode(fakeOps); err != nil {
			t.Fatal("test server failed to encode response", err)
		}

	}))
	defer ts.Close()

	// Map endpoints to the test server.
	urls = newEndpoints(ts.URL)

	tests := []struct {
		name string
		ops  []string
		out  []map[string]string
	}{
		{"allops", []string{}, fakeOps},
		{"op1", []string{"op1"}, []map[string]string{fakeOps[0]}},
		{"op2-op1", []string{"op2", "op1"}, []map[string]string{fakeOps[0], fakeOps[1]}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, w := io.Pipe()
			defer r.Close()
			go func() {
				defer w.Close()
				if err := OperatorInfo(retryablehttp.NewClient(), w, tc.ops...); err != nil {
					t.Fatal(err)
				}
			}()

			var resp []map[string]string
			if err := json.NewDecoder(r).Decode(&resp); err != nil {
				t.Fatal(err)
			}

			sort.Slice(resp, func(i int, j int) bool { return resp[i]["name"] < resp[j]["name"] })
			sort.Slice(tc.out, func(i int, j int) bool { return tc.out[i]["name"] < tc.out[j]["name"] })

			if diff := cmp.Diff(resp, tc.out); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestStripInfo(t *testing.T) {
	testInfo := map[string]string{
		"name": "catalog-id",
		"type": "ortho",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
		base, catID := path.Split(r.URL.Path)
		if catID == "factoryMetadata" {
			catID = path.Base(base)
			testInfo["zipped"] = "true"
		} else {
			testInfo["zipped"] = "false"
		}

		if catID != testInfo["name"] {
			t.Fatalf("expected %s to be parsed as catid, but got %s", testInfo["name"], catID)
		}

		if err := json.NewEncoder(w).Encode(testInfo); err != nil {
			t.Fatal("test server failed to encode response", err)
		}

	}))
	defer ts.Close()

	urls = newEndpoints(ts.URL)

	t.Run("nozip", func(t *testing.T) {
		r, w := io.Pipe()
		defer r.Close()
		go func() {
			defer w.Close()
			if err := StripInfo(retryablehttp.NewClient(), w, "catalog-id", false); err != nil {
				t.Fatal(err)
			}
		}()

		var resp map[string]string
		if err := json.NewDecoder(r).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		if resp["zipped"] != "false" {
			t.Fatal("should not have recieved a zipped response")
		}
	})

	t.Run("zip", func(t *testing.T) {
		r, w := io.Pipe()
		defer r.Close()
		go func() {
			defer w.Close()
			if err := StripInfo(retryablehttp.NewClient(), w, "catalog-id", true); err != nil {
				t.Fatal(err)
			}
		}()

		var resp map[string]string
		if err := json.NewDecoder(r).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		if resp["zipped"] != "true" {
			t.Fatal("should have recieved a zipped response")
		}
	})

}
