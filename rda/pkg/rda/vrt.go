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
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type VRTDataset struct {
	XMLName      xml.Name        `xml:"VRTDataset"`
	RasterXSize  int             `xml:",attr"`
	RasterYSize  int             `xml:",attr"`
	SRS          string          `xml:",omitempty"`
	GeoTransform *GeoTransform   `xml:",omitempty"`
	Bands        []VRTRasterBand `xml:"VRTRasterBand"`
	Metadata     *VRTMetadata    `xml:",omitempty"`
}

type VRTMetadata struct {
	XMLName xml.Name `xml:"Metadata"`
	Domain  string   `xml:"domain,attr"`
	MDI     []MDI
}

type MDI struct {
	Key   string      `xml:"key,attr"`
	Value interface{} `xml:",chardata"`
}

type GeoTransform [6]float64

type VRTRasterBand struct {
	DataType     string `xml:"dataType,attr"`
	Band         int    `xml:"band,attr,omitempty"`
	SimpleSource []SimpleSource
}

type SimpleSource struct {
	SourceFilename   SourceFilename
	SourceBand       int
	SourceProperties SourceProperties
	SrcRect          Rect
	DstRect          Rect
}

type VRTBool bool

func (b VRTBool) MarshalText() (text []byte, err error) {
	if b {
		return []byte("1"), nil
	}
	return []byte("0"), nil
}

type SourceFilename struct {
	RelativeToVRT VRTBool `xml:"relativeToVRT,attr"`
	Shared        VRTBool `xml:"shared,attr"`
	Filename      string  `xml:",chardata"`
}

type SourceProperties struct {
	RasterXSize int    `xml:",attr"`
	RasterYSize int    `xml:",attr"`
	DataType    string `xml:",attr"`
	BlockXSize  int    `xml:",attr"`
	BlockYSize  int    `xml:",attr"`
}

type Rect struct {
	XOff  int `xml:"xOff,attr"`
	YOff  int `xml:"yOff,attr"`
	XSize int `xml:"xSize,attr"`
	YSize int `xml:"ySize,attr"`
}

func (g GeoTransform) MarshalText() (text []byte, err error) {
	return []byte(fmt.Sprintf("%.16e, %.16e, %.16e, %.16e, %.16e, %.16e", g[0], g[1], g[2], g[3], g[4], g[5])), nil
}

func RDAToGDALType(rda string) (string, error) {
	switch s := strings.ToLower(rda); s {
	case "byte":
		return "Byte", nil
	case "short":
		return "Int16", nil
	case "unsigned_short":
		return "UInt16", nil
	case "integer":
		return "Int32", nil
	case "unsigned_integer":
		return "UInt32", nil
	case "float":
		return "Float32", nil
	case "double":
		return "Float64", nil
	}
	return "", errors.Errorf("RDA type %q has no mapping to a GDAL type", rda)
}

func tileExtents(tiles []TileInfo) (minX, minY, maxX, maxY int) {
	if len(tiles) > 0 {
		minX = tiles[0].XTile
		maxX = minX
		minY = tiles[0].YTile
		maxY = minY
	}
	for _, tile := range tiles {
		if tile.XTile < minX {
			minX = tile.XTile
		}
		if tile.YTile < minY {
			minY = tile.YTile
		}
		if tile.XTile > maxX {
			maxX = tile.XTile
		}
		if tile.YTile > maxY {
			maxY = tile.YTile
		}
	}
	return minX, minY, maxX, maxY
}

// NewVRT returns a populated VRT struct composed of the tiles and metadata given to it.
func NewVRT(m *Metadata, tiles []TileInfo, md Metadatar) (*VRTDataset, error) {
	minXTile, minYTile, maxXTile, maxYTile := tileExtents(tiles)
	numXTiles, numYTiles := maxXTile-minXTile+1, maxYTile-minYTile+1

	// The outer container of the VRT.
	vrt := VRTDataset{
		RasterXSize: m.ImageMetadata.TileXSize * numXTiles,
		RasterYSize: m.ImageMetadata.TileYSize * numYTiles,
		Bands:       make([]VRTRasterBand, 0, m.ImageMetadata.NumBands),
	}

	if md != nil {
		vmd, err := md.ToVRTMetadata()
		if err != nil {
			return nil, err
		}
		vrt.Metadata = vmd
	}

	if m.ImageMetadata.tileGeoTransform.SpatialReferenceSystemCode != "" {
		tx, ty := m.ImageMetadata.tileGeoTransform.Apply(float64(minXTile), float64(minYTile))
		vrt.SRS = m.ImageGeoreferencing.SpatialReferenceSystemCode
		gt := GeoTransform([6]float64{
			tx,
			m.ImageGeoreferencing.ScaleX,
			m.ImageGeoreferencing.ShearX,
			ty,
			m.ImageGeoreferencing.ShearY,
			m.ImageGeoreferencing.ScaleY,
		})
		vrt.GeoTransform = &gt
	} else {
		// Without geoinformation, we assume we're dealing with a 1B and so we want exact pixel dimensions on the VRT.
		vrt.RasterXSize, vrt.RasterYSize = m.ImageMetadata.ImageWidth, m.ImageMetadata.ImageHeight
	}

	// These guys are the same for all the tiles that come back from RDA.
	GDALType, err := RDAToGDALType(m.ImageMetadata.DataType)
	if err != nil {
		return nil, err
	}

	srcProps := SourceProperties{
		BlockXSize:  m.ImageMetadata.TileXSize,
		BlockYSize:  m.ImageMetadata.TileYSize,
		DataType:    GDALType,
		RasterXSize: m.ImageMetadata.TileXSize,
		RasterYSize: m.ImageMetadata.TileYSize,
	}
	srcRect := Rect{
		XOff:  0,
		YOff:  0,
		XSize: m.ImageMetadata.TileXSize,
		YSize: m.ImageMetadata.TileYSize,
	}

	// Build up the vrt bands.
	for b := 0; b < m.ImageMetadata.NumBands; b++ {
		band := VRTRasterBand{
			DataType: GDALType,
			Band:     b + 1,
		}
		for _, tile := range tiles {
			ss := SimpleSource{
				SourceFilename:   SourceFilename{Filename: tile.FilePath, Shared: false, RelativeToVRT: true},
				SourceBand:       b + 1,
				SourceProperties: srcProps,
				SrcRect:          srcRect,
				DstRect: Rect{
					XOff:  (tile.XTile - minXTile) * m.ImageMetadata.TileXSize,
					YOff:  (tile.YTile - minYTile) * m.ImageMetadata.TileYSize,
					XSize: m.ImageMetadata.TileXSize,
					YSize: m.ImageMetadata.TileYSize,
				},
			}
			band.SimpleSource = append(band.SimpleSource, ss)

		}
		vrt.Bands = append(vrt.Bands, band)
	}

	return &vrt, nil
}
