package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type rawPackage struct {
	ImportPath string   `json:"ImportPath"`
	Name       string   `json:"Name"`
	Imports    []string `json:"Imports"`
	Standard   bool     `json:"Standard"`
	ForTest    string   `json:"ForTest"`
}

type GoExtractor struct{}

func (e *GoExtractor) Name() string { return "go" }

func (e *GoExtractor) Extract() ([]Node, error) {
	cmd := exec.Command("go", "list", "-json", "-e", "./projection/...")
	cmd.Dir = "."
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list failed: %w", err)
	}

	var pkgs []rawPackage
	dec := json.NewDecoder(strings.NewReader(string(output)))
	for dec.More() {
		var p rawPackage
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("decode go list output: %w", err)
		}
		if p.ForTest != "" {
			continue
		}
		pkgs = append(pkgs, p)
	}

	if len(pkgs) == 0 {
		return nil, nil
	}

	depSet := map[string]bool{}
	for _, p := range pkgs {
		for _, d := range p.Imports {
			depSet[d] = true
		}
	}

	stdlibSet := goClassifyStdlib(depSet)

	var nodes []Node
	for _, p := range pkgs {
		var deps []string
		for _, d := range p.Imports {
			if !stdlibSet[d] {
				deps = append(deps, d)
			}
		}
		nodes = append(nodes, Node{
			ImportPath: p.ImportPath,
			Name:       p.Name,
			DirectDeps: deps,
		})
	}

	return nodes, nil
}

func goClassifyStdlib(depSet map[string]bool) map[string]bool {
	stdlib := map[string]bool{}
	if len(depSet) == 0 {
		return stdlib
	}

	depList := make([]string, 0, len(depSet))
	for d := range depSet {
		depList = append(depList, d)
	}

	args := append([]string{"list", "-json", "-find", "-e"}, depList...)
	cmd := exec.Command("go", args...)
	cmd.Dir = "."
	output, err := cmd.Output()
	if err != nil {
		return stdlib
	}

	dec := json.NewDecoder(strings.NewReader(string(output)))
	for dec.More() {
		var p rawPackage
		if err := dec.Decode(&p); err != nil {
			break
		}
		if p.Standard {
			stdlib[p.ImportPath] = true
		}
	}

	return stdlib
}
