package main

import (
	"encoding/xml"
	"fmt"
)

type VRTDataset struct {
	XMLName      xml.Name        `xml:"VRTDataset"`
	RasterXSize  int             `xml:"RasterXSize,attr"`
	RasterYSize  int             `xml:"RasterYSize,attr"`
	SRS          string          `xml:"SRC"`
	GeoTransform GeoTransform    `xml:"GeoTransform"`
	Bands        []VRTRasterBand `xml:"VRTRasterBand"`
}

type GeoTransform [6]float64

// <VRTRasterBand dataType="Byte" band="1">
type VRTRasterBand struct {
	//XMLName  xml.Name `xml:"VRTRasterBand"`
	DataType     string         `xml:"dataType,attr"`
	Band         int            `xml:"band,attr"` // Mask bands would omit empty somehow...
	SimpleSource []SimpleSource `xml:"SimpleSource"`
}

/*      <SimpleSource>
        <SourceFilename relativeToVRT="1" shared="0">bl/130031330.tif</SourceFilename>
        <SourceBand>mask,1</SourceBand>
        <SourceProperties RasterXSize="2387" RasterYSize="2387" DataType="Byte" BlockXSize="256" BlockYSize="256" />
        <SrcRect xOff="0" yOff="0" xSize="2387" ySize="2387" />
        <DstRect xOff="0" yOff="4774" xSize="2387" ySize="2387" />
      </SimpleSource>*/
type SimpleSource struct {
	SourceFilename SourceFilename `xml:"SourceFilename"`
	SourceBand     int            `xml:"SourceBand"`
}

type SourceFilename struct {
	RelativeToVRT bool   `xml:"relativeToVRT,attr"`
	Shared        bool   `xml:"shared,attr"`
	Filename      string `xml:",chardata"`
}

func (g GeoTransform) MarshalText() (text []byte, err error) {
	return []byte(fmt.Sprintf("%.16e, %.16e, %.16e, %.16e, %.16e, %.16e", g[0], g[1], g[2], g[3], g[4], g[5])), nil
}

func NewVRT(m *Metadata) (*VRTDataset, error) {

	gt := [6]float64{
		m.ImageGeoreferencing.TranslateX,
		m.ImageGeoreferencing.ScaleX,
		m.ImageGeoreferencing.ShearX,
		m.ImageGeoreferencing.TranslateY,
		m.ImageGeoreferencing.ShearY,
		m.ImageGeoreferencing.ScaleY,
	}

	newGT := gt
	newGT[0] = gt[0] + gt[1]*float64(m.ImageMetadata.MinTileX*m.ImageMetadata.TileXSize) + gt[2]*float64(m.ImageMetadata.MinTileY*m.ImageMetadata.TileYSize)
	newGT[3] = gt[3] + gt[4]*float64(m.ImageMetadata.MinTileX*m.ImageMetadata.TileXSize) + gt[5]*float64(m.ImageMetadata.MinTileY*m.ImageMetadata.TileYSize)

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
					SourceFilename: SourceFilename{Filename: fmt.Sprintf("%d_%d.tif", x, y), Shared: false, RelativeToVRT: true},
					SourceBand:     b + 1,
				}
				band.SimpleSource = append(band.SimpleSource, ss)
			}
		}
		vrt.Bands = append(vrt.Bands, band)
	}

	return &vrt, nil

}
