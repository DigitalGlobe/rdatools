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

package cmd

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/pkg/errors"
)

type sourceWindow struct {
	xOff, yOff, xSize, ySize int
}

func (s *sourceWindow) String() string {
	return ""
}

func (s *sourceWindow) Set(value string) error {
	vals := strings.SplitN(value, ",", 4)
	if len(vals) != 4 {
		return fmt.Errorf("expected 4 values, but got %d", len(vals))
	}
	var err error
	if s.xOff, err = strconv.Atoi(vals[0]); err != nil {
		return fmt.Errorf("failed setting xOff = %s, err := %+v", vals[0], err)
	}
	if s.yOff, err = strconv.Atoi(vals[1]); err != nil {
		return fmt.Errorf("failed setting yOff = %s, err := %+v", vals[1], err)
	}
	if s.xSize, err = strconv.Atoi(vals[2]); err != nil {
		return fmt.Errorf("failed setting xSize = %s, err := %+v", vals[2], err)
	}
	if s.ySize, err = strconv.Atoi(vals[3]); err != nil {
		return fmt.Errorf("failed setting ySize = %s, err := %+v", vals[3], err)
	}
	return nil
}

func (s *sourceWindow) Type() string {
	return "int,int,int,int"
}

type projectionWindow struct {
	ulx, uly, lrx, lry float64
}

func (p *projectionWindow) String() string {
	return ""
}

func (p *projectionWindow) Set(value string) error {
	vals := strings.SplitN(value, ",", 4)
	if len(vals) != 4 {
		return fmt.Errorf("expected 4 values, but got %d", len(vals))
	}
	var err error
	if p.ulx, err = strconv.ParseFloat(vals[0], 64); err != nil {
		return fmt.Errorf("failed setting ulx = %s, err := %+v", vals[0], err)
	}
	if p.uly, err = strconv.ParseFloat(vals[1], 64); err != nil {
		return fmt.Errorf("failed setting uly = %s, err := %+v", vals[1], err)
	}
	if p.lrx, err = strconv.ParseFloat(vals[2], 64); err != nil {
		return fmt.Errorf("failed setting lrx = %s, err := %+v", vals[2], err)
	}
	if p.lry, err = strconv.ParseFloat(vals[3], 64); err != nil {
		return fmt.Errorf("failed setting lry = %s, err := %+v", vals[3], err)
	}
	return nil
}

func (p *projectionWindow) Type() string {
	return "float,float,float,float"
}

func processSubWindows(srcWin *sourceWindow, projWin *projectionWindow, md *rda.Metadata) (*rda.TileWindow, error) {
	if (*projWin != projectionWindow{} && *srcWin != sourceWindow{}) {
		return nil, errors.New("--projwin and --srcwin cannot be set at the same time")
	}

	// Convert projWin into a srcWin if we were given one.
	if (*projWin != projectionWindow{}) {
		igt, err := md.ImageGeoreferencing.Invert()
		if err != nil {
			return nil, err
		}
		xOff, yOff := igt.Apply(projWin.ulx, projWin.uly)
		srcWin.xOff = int(math.Floor(xOff))
		srcWin.yOff = int(math.Floor(yOff))

		xOffLR, yOffLR := igt.Apply(projWin.lrx, projWin.lry)
		srcWin.xSize = int(math.Ceil(xOffLR - xOff))
		srcWin.ySize = int(math.Ceil(yOffLR - yOff))
	}
	return md.Subset(srcWin.xOff, srcWin.yOff, srcWin.xSize, srcWin.ySize)
}

type coordRefSys string

func (c *coordRefSys) String() string {
	if c == nil || *c == "" {
		return "UTM"
	}
	return string(*c)
}

func (c *coordRefSys) Set(value string) error {
	*c = coordRefSys(strings.ToUpper(value))
	switch {
	case *c == "UTM":
	case strings.HasPrefix(string(*c), "EPSG"):
	default:
		return fmt.Errorf("must either be \"UTM\" or \"EPSG:<EPSG CODE>\"")
	}
	return nil
}

func (c *coordRefSys) Type() string {
	return "string"
}

type bandType string

func (bt *bandType) String() string {
	if bt == nil || *bt == "" {
		return "MS"
	}
	return string(*bt)
}

func (bt *bandType) Set(value string) error {
	v := strings.ToUpper(value)
	switch v {
	case "PAN":
		*bt = "PAN"
	case "MS":
		*bt = "MS"
	case "PANSHARP", "PS":
		*bt = "Pansharp"
	case "SWIR":
		*bt = "SWIR"
	default:
		return errors.Errorf("Unrecogized band type %s", value)
	}
	return nil
}

func (bt *bandType) Type() string {
	return "string"
}

type bandCombo string

func (bc *bandCombo) String() string {
	if bc == nil || *bc == "" {
		return "ALL"
	}
	return string(*bc)
}

func (bc *bandCombo) Set(value string) error {
	v := strings.ToUpper(value)
	switch v {
	case "ALL", "RGB":
		*bc = bandCombo(v)
	case "":
		*bc = "ALL"
	default:
		bandsplit := strings.Split(v, ",")
		for _, b := range bandsplit {
			bint, err := strconv.Atoi(b)
			if err != nil {
				return errors.Errorf("Unrecogized band combo %s", value)
			}
			if bint < 0 {
				return errors.Errorf("band numbers cannot be negative in band combo %s", value)
			}
		}
		*bc = bandCombo(strings.Join(bandsplit, ","))
	}
	return nil
}

func (bc *bandCombo) Type() string {
	return "string"
}
