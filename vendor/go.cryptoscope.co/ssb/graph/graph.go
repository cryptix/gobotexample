package graph

import (
	"math"

	"go.cryptoscope.co/ssb"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/path"
	"gonum.org/v1/gonum/graph/simple"
)

type key2node map[[32]byte]graph.Node

type Graph struct {
	simple.WeightedDirectedGraph
	lookup key2node
}

func (g *Graph) getEdge(from, to *ssb.FeedRef) (graph.WeightedEdge, bool) {
	var bfrom [32]byte
	copy(bfrom[:], from.ID)
	nFrom, has := g.lookup[bfrom]
	if !has {
		return nil, false
	}
	var bto [32]byte
	copy(bto[:], to.ID)
	nTo, has := g.lookup[bto]
	if !has {
		return nil, false
	}
	if !g.HasEdgeFromTo(nFrom.ID(), nTo.ID()) {
		return nil, false
	}
	edg := g.Edge(nFrom.ID(), nTo.ID())
	return edg.(graph.WeightedEdge), true
}

func (g *Graph) Follows(from, to *ssb.FeedRef) bool {
	w, has := g.getEdge(from, to)
	if !has {
		return false
	}
	return w.Weight() == 1
}

func (g *Graph) Blocks(from, to *ssb.FeedRef) bool {
	w, has := g.getEdge(from, to)
	if !has {
		return false
	}
	return w.Weight() == math.Inf(1)
}

func (g *Graph) BlockedList(from *ssb.FeedRef) map[[32]byte]bool {
	var bfrom [32]byte
	copy(bfrom[:], from.ID)
	nFrom, has := g.lookup[bfrom]
	if !has {
		return nil
	}
	blocked := make(map[[32]byte]bool)
	edgs := g.From(nFrom.ID())
	for edgs.Next() {
		edg := g.Edge(nFrom.ID(), edgs.Node().ID()).(contactEdge)

		if edg.Weight() == math.Inf(1) {
			ctNode := edg.To().(*contactNode)
			var k [32]byte
			copy(k[:], ctNode.feed.ID)
			blocked[k] = true
		}
	}
	return blocked
}

func (g *Graph) MakeDijkstra(from *ssb.FeedRef) (*Lookup, error) {
	var bfrom [32]byte
	copy(bfrom[:], from.ID)
	nFrom, has := g.lookup[bfrom]
	if !has {
		return nil, &ErrNoSuchFrom{from}
	}
	return &Lookup{
		path.DijkstraFrom(nFrom, g),
		g.lookup,
	}, nil
}
