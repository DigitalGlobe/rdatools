package rda

import (
	"math"

	"github.com/pkg/errors"
)

// Metadata holds the various pieces of information returned by RDA's metadata endpoint.
type Metadata struct {
	ImageMetadata       ImageMetadata
	ImageGeoreferencing ImageGeoreferencing
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
