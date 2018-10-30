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

const (
	graphMetadataEnpoint = "https://rda.geobigdata.io/v1/metadata/%s/%s/metadata.json"
	graphTileEndpoint    = "https://rda.geobigdata.io/v1/tile/%s/%s/%%d/%%d.tif"

	// TemplateEndpoint returns description of the graph that backs the template.
	TemplateEndpoint         = "https://rda.geobigdata.io/v1/template/%s"
	templateMetadataEndpoint = "https://rda.geobigdata.io/v1/template/%s/metadata"
	templateTileEnpoint      = "https://rda.geobigdata.io/v1/template/%s/tile/%%d/%%d"
	templateBatchEndpoint    = "https://rda.geobigdata.io/v1/template/materialize"
	templateJobEndpoint      = "https://rda.geobigdata.io/v1/template/materialize/status/%s"

	// OperatorEndpoint is where to get information about RDA operators.
	OperatorEndpoint = "https://rda.geobigdata.io/v1/operator"

	// StripInfoEndpoint returns strip level metadata for a given catalog id.
	StripInfoEndpoint = "https://rda.geobigdata.io/v1/stripMetadata/%s"
)
