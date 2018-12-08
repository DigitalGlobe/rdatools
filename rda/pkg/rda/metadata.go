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
	"encoding/json"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"archive/zip"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
)

// Metadata holds the various pieces of information returned by RDA's metadata endpoint.
type Metadata struct {
	ImageMetadata       ImageMetadata
	ImageGeoreferencing ImageGeoreferencing
}

// Error represents an error we've recieved from the RDA backend.
type rdaError struct {
	Msg string `json:"error"`
}

func (err rdaError) Error() string {
	return err.Msg
}

// ResponseToError takes an errant RDA response and tries to parse
// out the response body into an error for reporting.
func ResponseToError(reader io.Reader, msg string) error {
	rdaerr := rdaError{}
	if derr := json.NewDecoder(reader).Decode(&rdaerr); derr != nil || rdaerr.Msg == "" {
		return errors.New(msg)
	}
	return errors.Wrap(rdaerr, msg)
}

// Subset returns a TileWindow holding the tiles that contain the
// pixel space subsets provided.  If the inputs are all 0, we return the
// Metadata's TileWindow, e.g. all the tiles in the image.
func (m *Metadata) Subset(xOff, yOff, xSize, ySize int) (*TileWindow, error) {
	tm := m.ImageMetadata.TileWindow
	if xOff == 0 && yOff == 0 && xSize == 0 && ySize == 0 {
		return &tm, nil
	}
	if xSize < 1 || ySize < 1 {
		return nil, errors.Errorf("(xSize, ySize) = (%d, %d), but must be positive", xSize, ySize)
	}
	if (xOff+xSize <= 0) || (yOff+ySize <= 0) || (xOff >= m.ImageMetadata.ImageWidth) || (yOff >= m.ImageMetadata.ImageHeight) {
		return nil, errors.Errorf("requested window (%d,%d) - (%d,%d) not contained in image window (%d,%d) - (%d,%d)",
			xOff, yOff, xOff+xSize, yOff+ySize,
			0, 0, m.ImageMetadata.ImageWidth, m.ImageMetadata.ImageHeight)
	}

	invTileGT, err := tm.tileGeoTransform.Invert()
	if err != nil {
		return nil, err
	}

	xGeoTL, yGeoTL := m.ImageGeoreferencing.Apply(float64(xOff), float64(yOff))
	xGeoLR, yGeoLR := m.ImageGeoreferencing.Apply(float64(xOff+xSize-1), float64(yOff+ySize-1))

	xTileTL, yTileTL := invTileGT.Apply(xGeoTL, yGeoTL)
	xTileLR, yTileLR := invTileGT.Apply(xGeoLR, yGeoLR)

	tm.MinTileX, tm.MinTileY = int(xTileTL), int(yTileTL)
	tm.MaxTileX, tm.MaxTileY = int(xTileLR), int(yTileLR)

	// Truncate to fit into the window.
	if tm.MinTileX < m.ImageMetadata.MinTileX {
		tm.MinTileX = m.ImageMetadata.MinTileX
	}
	if tm.MaxTileX > m.ImageMetadata.MaxTileX {
		tm.MaxTileX = m.ImageMetadata.MaxTileX
	}
	if tm.MinTileY < m.ImageMetadata.MinTileY {
		tm.MinTileY = m.ImageMetadata.MinTileY
	}
	if tm.MaxTileY > m.ImageMetadata.MaxTileY {
		tm.MaxTileY = m.ImageMetadata.MaxTileY
	}

	tm.NumXTiles, tm.NumYTiles = tm.MaxTileX-tm.MinTileX+1, tm.MaxTileY-tm.MinTileY+1

	return &tm, nil
}

// TileGeoreferencing returns an ImageGeoreferencing but appropriate for for tile coordinates (rather than pixel coordinates).
func (m *Metadata) TileGeoreferencing() ImageGeoreferencing {
	return m.ImageMetadata.tileGeoTransform
}

func (m *Metadata) setTileGeoreferencing() {
	m.ImageMetadata.tileGeoTransform = m.ImageGeoreferencing
	tileGT := &m.ImageMetadata.tileGeoTransform

	xsize, ysize := float64(m.ImageMetadata.TileXSize), float64(m.ImageMetadata.TileYSize)
	tileGT.ScaleX *= xsize
	tileGT.ShearX *= ysize
	tileGT.ScaleY *= ysize
	tileGT.ShearY *= xsize
}

// ImageMetadata holds metadata specific to the image, aka stuff unrelated to the geo aspect of the image.
type ImageMetadata struct {
	ImageWidth  int
	ImageHeight int
	NumBands    int
	MinX        int
	MinY        int
	DataType    string

	TileXSize int
	TileYSize int
	TileWindow

	AcquisitionDate time.Time
	ImageID         string
	TileBucketName  string
}

