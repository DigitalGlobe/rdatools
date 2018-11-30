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
	"fmt"
	"log"
	"net/url"
	"path"

	"github.com/pkg/errors"
)

// urls is a package global we use to fetch rda urls; we do it this
// way as a private global so that tests can configure the base url in
// one place.
var urls endpoints

func init() {
	urls = newEndpoints("https://rda.geobigdata.io/v1")
}

type endpoints struct {
	u *url.URL

	// operator is the endpoint for getting information on a RDA operator.
	operator string

	// stripinfo is the endpoint for fetching metadata about a given DG catalog id.
	stripinfo string

	// upload is the endpoint for uploading an RDA template
	upload string

	// describe is the endpoint for describing a RDA template
	describe string

	// metadata is the endpoint for fetching template metadata
	metadata string

	// tile is the endpoint for getting RDA tiles
	tile string

	// batch is the endpoint for submitting RDA batch materialization jobs
	batch string

	// job is the endpoint for checking on the status of RDA batch materialization jobs
	job string
}

func newEndpoints(base string) endpoints {
	u, err := url.Parse(base)
	if err != nil {
		log.Fatalf("base RDA url %s must parse successfully", base)
	}

	return endpoints{
		u: u,

		operator:  "operator",
		stripinfo: "stripMetadata/%s",
		upload:    "template",
		describe:  "template/%s",
		metadata:  "template/%s/metadata",
		tile:      "template/%s/tile/%d/%d",
		batch:     "template/materialize",
		job:       "template/materialize/status/%s",
	}
}

func (e *endpoints) formURL(toJoin ...string) string {
	u := *e.u
	u.Path = path.Join(append([]string{u.Path}, toJoin...)...)
	return u.String()
}

func (e *endpoints) stripinfoURL(catalogID string, zipped bool) string {
	ep := fmt.Sprintf(e.stripinfo, catalogID)
	if zipped {
		ep = path.Join(ep, "factoryMetadata")
	}
	return e.formURL(ep)
}

func (e *endpoints) operatorURL(opNames ...string) []string {
	if len(opNames) == 0 {
		return []string{e.formURL(e.operator)}
	}

	eps := []string{}
	for _, opName := range opNames {
		eps = append(eps, e.formURL(e.operator, opName))
	}
	return eps
}

func (e *endpoints) jobURL(jobID string) string {
	return e.formURL(fmt.Sprintf(e.job, jobID))
}

func (e *endpoints) batchURL() string {
	return e.formURL(e.batch)
}

func (e *endpoints) uploadURL() string {
	return e.formURL(e.upload)
}

func (e *endpoints) describeURL(templateID string) string {
	return e.formURL(fmt.Sprintf(e.describe, templateID))
}

func (e *endpoints) metadataURL(templateID string, queryParams url.Values) (string, error) {
	return e.addQueryParams(e.formURL(fmt.Sprintf(e.metadata, templateID)), queryParams)
}

func (e *endpoints) tileURL(templateID string, x, y int, queryParams url.Values) (string, error) {
	return e.addQueryParams(e.formURL(fmt.Sprintf(e.tile, templateID, x, y)), queryParams)
}

func (e *endpoints) addQueryParams(ep string, queryParams url.Values) (string, error) {
	u, err := url.Parse(ep)
	if err != nil {
		return "", errors.Wrapf(err, "failed parsing %s as a URL", ep)
	}
	u.RawQuery = queryParams.Encode()
	return u.String(), nil
}
