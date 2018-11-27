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
	"os"
	"path/filepath"
	"testing"
)

func loadGraph(file string, t *testing.T) *Graph {
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	g, err := NewGraphFromAPI(f)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestHasCycle(t *testing.T) {
	tests := []struct {
		testFile string
		hasCycle bool
	}{
		{filepath.Join("test-fixtures", "template", "idaho-read.json"), false},
		{filepath.Join("test-fixtures", "template", "dgstrip.json"), false},
		{filepath.Join("test-fixtures", "template", "dgstrip-with-cycle.json"), true},
	}

	for _, tc := range tests {
		g := loadGraph(tc.testFile, t)
		c := g.findCycle()
		if tc.hasCycle && c == nil {
			t.Fatalf("%q should have a cycle", tc.testFile)
		} else if !tc.hasCycle && c != nil {
			t.Fatalf("%q should not have a cycle", tc.testFile)
		}
	}
}
