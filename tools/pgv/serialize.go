package main

import (
	"encoding/json"
	"fmt"
)

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
