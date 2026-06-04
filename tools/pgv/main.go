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
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "diff":
			runDiffCLI(os.Args[2:])
			return
		case "extract":
			runExtractCLI()
			return
		case "diff-files":
			runDiffFilesCLI(os.Args[2:])
			return
		}
	}
	runValidate()
}

func runExtractCLI() {
	g := extractGraph()
	if g == nil {
		return
	}
	data, err := SerializeGraph(g)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PGV: serialize error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func runDiffFilesCLI(args []string) {
	machineMode := false
	humanMode := false
	rest := args
	if len(rest) > 0 && rest[0] == "--machine" {
		machineMode = true
		rest = rest[1:]
	} else if len(rest) > 0 && rest[0] == "--human" {
		humanMode = true
		rest = rest[1:]
	}
	if len(rest) < 2 {
		fmt.Fprintf(os.Stderr, "PGV: usage: pgv diff-files [--machine|--human] <base-graph.json> <head-graph.json>\n")
		os.Exit(2)
	}
	baseData, err := os.ReadFile(rest[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "PGV: read base graph: %v\n", err)
		os.Exit(1)
	}
	headData, err := os.ReadFile(rest[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "PGV: read head graph: %v\n", err)
		os.Exit(1)
	}
	baseGraph, err := DeserializeGraph(baseData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PGV: deserialize base graph: %v\n", err)
		os.Exit(1)
	}
	headGraph, err := DeserializeGraph(headData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PGV: deserialize head graph: %v\n", err)
		os.Exit(1)
	}
	delta := ComputeDiffPair(baseGraph, headGraph)
	if machineMode {
		machineOut := cliMachineOutput{
			CLIVersion:         "pgv.cli.diff.machine.v1",
			IRSchemaVersion:    IrSchemaVersion,
			DeltaSchemaVersion: "pgv.ir.delta.v1",
			Inputs: inputs{
				Base: baseGraph.Metadata.Hash,
				Head: headGraph.Metadata.Hash,
			},
			Result: delta,
		}
		data, err := json.Marshal(machineOut)
		if err != nil {
			fmt.Fprintf(os.Stderr, "PGV: marshal error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
	} else if humanMode {
		fmt.Print(FormatHuman(delta))
	} else {
		fmt.Print(FormatHuman(delta))
	}
}

func runDiffCLI(args []string) {
	machineMode := false
	humanMode := false
	rest := args
	if len(rest) > 0 && rest[0] == "--machine" {
		machineMode = true
		rest = rest[1:]
	} else if len(rest) > 0 && rest[0] == "--human" {
		humanMode = true
		rest = rest[1:]
	}

	if machineMode || humanMode {
		g := extractGraph()
		delta := ComputeDiffPair(g, g)

		if machineMode {
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
		} else {
			fmt.Print(FormatHuman(delta))
		}
		return
	}

	if len(rest) < 2 {
		fmt.Fprintf(os.Stderr, "PGV: usage: pgv diff [--machine|--human] <base> <head>\n")
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
	extractorVersions := map[string]string{}
	for _, ext := range extractors {
		nodes, err := ext.Extract()
		if err != nil {
			fmt.Fprintf(os.Stderr, "PGV: extractor %s failed: %v\n", ext.Name(), err)
			os.Exit(1)
		}
		allNodes = append(allNodes, nodes...)
		extractorVersions[ext.Name()] = ext.Version()
	}

	if len(allNodes) == 0 {
		fmt.Println("PGV: no packages found from any extractor")
		os.Exit(0)
	}

	g := BuildGraph(allNodes)
	g.ExtractorVersions = extractorVersions
	return g
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
