package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type cliMachineOutput struct {
	CLIVersion        string   `json:"cli_version"`
	IRSchemaVersion   string   `json:"ir_schema_version"`
	DeltaSchemaVersion string  `json:"delta_schema_version"`
	Inputs            inputs   `json:"inputs"`
	Result            IRDelta  `json:"result"`
}

type inputs struct {
	Base string `json:"base"`
	Head string `json:"head"`
}

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
	if len(os.Args) > 1 && os.Args[1] == "diff" {
		runDiffCLI(os.Args[2:])
		return
	}
	runValidate()
}

func runDiffCLI(args []string) {
	machineMode := false
	rest := args
	if len(rest) > 0 && rest[0] == "--machine" {
		machineMode = true
		rest = rest[1:]
	}

	if machineMode {
		g := extractGraph()
		delta := ComputeDiffPair(g, g)
		machineOut := cliMachineOutput{
			CLIVersion:         "pgv.cli.diff.machine.v1",
			IRSchemaVersion:    IrSchemaVersion,
			DeltaSchemaVersion: "pgv.ir.delta.v1",
			Inputs: inputs{
				Base: g.Metadata.Hash,
				Head: g.Metadata.Hash,
			},
			Result: delta,
		}
		data, err := json.Marshal(machineOut)
		if err != nil {
			fmt.Fprintf(os.Stderr, "PGV: marshal error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	if len(rest) < 2 {
		fmt.Fprintf(os.Stderr, "PGV: usage: pgv diff [--machine] <base> <head>\n")
		os.Exit(2)
	}
	fmt.Printf("PGV: diff subcommand (not yet implemented)\n")
	fmt.Printf("PGV: base=%s head=%s\n", rest[0], rest[1])
}

func extractGraph() *Graph {
	goExt := &GoExtractor{}
	rustExt := &RustExtractor{
		Config: RustMappingConfig{
			CrateRoot: "ccnf-verifier",
			Namespace: "rust",
			SrcPath:   "../../../rust/wrp/ccnf-verifier/src",
		},
	}

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

	return BuildGraph(allNodes)
}

func runValidate() {
	graph := extractGraph()
	if graph == nil {
		return
	}

	cfg := &Config{
		KernelRoots:   []string{"runtime/", "ccnf/", "replay/", "rust/runtime/", "rust/ccnf/"},
		Allowlist:     []string{"runtime/rehydrate/snapshot", "rust/runtime/rehydrate/snapshot"},
		SchemaVersion: IrSchemaVersion,
	}

	comExt := &CommentExtractor{}
	declaredDeps, _ := comExt.ExtractDeclared()

	result := ValidateWithDeclared(graph, cfg, declaredDeps)

	fmt.Printf("PGV: ir_schema_version: %s\n", IrSchemaVersion)
	fmt.Printf("PGV: IR hash: %s\n", result.Hash)
	fmt.Printf("PGV: %d nodes, %d edges\n", len(graph.Nodes), len(graph.Edges))
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
