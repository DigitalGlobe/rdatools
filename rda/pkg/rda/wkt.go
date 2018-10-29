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

import "fmt"

// WKTBox is a Stringer that returns WKT describing the provided bounding box.
type WKTBox struct {
	ULX, ULY, LRX, LRY float64
}

// String returns a WKT representation of WKTBox.
func (b WKTBox) String() string {
	return fmt.Sprintf("POLYGON ((%f %f, %f %f, %f %f, %f %f, %f %f))", b.ULX, b.ULY, b.LRX, b.ULY, b.LRX, b.LRY, b.ULX, b.LRY, b.ULX, b.ULY)
}

// NewWKTBox returns a WKTBox formed from the given source window offsets and geo referencing.
func NewWKTBox(xOff, yOff, xSize, ySize int, gt ImageGeoreferencing) WKTBox {
	b := WKTBox{}
	b.ULX, b.ULY = gt.Apply(float64(xOff), float64(yOff))
	b.LRX, b.LRY = gt.Apply(float64(xOff+xSize), float64(yOff+ySize))
	return b
}
