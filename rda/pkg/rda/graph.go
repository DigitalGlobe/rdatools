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
	"encoding/json"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// rdaGraph is the representation that the RDA API uses for describing a graph/template.
type rdaGraph struct {
	DefaultNodeID string
	Edges         []struct {
		ID          string // ID is never needed by us, but the RDA API expects it.
		Index       int    // Index is the order in which this edge is fed into its desitination node; this is an artifact of how JAI works.
		Source      string
		Destination string
	}
	Nodes []struct {
		ID         string
		Operator   string
		Parameters map[string]string
	}
}

// NewGraphFromAPI creates a Graph from the repsonse body provided by the RDA API.
func NewGraphFromAPI(r io.Reader) (*Graph, error) {
	resp := rdaGraph{}
	if err := json.NewDecoder(r).Decode(&resp); err != nil {
		return nil, errors.Wrap(err, "failed decoding RDA API response that describes an rda graph")
	}

	// We have to sort the edges in resp such that edges with the same destination are ordered by their index in the Graph's adjacency list.
	sort.Slice(resp.Edges, func(i, j int) bool {
		if resp.Edges[i].Destination == resp.Edges[j].Destination {
			return resp.Edges[i].Index < resp.Edges[j].Index
		}
		return resp.Edges[i].Destination < resp.Edges[j].Destination
	})

	// Build the graph.
	g := Graph{
		nodes: make([]node, 0, len(resp.Nodes)),
		edges: make([][]edge, len(resp.Nodes)),
	}
	idToIdx := map[string]int{}
	for i, n := range resp.Nodes {
		idToIdx[n.ID] = i
		g.nodes = append(g.nodes, node{n.ID, n.Operator, n.Parameters})
	}
	if len(idToIdx) != len(g.nodes) {
		return nil, errors.Errorf("graph contains %d unique node ids, but has %d nodes", len(g.nodes), len(idToIdx))
	}

	for _, e := range resp.Edges {
		srcID, ok := idToIdx[e.Source]
		if !ok {
			return nil, errors.Errorf("the source %q for edge %+v is not listed as a node", e.Source, e)
		}

		dstID, ok := idToIdx[e.Destination]
		if !ok {
			return nil, errors.Errorf("the destination %q for edge %+v is not listed as a node", e.Destination, e)
		}

		g.edges[srcID] = append(g.edges[srcID], edge{nIdx: dstID, sourceIndex: e.Index})
	}

	// Find a default node, we choose the one with the longest path from its source (this also makes sure g is a DAG).
	defNodeIdx, err := g.findDefaultNode()
	if err != nil {
		return nil, err
	}

	// If given a default node in the body, use that instead.
	var ok bool
	if g.defaultNode, ok = idToIdx[resp.DefaultNodeID]; !ok {
		if resp.DefaultNodeID != "" {
			return nil, errors.Errorf("the default node id %q is not listed as a node", resp.DefaultNodeID)
		}
		g.defaultNode = defNodeIdx
	}

	return &g, nil
}

// Graph encapsulates an RDA template graph.
type Graph struct {
	nodes []node

	// edges is an adjacency list; the first index is the same
	// order as the nodes slice, and the second index says what
	// nodes you can traverse to.
	edges [][]edge

	// defaultNode is the default node to evaluate in an RDA template.
	defaultNode int
}

// node is a node in an RDA graph.
type node struct {
	ID         string
	Operator   string
	Parameters map[string]string
}

type edge struct {
	nIdx        int // nIdx is node index this edge points to.
	sourceIndex int // sourceIndex is needed as RDA cares about the order of edges connecting a destination node.
}

func (g *Graph) numEdges() int {
	nEdges := 0
	for _, eList := range g.edges {
		nEdges += len(eList)
	}
	return nEdges
}

func (g *Graph) toRDA() *rdaGraph {
	rg := rdaGraph{
		DefaultNodeID: g.nodes[g.defaultNode].ID,
	}
	for _, n := range g.nodes {
		rg.Nodes = append(rg.Nodes, struct {
			ID         string
			Operator   string
			Parameters map[string]string
		}{n.ID, n.Operator, n.Parameters})
	}
	eNum := 0
	for srcID, eList := range g.edges {
		for _, e := range eList {
			rg.Edges = append(rg.Edges, struct {
				ID          string
				Index       int
				Source      string
				Destination string
			}{strconv.Itoa(eNum), e.sourceIndex, g.nodes[srcID].ID, g.nodes[e.nIdx].ID})
			eNum++
		}
	}
	return &rg
}

// MarshalJSON lets a Graph marshal itself as a user friendly format.
func (g *Graph) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.toRDA())
}

// findCycle returns a cycle found in g; if there is no cycle, an empty slice is returned.
func (g *Graph) findDefaultNode() (int, error) {
	// Check for cycles (e.g. is it a DAG), get a topological sort.
	c := newCycles(g)
	for nIdx := range g.nodes {
		if c.cycle != nil {
			break
		}
		if c.marked[nIdx] {
			continue
		}

		c.findCycle(nIdx)
	}
	if c.cycle != nil {
		ids := []string{}
		for _, n := range c.cycle {
			ids = append(ids, strconv.Itoa(n))
		}
		return 0, errors.Errorf("the input graph contains a cycle: %s", strings.Join(ids, " -> "))
	}

	// Walking the graph in topological order, cascading distances
	// to connected nodes lets us determine the longest distance
	// between nodes in the graph.  We can then select the
	// terminal node with the maximal distance as our default in
	// the graph.
	ndist := make([]int, len(g.nodes))
	for i := len(g.nodes) - 1; i >= 0; i-- {
		nIdx := c.postTraversal[i]
		for _, e := range g.edges[nIdx] {
			ndist[e.nIdx] = ndist[nIdx] + 1
		}
	}

	maxNode := 0
	maxDist := 0
	for i, dist := range ndist {
		if len(g.edges[i]) != 0 || maxDist >= dist {
			continue
		}
		maxDist = dist
		maxNode = i
	}
	return maxNode, nil
}

// cycles is a helper class for finding cycles in a Graph.
type cycles struct {
	g             *Graph
	onStack       []bool
	marked        []bool
	edgeTo        []int
	cycle         []int
	postTraversal []int
}

func newCycles(g *Graph) *cycles {
	return &cycles{
		g:             g,
		onStack:       make([]bool, len(g.nodes)),
		marked:        make([]bool, len(g.nodes)),
		edgeTo:        make([]int, len(g.nodes)),
		postTraversal: make([]int, 0, len(g.nodes)), // postTraversal records a topological ordering of the DAG.
	}
}

func (c *cycles) findCycle(nIdx int) {
	c.onStack[nIdx] = true
	defer func() { c.onStack[nIdx] = false }()

	c.marked[nIdx] = true
	for _, e := range c.g.edges[nIdx] {
		switch {
		case c.cycle != nil:
			// Bail if we've already found a cycle.
			return
		case !c.marked[e.nIdx]:
			c.edgeTo[e.nIdx] = nIdx
			c.findCycle(e.nIdx)
		case c.onStack[e.nIdx]:
			// We've found a cycle, record what it is by recursing through edgeTo.
			for x := nIdx; x != e.nIdx; x = c.edgeTo[x] {
				c.cycle = append(c.cycle, x)
			}
			c.cycle = append(c.cycle, e.nIdx)
			c.cycle = append(c.cycle, nIdx)
		}
	}
	c.postTraversal = append(c.postTraversal, nIdx)
}
