package rda

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
	ImageGeoreferencing struct {
		SpatialReferenceSystemCode string
		ScaleX                     float64
		ScaleY                     float64
		TranslateX                 float64
		TranslateY                 float64
		ShearX                     float64
		ShearY                     float64
	}
}
