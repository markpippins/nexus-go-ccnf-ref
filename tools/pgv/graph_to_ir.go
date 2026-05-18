package main

import (
	"crypto/sha256"
	"fmt"
)

func ToIR(g *Graph) IR {
	nodes := make(map[string]IRNode, len(g.Nodes))
	for _, n := range g.Nodes {
		id := n.ImportPath
		nodes[id] = IRNode{
			ID:       id,
			Path:     n.ImportPath,
			Type:     "module",
			ParentID: deriveParent(n.ImportPath),
			Hash:     hashNode(n),
		}
	}
	return IR{Nodes: nodes}
}

func deriveParent(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return ""
}

func hashNode(n Node) string {
	h := sha256.New()
	h.Write([]byte(n.ImportPath))
	h.Write([]byte("\x00"))
	for _, d := range n.DirectDeps {
		h.Write([]byte(d))
		h.Write([]byte("\x00"))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