// TileWindow contains tile specific metadata.
type TileWindow struct {
	NumXTiles        int
	NumYTiles        int
	MinTileX         int
	MinTileY         int
	MaxTileX         int
	MaxTileY         int
	tileGeoTransform ImageGeoreferencing
}

func (t *TileWindow) wkt() string {
	if t == nil || (*t == TileWindow{}) {
		return ""
	}
	return NewWKTBox(t.MinTileX, t.MinTileY, t.NumXTiles, t.NumYTiles, t.tileGeoTransform).String()
}

// ImageGeoreferencing holds a geo transform (an affine transform).
type ImageGeoreferencing struct {
	SpatialReferenceSystemCode string

	TranslateX float64
	ScaleX     float64
	ShearX     float64

	TranslateY float64
	ShearY     float64
	ScaleY     float64
}

// UnmarshalJSON is our custom json unmarshaler to handle when we
// recieved a null geotransform and to set it to something usable as
// GDAL does.  This is helpful for viewing things like 1Bs.
func (gt *ImageGeoreferencing) UnmarshalJSON(b []byte) error {
	tmp := struct {
		SpatialReferenceSystemCode string

		TranslateX float64
		ScaleX     float64
		ShearX     float64

		TranslateY float64
		ShearY     float64
		ScaleY     float64
	}{}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	*gt = tmp

	// Set nonzeo geo transform when no geotransform was provided.
	if (*gt == ImageGeoreferencing{}) {
		gt.ScaleX = 1.0
		gt.ScaleY = -1.0
	}

	return nil
}

// Apply applies the geo transform to the provided pixel coordinate, returning the corresponding geo coordinates (unless you've got an inverted geo transform).
func (gt *ImageGeoreferencing) Apply(xPix, yPix float64) (xGeo, yGeo float64) {
	return gt.TranslateX + gt.ScaleX*xPix + gt.ShearX*yPix, gt.TranslateY + gt.ShearY*xPix + gt.ScaleY*yPix
}

// Return an inverse geo referencing, e.g. it maps geo coordinates to pixel coordinates.
func (gt *ImageGeoreferencing) Invert() (ImageGeoreferencing, error) {
	// Doing it how its done in GDALInvGeoTransform.
	if gt.ShearX == 0.0 && gt.ShearY == 0.0 && gt.ScaleX != 0.0 && gt.ScaleY != 0.0 {
		return gt.easyInvert()
	}
	return gt.hardInvert()
}

func (gt *ImageGeoreferencing) easyInvert() (ImageGeoreferencing, error) {
	// Simplified computation when there is no shear/rotation (which is typical).
	return ImageGeoreferencing{
		SpatialReferenceSystemCode: gt.SpatialReferenceSystemCode,
		TranslateX:                 -gt.TranslateX / gt.ScaleX,
		ScaleX:                     1.0 / gt.ScaleX,
		TranslateY:                 -gt.TranslateY / gt.ScaleY,
		ScaleY:                     1.0 / gt.ScaleY,
	}, nil
}

func (gt *ImageGeoreferencing) hardInvert() (ImageGeoreferencing, error) {
	// The more general case; we assume the third row of the affine matrix is [0 0 1].
	det := gt.ScaleX*gt.ScaleY - gt.ShearX*gt.ShearY
	if math.Abs(det) < 0.000000000000001 {
		return ImageGeoreferencing{}, errors.Errorf("non invertable geo transform = %+v", gt)
	}
	invDet := 1.0 / det

	return ImageGeoreferencing{
		SpatialReferenceSystemCode: gt.SpatialReferenceSystemCode,
		ScaleX: gt.ScaleY * invDet,
		ShearY: -gt.ShearY * invDet,

		ShearX: -gt.ShearX * invDet,
		ScaleY: gt.ScaleX * invDet,

		TranslateX: (gt.ShearX*gt.TranslateY - gt.TranslateX*gt.ScaleY) * invDet,
		TranslateY: (-gt.ScaleX*gt.TranslateY + gt.TranslateX*gt.ShearY) * invDet,
	}, nil
}

// OperatorInfo returns information describing the RDA operators with
// the given name.  If no names are provided, all operators will be
// described.
func OperatorInfo(client *retryablehttp.Client, w io.Writer, opNames ...string) error {
	opInfo := []interface{}{}
	for _, ep := range urls.operatorURL(opNames...) {

		if err := func() error {
			res, err := client.Get(ep)
			if err != nil {
				return errors.Wrapf(err, "failure requesting %s", ep)
			}
			defer res.Body.Close()
			if res.StatusCode != http.StatusOK {
				return errors.Errorf("failed fetching operator info from %s, HTTP Status: %s", ep, res.Status)
			}

			var blob interface{}
			if err := json.NewDecoder(res.Body).Decode(&blob); err != nil {
				return errors.Wrapf(err, "couldn't unmarshal response from %s", ep)
			}
			opInfo = append(opInfo, blob)
			return nil
		}(); err != nil {
			return err
		}
	}

	var err error
	if len(opNames) == 0 { // The response should be a list of all the operators in this case.
		err = json.NewEncoder(w).Encode(opInfo[0])
	} else {
		err = json.NewEncoder(w).Encode(opInfo)
	}
	return errors.Wrap(err, "failed encoding responses describing the given operators")
}

