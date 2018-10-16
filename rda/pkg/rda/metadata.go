package rda

import "fmt"

// Metadata holds the various pieces of information returned by RDA's metadata endpoint.
type Metadata struct {
	ImageMetadata struct {
		NumXTiles   int
		NumYTiles   int
		TileXSize   int
		TileYSize   int
		ImageWidth  int
		ImageHeight int
		NumBands    int
		MinX        int
		MinY        int
		MinTileX    int
		MinTileY    int
		MaxTileX    int
		MaxTileY    int
		DataType    string
	}
	ImageGeoreferencing ImageGeoreferencing
}

// ImageGeoreferencing holds a geo transform (an affine transform).
type ImageGeoreferencing struct {
	SpatialReferenceSystemCode string
	ScaleX                     float64
	ScaleY                     float64
	TranslateX                 float64
	TranslateY                 float64
	ShearX                     float64
	ShearY                     float64
}

// Apply applies the geo transform to the provided pixel coordinate, returning the corresponding geo coordinates.
func (gt *ImageGeoreferencing) Apply(xPix, yPix float64) (xGeo, yGeo float64) {
	return gt.TranslateX + gt.ScaleX*xPix + gt.ShearX*yPix, gt.TranslateY + gt.ScaleY*xPix + gt.ShearY*yPix
}

// Subset returns a new Metadata but holding the tiles that contain the pixel space subsets provided.
func (m *Metadata) Subset(xOff, yOff, xSize, ySize int) (Metadata, error) {
	if xOff == 0 && yOff == 0 && xSize == 0 && ySize == 0 {
		return *m, nil
	}
	if xSize < 1 || ySize < 1 {
		return Metadata{}, fmt.Errorf("(xSize, ySize) = (%d, %d), but must be positive", xSize, ySize)
	}

}
