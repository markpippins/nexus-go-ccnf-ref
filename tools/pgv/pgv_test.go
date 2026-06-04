package main

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"
)

// TestExtractorVersions verifies all extractors implement Version().
func TestExtractorVersions(t *testing.T) {
	extractors := []Extractor{
		&GoExtractor{},
		&RustExtractor{Config: RustMappingConfig{
			CrateRoot: "ccnf-verifier",
			Namespace: "rust",
			SrcPath:   "../../../rust/wrp/ccnf-verifier/src",
		}},
		&CommentExtractor{},
	}
	for _, ext := range extractors {
		v := ext.Version()
		if v == "" {
			t.Errorf("extractor %s returned empty version", ext.Name())
		}
	}
}

// TestExtractorSmoke verifies each extractor runs without error.
func TestExtractorSmoke(t *testing.T) {
	goExt := &GoExtractor{}
	rustExt := &RustExtractor{Config: RustMappingConfig{
		CrateRoot: "ccnf-verifier",
		Namespace: "rust",
		SrcPath:   "../../../rust/wrp/ccnf-verifier/src",
	}}

	for _, ext := range []Extractor{goExt, rustExt} {
		nodes, err := ext.Extract()
		if err != nil {
			t.Errorf("extractor %s failed: %v", ext.Name(), err)
		}
		// Non-nil nodes are valid, nil means no packages found (acceptable)
		if nodes != nil && len(nodes) == 0 {
			t.Errorf("extractor %s returned empty node list", ext.Name())
		}
	}
}

var goldenNodes = []Node{
	{ImportPath: "auth/service", Name: "service", DirectDeps: []string{"auth/model", "shared/log"}},
	{ImportPath: "auth/model", Name: "model", DirectDeps: []string{}},
	{ImportPath: "shared/log", Name: "log", DirectDeps: []string{}},
}

func goldenGraphFixture() *Graph {
	return BuildGraph(goldenNodes)
}

func loadGoldenIRSnapshot(t *testing.T) IR {
	t.Helper()
	data, err := os.ReadFile("testdata/golden_ir.json")
	if err != nil {
		t.Fatalf("failed to read golden snapshot: %v", err)
	}
	var ir IR
	if err := json.Unmarshal(data, &ir); err != nil {
		t.Fatalf("failed to unmarshal golden snapshot: %v", err)
	}
	return ir
}

func flattenAndSort(nodes map[string]IRNode) []IRNode {
	keys := make([]string, 0, len(nodes))
	for k := range nodes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sorted := make([]IRNode, len(keys))
	for i, k := range keys {
		sorted[i] = nodes[k]
	}
	return sorted
}

func updateGolden(t *testing.T) {
	t.Helper()
	g := goldenGraphFixture()
	got := ToIR(g)
	data, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal golden IR: %v", err)
	}
	if err := os.WriteFile("testdata/golden_ir.json", data, 0644); err != nil {
		t.Fatalf("failed to write golden snapshot: %v", err)
	}
}

func TestGoldenGraphToIR(t *testing.T) {
	if os.Getenv("PGV_UPDATE_GOLDEN") == "1" {
		updateGolden(t)
		return
	}

	g := goldenGraphFixture()
	got := ToIR(g)
	want := loadGoldenIRSnapshot(t)

	gotList := flattenAndSort(got.Nodes)
	wantList := flattenAndSort(want.Nodes)

	if !reflect.DeepEqual(gotList, wantList) {
		t.Fatalf("IR drift detected:\nGOT:  %+v\nWANT: %+v", gotList, wantList)
	}

	for id := range want.Nodes {
		if got.Nodes[id].Hash != want.Nodes[id].Hash {
			t.Fatalf("hash instability for node %q: got %s, want %s",
				id, got.Nodes[id].Hash, want.Nodes[id].Hash)
		}
	}
}
