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
	"testing"
)

func TestImageGeoreferencingApply(t *testing.T) {
	gt := ImageGeoreferencing{
		TranslateX: 10.0,
		ScaleX:     0.1,
		ShearX:     0.0,
		TranslateY: 20.0,
		ShearY:     0.0,
		ScaleY:     -1.0,
	}
	xGeo, yGeo := gt.Apply(10.0, 1.0)
	if xGeo != 11.0 || yGeo != 19.0 {
		t.Fatalf("Expected Apply(10.0, 1.0) = (11.0, 19.0), got (%f, %f)", xGeo, yGeo)
	}
}

func TestImageGeoreferencingInvert(t *testing.T) {
	gt := ImageGeoreferencing{
		TranslateX: 10.0,
		ScaleX:     0.1,
		ShearX:     0.0,
		TranslateY: 20.0,
		ShearY:     0.0,
		ScaleY:     -1.0,
	}

	igt := ImageGeoreferencing{
		TranslateX: -100.0,
		ScaleX:     10.0,
		ShearX:     0.0,
		TranslateY: 20.0,
		ShearY:     0.0,
		ScaleY:     -1.0,
	}

	igtc, err := gt.Invert()
	if err != nil {
		t.Fatalf("failed to invert, err: %+v", err)
	}
	if igtc != igt {
		t.Fatalf("bad inverse, %+v != %+v", igtc, igt)
	}

	igtc, err = gt.hardInvert()
	if err != nil {
		t.Fatalf("failed to hard invert, err: %+v", err)
	}
	if igtc != igt {
		t.Fatalf("bad inverse on hard invert, %+v != %+v", igtc, igt)
	}
}
