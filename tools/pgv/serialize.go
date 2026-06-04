package main

import (
	"encoding/json"
	"fmt"
)

type graphJSON struct {
	Nodes             []Node            `json:"nodes"`
	Edges             []Edge            `json:"edges"`
	SchemaVersion     string            `json:"schema_version"`
	Hash              string            `json:"hash"`
	ExtractorVersions map[string]string `json:"extractor_versions,omitempty"`
}

func SerializeGraph(g *Graph) ([]byte, error) {
	gj := graphJSON{
		Nodes:             g.Nodes,
		Edges:             g.Edges,
		SchemaVersion:     g.Metadata.SchemaVersion,
		Hash:              g.Metadata.Hash,
		ExtractorVersions: g.ExtractorVersions,
	}
	out, err := json.MarshalIndent(gj, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal Graph: %w", err)
	}
	return out, nil
}

func DeserializeGraph(data []byte) (*Graph, error) {
	var gj graphJSON
	if err := json.Unmarshal(data, &gj); err != nil {
		return nil, fmt.Errorf("unmarshal Graph: %w", err)
	}
	return &Graph{
		Nodes: gj.Nodes,
		Edges: gj.Edges,
		Metadata: IRMetadata{
			SchemaVersion: gj.SchemaVersion,
			Hash:          gj.Hash,
		},
		ExtractorVersions: gj.ExtractorVersions,
	}, nil
}

func SerializeIRDelta(delta IRDelta) ([]byte, error) {
	if delta.IRSchemaVersion != IrSchemaVersion {
		return nil, fmt.Errorf("IR schema version mismatch: delta=%s expected=%s", delta.IRSchemaVersion, IrSchemaVersion)
	}
	if delta.SchemaVersion != "pgv.ir.delta.v1" {
		return nil, fmt.Errorf("delta schema version mismatch: got=%s expected=pgv.ir.delta.v1", delta.SchemaVersion)
	}
	out, err := json.Marshal(delta)
	if err != nil {
		return nil, fmt.Errorf("marshal IRDelta: %w", err)
	}
	return out, nil
}
