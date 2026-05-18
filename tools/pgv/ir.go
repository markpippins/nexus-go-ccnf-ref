package main

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

type Node struct {
	ImportPath string
	Name       string
	DirectDeps []string
}

type Edge struct {
	From string
	To   string
}

type Graph struct {
	Nodes    []Node
	Edges    []Edge
	Metadata IRMetadata
}

type IRMetadata struct {
	Hash string
}

func BuildGraph(nodes []Node) *Graph {
	sorted := make([]Node, len(nodes))
	copy(sorted, nodes)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ImportPath < sorted[j].ImportPath
	})

	for i := range sorted {
		sort.Strings(sorted[i].DirectDeps)
	}

	edgeSet := map[string]struct{}{}
	for _, n := range sorted {
		for _, dep := range n.DirectDeps {
			key := n.ImportPath + "\x00" + dep
			edgeSet[key] = struct{}{}
		}
	}

	edges := make([]Edge, 0, len(edgeSet))
	for k := range edgeSet {
		parts := strings.SplitN(k, "\x00", 2)
		edges = append(edges, Edge{From: parts[0], To: parts[1]})
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})

	return &Graph{
		Nodes: sorted,
		Edges: edges,
		Metadata: IRMetadata{
			Hash: computeIRHash(sorted, edges),
		},
	}
}

func computeIRHash(nodes []Node, edges []Edge) string {
	h := sha256.New()
	for _, n := range nodes {
		h.Write([]byte(n.ImportPath))
		h.Write([]byte("\x00"))
	}
	for _, e := range edges {
		h.Write([]byte(e.From))
		h.Write([]byte("\x00"))
		h.Write([]byte(e.To))
		h.Write([]byte("\x00"))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
