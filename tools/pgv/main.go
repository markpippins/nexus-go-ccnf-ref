package main

import (
	"fmt"
	"os"
)

func ComputeDiffPair(base, head *Graph) IRDelta {
	return DiffGraphs(base, head)
}

func ValidateAndDiff(g *Graph, cfg *Config) (*ValidationResult, IRDelta) {
	declaredDeps, _ := (&CommentExtractor{}).ExtractDeclared()
	result := ValidateWithDeclared(g, cfg, declaredDeps)
	delta := DiffGraphs(g, g)
	return result, delta
}

func main() {
	cfg := &Config{
		KernelRoots:   []string{"runtime/", "ccnf/", "replay/", "rust/runtime/", "rust/ccnf/"},
		Allowlist:     []string{"runtime/rehydrate/snapshot", "rust/runtime/rehydrate/snapshot"},
		SchemaVersion: IrSchemaVersion,
	}

	goExt := &GoExtractor{}
	rustExt := &RustExtractor{
		Config: RustMappingConfig{
			CrateRoot: "ccnf-verifier",
			Namespace: "rust",
			SrcPath:   "../../../rust/wrp/ccnf-verifier/src",
		},
	}
	comExt := &CommentExtractor{}

	extractors := []Extractor{goExt, rustExt}

	var allNodes []Node
	for _, ext := range extractors {
		nodes, err := ext.Extract()
		if err != nil {
			fmt.Fprintf(os.Stderr, "PGV: extractor %s failed: %v\n", ext.Name(), err)
			os.Exit(1)
		}
		allNodes = append(allNodes, nodes...)
	}

	if len(allNodes) == 0 {
		fmt.Println("PGV: no packages found from any extractor")
		os.Exit(0)
	}

	graph := BuildGraph(allNodes)

	declaredDeps, _ := comExt.ExtractDeclared()

	result := ValidateWithDeclared(graph, cfg, declaredDeps)

	fmt.Printf("PGV: ir_schema_version: %s\n", IrSchemaVersion)
	fmt.Printf("PGV: IR hash: %s\n", result.Hash)
	fmt.Printf("PGV: %d nodes, %d edges (%d extractors)\n", len(graph.Nodes), len(graph.Edges), len(extractors))
	fmt.Printf("PGV: max depth=%d, cycles=%v\n", result.Depth, result.HasCycles)

	hasError := false
	for _, v := range result.Violations {
		hasError = true
		fmt.Printf("PGV: [ERROR] %s: %s\n", v.Code, v.Message)
		if v.Edge != "" {
			fmt.Printf("PGV:        edge: %s\n", v.Edge)
		}
	}

	if hasError {
		fmt.Println("PGV: FAILED (violations found)")
		os.Exit(1)
	}
	fmt.Println("PGV: PASSED")
}
