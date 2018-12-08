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
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// RPCs hold rational polynomial coefficents parsed from DG metadata XML files.
type RPCs struct {
	ERRBIAS         float64
	ERRRAND         float64
	LINEOFFSET      int
	SAMPOFFSET      int
	LATOFFSET       float64
	LONGOFFSET      float64
	HEIGHTOFFSET    int
	LINESCALE       int
	SAMPSCALE       int
	LATSCALE        float64
	LONGSCALE       float64
	HEIGHTSCALE     int
	LINENUMCOEFList struct {
		LINENUMCOEF FloatsAsString
	}
	LINEDENCOEFList struct {
		LINEDENCOEF FloatsAsString
	}
	SAMPNUMCOEFList struct {
		SAMPNUMCOEF FloatsAsString
	}
	SAMPDENCOEFList struct {
		SAMPDENCOEF FloatsAsString
	}
}

// RPCsFromReader parses RPC values from a DG XML metadata file.
func RPCsFromReader(r io.Reader) (*RPCs, error) {
	d := struct {
		XMLName xml.Name `xml:"isd"`
		RPB     struct {
			IMAGE RPCs
		}
	}{}

	if err := xml.NewDecoder(r).Decode(&d); err != nil {
		return nil, errors.Wrap(err, "failed parsing RPCs")
	}
	return &d.RPB.IMAGE, nil
}

// Metadatar can produces VRT metadata to be added when building out metadata in a VRT.
type Metadatar interface {
	ToVRTMetadata() (*VRTMetadata, error)
}

// ToVRTMetadata converts RPCs to VRT appropriate metadata.
func (r *RPCs) ToVRTMetadata() (*VRTMetadata, error) {

	items := []MDI{
		MDI{Key: "HEIGHT_OFF", Value: r.HEIGHTOFFSET},
		MDI{Key: "HEIGHT_SCALE", Value: r.HEIGHTSCALE},
		MDI{Key: "LAT_OFF", Value: r.LATOFFSET},
		MDI{Key: "LAT_SCALE", Value: r.LATSCALE},
		MDI{Key: "LINE_DEN_COEFF", Value: r.LINEDENCOEFList.LINEDENCOEF.String()},
		MDI{Key: "LINE_NUM_COEFF", Value: r.LINENUMCOEFList.LINENUMCOEF.String()},
		MDI{Key: "LINE_OFF", Value: r.LINEOFFSET},
		MDI{Key: "LINE_SCALE", Value: r.LINESCALE},
		MDI{Key: "LONG_OFF", Value: r.LONGOFFSET},
		MDI{Key: "LONG_SCALE", Value: r.LONGSCALE},
		MDI{Key: "SAMP_DEN_COEFF", Value: r.SAMPDENCOEFList.SAMPDENCOEF.String()},
		MDI{Key: "SAMP_NUM_COEFF", Value: r.SAMPNUMCOEFList.SAMPNUMCOEF.String()},
		MDI{Key: "SAMP_OFF", Value: r.SAMPOFFSET},
		MDI{Key: "SAMP_SCALE", Value: r.SAMPSCALE},
	}

	v := VRTMetadata{
		MDI:    items,
		Domain: "RPC",
	}
	return &v, nil
}

// FloatsAsString exists so we can parse arrays of floats stored in an
// XML element as a string into a slice of floats.
type FloatsAsString []float64

// UnmarshalXML is the custom XML parser for FloatsAsString.
func (f *FloatsAsString) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}

	for _, val := range strings.Split(s, " ") {
		v, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		*f = append(*f, v)
	}
	return nil
}

func (f FloatsAsString) String() string {
	var s []string
	for _, val := range f {
		s = append(s, fmt.Sprintf("%+E", val))
	}
	return strings.Join(s, " ")
}