// StripInfo returns information describing the DG catalog id.  If
// zipped is true, we call the endpoint that returns zipped metadata,
// otherwise we stream the expected json response.
func StripInfo(client *retryablehttp.Client, w io.Writer, catalogID string, zipped bool) error {
	ep := urls.stripinfoURL(catalogID, zipped)
	res, err := client.Get(ep)
	if err != nil {
		return errors.Wrapf(err, "failure requesting %s", ep)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.Errorf("failed fetching strip info from %s, HTTP Status: %s", ep, res.Status)
	}

	if !zipped {
		var blob interface{}
		if err := json.NewDecoder(res.Body).Decode(&blob); err != nil {
			return errors.Wrapf(err, "couldn't unmarshal response from %s", ep)
		}
		return errors.Wrapf(json.NewEncoder(w).Encode(blob), "failed writing json response from %s", ep)
	}
	_, err = io.Copy(w, res.Body)
	return errors.Wrapf(err, "failed writing zipped response from %s", ep)
}

// PartMetadata downloads the DG metadata returned by RDA for the
// given catalog id.  Metadata in this case is the "raw" data that the
// DG factory provides, not RDA metadata.
//
// Note that prefix is used to identify in the zip returned from RDA
// which files to extract, e.g. PAN_001 would grab all metadata files
// that start with that string.
func PartMetadata(client *retryablehttp.Client, catalogID, prefix, outDir string) (*RPCs, error) {
	if err := os.MkdirAll(outDir, 0775); err != nil {
		return nil, errors.Wrap(err, "couldn't make directory to write metadata to")
	}

	// Get all the zipped metadata from RDA.
	ep := urls.stripinfoURL(catalogID, true)
	res, err := client.Get(ep)
	if err != nil {
		return nil, errors.Wrapf(err, "failure requesting %s", ep)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("failed fetching strip info from %s, HTTP Status: %s", ep, res.Status)
	}

	// We have to get all the bytes down into a io.ReaderAt to be able to unzip the response body.
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed reading all the bytes from the reader when extracting metadata from a zip")
	}
	br := bytes.NewReader(b)
	zr, err := zip.NewReader(br, int64(br.Len()))
	if err != nil {
		return nil, errors.Wrap(err, "failed creating a zip reader when extracting metadata")
	}

	// Extract just the files we need from the zipped blob.
	var rpcs *RPCs
	for _, finfo := range zr.File {
		if !strings.HasPrefix(finfo.Name, prefix) {
			continue
		}
		f, err := finfo.Open()
		if err != nil {
			return nil, errors.Wrapf(err, "failed opening %q in zip file", finfo.Name)
		}

		file := filepath.Join(outDir, finfo.Name)
		fout, err := os.Create(file)
		if err != nil {
			return nil, errors.Wrapf(err, "failed creating output metadata file %q", file)
		}
		if _, err := io.Copy(fout, f); err != nil {
			fout.Close()
			return nil, errors.Wrapf(err, "failed writing output metadata file %q", file)
		}
		fout.Close()

		if strings.HasSuffix(file, ".XML") {
			fout, err := os.Open(file)
			if err != nil {
				errors.Wrapf(err, "failed opening output metadata file %q", file)
			}
			rpcs, err = RPCsFromReader(fout)
			if err != nil {
				return nil, err
			}
		}
	}

	return rpcs, nil
}

// ImageParts describes the images that compose a DigitalGlobe Catalog ID.
type ImageParts struct {
	CatID       string `json:"catalogIdentifier"`
	CavisImages []ImageMetadata
	PanImages   []ImageMetadata
	SWIRImages  []ImageMetadata
	VNIRImages  []ImageMetadata
}

// PartSummary returns information describing the DG 1B parts stored in RDA.
func PartSummary(client *retryablehttp.Client, catalogID string) (*ImageParts, error) {
	ep := urls.stripinfoURL(catalogID, false)
	res, err := client.Get(ep)
	if err != nil {
		return nil, errors.Wrapf(err, "failure requesting %s", ep)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("failed fetching strip info from %s, HTTP Status: %s", ep, res.Status)
	}

	var parts ImageParts
	if err := json.NewDecoder(res.Body).Decode(&parts); err != nil {
		return nil, errors.Wrapf(err, "couldn't unmarshal response from %s", ep)
	}
	return &parts, nil
}
