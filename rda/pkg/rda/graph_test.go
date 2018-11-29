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

func loadGraph(file string, t *testing.T) (*Graph, error) {
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	return NewGraphFromAPI(f)
}

func TestNewGraphFromAPI(t *testing.T) {
	tests := []struct {
		testFile string
		hasCycle bool
	}{
		{filepath.Join("test-fixtures", "template", "idaho-read.json"), false},
		{filepath.Join("test-fixtures", "template", "dgstrip.json"), false},
		{filepath.Join("test-fixtures", "template", "dgstrip-with-cycle.json"), true},
	}

	for _, tc := range tests {
		_, err := loadGraph(tc.testFile, t)
		if tc.hasCycle && err == nil {
			t.Fatalf("%q should have a cycle", tc.testFile)
		} else if !tc.hasCycle && err != nil {
			t.Fatalf("%q should not have a cycle", tc.testFile)
		}
	}
}

func customGraph() *Graph {

	nodes := []node{}
	for i := 0; i < 13; i++ {
		nodes = append(nodes, node{})
	}

	edges := make([][]edge, 13)
	edges[0] = []edge{edge{nIdx: 1}, edge{nIdx: 6}}
	edges[2] = []edge{edge{nIdx: 0}, edge{nIdx: 3}}
	edges[3] = []edge{edge{nIdx: 5}}
	edges[5] = []edge{edge{nIdx: 4}}
	edges[6] = []edge{edge{nIdx: 4}, edge{nIdx: 9}}
	edges[7] = []edge{edge{nIdx: 6}}
	edges[8] = []edge{edge{nIdx: 7}}
	edges[9] = []edge{edge{nIdx: 10}, edge{nIdx: 11}, edge{nIdx: 12}}
	edges[11] = []edge{edge{nIdx: 12}}

	g := Graph{
		nodes: nodes,
		edges: edges,
	}
	return &g
}

func TestCustomGraph(t *testing.T) {
	// This test is a custom graph that is complicated enough to be a nontrivial test.
	g := customGraph()
	if n, err := g.findDefaultNode(); err != nil {
		t.Fatal(err)
	} else if n != 12 {
		t.Fatalf("the default node returned should be 12, not %d", n)
	}
}
