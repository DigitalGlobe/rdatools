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
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// Metadata holds the various pieces of information returned by RDA's metadata endpoint.
type Metadata struct {
	ImageMetadata       ImageMetadata
	ImageGeoreferencing ImageGeoreferencing
}

// GraphMetadata returns Metadata for the provided RDA graphID and nodeID.
func GraphMetadata(graphID, nodeID string, client Client) (*Metadata, error) {
	ep := fmt.Sprintf(graphMetadataEnpoint, graphID, nodeID)
	return fetchMetadata(ep, client)
}

// TemplateMetadata returns Metadata for the provided RDA template. queryParams can be nil.
func TemplateMetadata(templateID string, client Client, qp url.Values) (*Metadata, error) {
	u, err := url.Parse(fmt.Sprintf(templateMetadataEnpoint, templateID))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't parse template metadata endpoint")
	}
	u.RawQuery = qp.Encode()
	ep := u.String()

	return fetchMetadata(ep, client)
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

func fetchMetadata(endpoint string, client Client) (*Metadata, error) {
	res, err := client.Get(endpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to form GET for fetching metadata")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, ResponseToError(res.Body, fmt.Sprintf("failed fetching metadata from %s, HTTP Status: %s", endpoint, res.Status))
	}

	md := Metadata{}
	err = json.NewDecoder(res.Body).Decode(&md)
	return &md, err
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
	if (xOff+xSize < 0) || (yOff+ySize < 0) || (xOff > m.ImageMetadata.ImageWidth) || (yOff > m.ImageMetadata.ImageHeight) {
		return nil, errors.Errorf("requested window (%d,%d) - (%d,%d) not contained in image window (%d,%d) - (%d,%d)",
			xOff, yOff, xOff+xSize, yOff+ySize,
			0, 0, m.ImageMetadata.ImageWidth, m.ImageMetadata.ImageHeight)
	}

	invTileGT, err := m.TileGeoreferencing().Invert()
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
func (m *Metadata) TileGeoreferencing() *ImageGeoreferencing {
	tileGT := m.ImageGeoreferencing
	xsize, ysize := float64(m.ImageMetadata.TileXSize), float64(m.ImageMetadata.TileYSize)
	tileGT.ScaleX *= xsize
	tileGT.ShearX *= ysize
	tileGT.ScaleY *= ysize
	tileGT.ShearY *= xsize
	return &tileGT
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
}

// TileWindow contains tile specific metadata.
type TileWindow struct {
	NumXTiles int
	NumYTiles int
	MinTileX  int
	MinTileY  int
	MaxTileX  int
	MaxTileY  int
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
		ScaleX:                     gt.ScaleY * invDet,
		ShearY:                     -gt.ShearY * invDet,

		ShearX: -gt.ShearX * invDet,
		ScaleY: gt.ScaleX * invDet,

		TranslateX: (gt.ShearX*gt.TranslateY - gt.TranslateX*gt.ScaleY) * invDet,
		TranslateY: (-gt.ScaleX*gt.TranslateY + gt.TranslateX*gt.ShearY) * invDet,
	}, nil
}
