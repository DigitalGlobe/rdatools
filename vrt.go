package main

import (
	"encoding/xml"
	"fmt"
)

type VRTDataset struct {
	XMLName      xml.Name `xml:"VRTDataset"`
	RasterXSize  int      `xml:",attr"`
	RasterYSize  int      `xml:",attr"`
	SRS          string
	GeoTransform GeoTransform
	Bands        []VRTRasterBand `xml:"VRTRasterBand"`
}

type GeoTransform [6]float64

// <VRTRasterBand dataType="Byte" band="1">
type VRTRasterBand struct {
	//XMLName  xml.Name `xml:"VRTRasterBand"`
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

func NewVRT(m *Metadata, tileMap map[string]string) (*VRTDataset, error) {

	gt := [6]float64{
		m.ImageGeoreferencing.TranslateX,
		m.ImageGeoreferencing.ScaleX,
		m.ImageGeoreferencing.ShearX,
		m.ImageGeoreferencing.TranslateY,
		m.ImageGeoreferencing.ShearY,
		m.ImageGeoreferencing.ScaleY,
	}

	newGT := gt
	//newGT[0] = gt[0] + gt[1]*float64(m.ImageMetadata.MinX) + gt[2]*float64(m.ImageMetadata.MinY)
	//newGT[3] = gt[3] + gt[4]*float64(m.ImageMetadata.MinX) + gt[5]*float64(m.ImageMetadata.MinY)

	vrt := VRTDataset{
		RasterXSize:  m.ImageMetadata.TileXSize * m.ImageMetadata.NumXTiles,
		RasterYSize:  m.ImageMetadata.TileYSize * m.ImageMetadata.NumYTiles,
		SRS:          m.ImageGeoreferencing.SpatialReferenceSystemCode,
		GeoTransform: newGT,
		Bands:        make([]VRTRasterBand, 0, m.ImageMetadata.NumBands),
	}

	for b := 0; b < m.ImageMetadata.NumBands; b++ {
		band := VRTRasterBand{
			DataType: m.ImageMetadata.DataType,
			Band:     b + 1,
		}
		for x := m.ImageMetadata.MinTileX; x < m.ImageMetadata.NumXTiles; x++ {
			for y := m.ImageMetadata.MinTileY; y < m.ImageMetadata.NumYTiles; y++ {
				ss := SimpleSource{
					SourceFilename: SourceFilename{Filename: tileMap[fmt.Sprintf("%d/%d", x, y)], Shared: false, RelativeToVRT: true},
					SourceBand:     b + 1,
					SourceProperties: SourceProperties{
						BlockXSize:  m.ImageMetadata.TileXSize,
						BlockYSize:  m.ImageMetadata.TileYSize,
						DataType:    m.ImageMetadata.DataType,
						RasterXSize: m.ImageMetadata.TileXSize,
						RasterYSize: m.ImageMetadata.TileYSize,
					},
					SrcRect: Rect{
						XOff:  0,
						YOff:  0,
						XSize: m.ImageMetadata.TileXSize,
						YSize: m.ImageMetadata.TileYSize,
					},
					DstRect: Rect{
						XOff:  x * m.ImageMetadata.TileXSize,
						YOff:  y * m.ImageMetadata.TileYSize,
						XSize: m.ImageMetadata.TileXSize,
						YSize: m.ImageMetadata.TileYSize,
					},
				}
				band.SimpleSource = append(band.SimpleSource, ss)
			}
		}
		vrt.Bands = append(vrt.Bands, band)
	}

	return &vrt, nil

}
