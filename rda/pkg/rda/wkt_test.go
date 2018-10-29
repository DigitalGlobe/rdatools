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
	"math"
	"testing"
)

func TestWKTBox(t *testing.T) {
	b := WKTBox{ULX: -116.79, ULY: 37.86, LRX: -116.70, LRY: 37.78}
	wkt := b.String()

	pts := [10]float64{}
	n, err := fmt.Sscanf(wkt, "POLYGON ((%f %f, %f %f, %f %f, %f %f, %f %f))",
		&pts[0], &pts[1], &pts[2], &pts[3], &pts[4], &pts[5], &pts[6], &pts[7], &pts[8], &pts[9])
	if err != nil {
		t.Fatal(wkt, err)
	}
	if n != 10 {
		t.Fatalf("only scanned %d x and y values, but should have scanned 10", n)
	}
	if pts[0] != pts[8] || pts[1] != pts[9] {
		t.Fatalf("First and last points should be the same, but (%f, %f) != (%f, %f)", pts[0], pts[1], pts[8], pts[9])
	}

	iULX, jULY := -1, -1
	for i := 0; i < 10; i += 2 {
		j := i + 1
		if pts[i] == b.ULX && pts[j] == b.ULY {
			iULX, jULY = i, j
			break
		}
	}
	if iULX < 0 || jULY < 0 {
		t.Fatal("failed fo find (ulx, uly) in parsed WKT output")
	}

	iLRX, jLRY := -1, -1
	for i := 0; i < 10; i += 2 {
		j := i + 1
		if pts[i] == b.LRX && pts[j] == b.LRY {
			iLRX, jLRY = i, j
			break
		}
	}
	if iLRX < 0 || jLRY < 0 {
		t.Fatal("failed fo find (lrx, lry) in parsed WKT output")
	}

	if !((iULX/2-iLRX/2 == 2 && jULY/2-jLRY/2 == 2) || (iULX/2-iLRX/2 == -2 && jULY/2-jLRY/2 == -2)) {
		t.Fatal("upper left and bottom right coorninates are not seperated by one point")
	}
}

func TestNewWKTBox(t *testing.T) {
	ulx, uly := 516719.632, 4209495.193
	lrx, lry := 535919.632, 4086615.193
	b := NewWKTBox(0, 0, 1280, 8192, ImageGeoreferencing{TranslateX: 516719.631503311276902, TranslateY: 4209495.193402875214815, ScaleX: 15.0, ScaleY: -15.0})

	if rdx, rdy := relDiff(b.ULX, ulx), relDiff(b.ULY, uly); rdx > 1e-8 || rdy > 1e-8 {
		t.Fatalf("upper left corner differences not within error bounds, (%f, %f) != (%f, %f)", b.ULX, b.ULY, ulx, uly)
	}
	if rdx, rdy := relDiff(b.LRX, lrx), relDiff(b.LRY, lry); rdx > 1e-8 || rdy > 1e-8 {
		t.Fatalf("lower right corner differences not within error bounds, (%f, %f) != (%f, %f)", b.LRX, b.LRY, lrx, lry)
	}
}

func relDiff(x, y float64) float64 {
	return math.Abs((x - y) / y)
}
