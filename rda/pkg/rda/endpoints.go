package rda

const (
	graphMetadataEnpoint = "https://rda.geobigdata.io/v1/metadata/%s/%s/metadata.json"
	graphTileEndpoint    = "https://rda.geobigdata.io/v1/tile/%s/%s/%%d/%%d.tif"

	// TemplateEndpoint returns description of the graph that backs the template.
	TemplateEndpoint        = "https://rda.geobigdata.io/v1/template/%s"
	templateMetadataEnpoint = "https://rda.geobigdata.io/v1/template/%s/metadata"
	templateTileEnpoint     = "https://rda.geobigdata.io/v1/template/%s/tile/%%d/%%d"

	// OperatorEndpoint is where to get information about RDA operators.
	OperatorEndpoint = "https://rda.geobigdata.io/v1/operator"

	// StripInfoEndpoint returns strip level metadata for a given catalog id.
	StripInfoEndpoint = "https://rda.geobigdata.io/v1/stripMetadata/%s"
)
