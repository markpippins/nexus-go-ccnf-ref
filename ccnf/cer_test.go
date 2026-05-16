package ccnf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCERSerializeRoundTrip(t *testing.T) {
	vectorDir := findVectorDir(t)
	files := listVectorFiles(t, vectorDir)

	for _, vf := range files {
		name := strings.TrimSuffix(vf, ".json")
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(vectorDir, vf))
			if err != nil {
				t.Fatal(err)
			}

			var vector map[string]any
			if err := json.Unmarshal(raw, &vector); err != nil {
				t.Fatal(err)
			}

			expected, _ := vector["expected"].(map[string]any)
			errStr, hasError := expected["error"].(string)

			inputSources := []struct {
				name string
				data map[string]any
			}{}

			if inputData, ok := vector["input"].(map[string]any); ok {
				inputSources = append(inputSources, struct {
					name string
					data map[string]any
				}{"a", inputData})
			}
			if inputA, ok := vector["input_a"].(map[string]any); ok {
				inputSources = append(inputSources, struct {
					name string
					data map[string]any
				}{"a", inputA})
			}
			if inputB, ok := vector["input_b"].(map[string]any); ok {
				inputSources = append(inputSources, struct {
					name string
					data map[string]any
				}{"b", inputB})
			}

			if len(inputSources) == 0 {
				t.Fatal("vector missing 'input', 'input_a', or 'input_b'")
			}

			for _, src := range inputSources {
				t.Run("input_"+src.name, func(t *testing.T) {
					inputJSON, err := json.Marshal(src.data)
					if err != nil {
						t.Fatal(err)
					}

					cer, err := Run(inputJSON, CurrentCCNFVersion)
					if err != nil {
						if hasError && errStr != "" {
							t.Skipf("vector %s expected error: %v", name, err)
						}
						t.Fatalf("Run(%s): %v", name, err)
					}
					if cer == nil {
						t.Fatal("expected non-nil CER")
					}

					serialized := SerializeCER(cer)

					rehydrated, err := ParseCER(serialized)
					if err != nil {
						t.Fatalf("ParseCER: %v", err)
					}

					originalJSON := string(serialized)
					rehydratedJSON := string(SerializeCER(rehydrated))

					if originalJSON != rehydratedJSON {
						t.Fatalf("round-trip CER mismatch:\noriginal:\n%s\nrehydrated:\n%s", originalJSON, rehydratedJSON)
					}
				})
			}
		})
	}
}

func TestCERParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "invalid json",
			input:   `{not json`,
			wantErr: "invalid json",
		},
		{
			name:    "null input",
			input:   `null`,
			wantErr: "identity: expected object",
		},
		{
			name:    "empty object",
			input:   `{}`,
			wantErr: "identity: expected object",
		},
		{
			name:    "bad identity type",
			input:   `{"identity": "not an object"}`,
			wantErr: "identity: expected object",
		},
		{
			name:    "bad state_delta type",
			input:   `{"identity": {"entity_key":"k","type":"t","scope":"s"}, "state_delta": "not an array"}`,
			wantErr: "state_delta: expected array",
		},
		{
			name:    "bad state_delta element type",
			input:   `{"identity": {"entity_key":"k","type":"t","scope":"s"}, "state_delta": ["not an object"]}`,
			wantErr: "state_delta[0]: expected object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCER([]byte(tt.input))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCERSerializationRoundTripCustom(t *testing.T) {
	inputs := []map[string]any{
		{
			"event_id":      "test-001",
			"event_version": 1,
			"ccnf_version":  1,
			"system":        "nexus",
			"domain":        "execution",
			"timestamp":     int64(1700000000),
			"actor":         map[string]any{"id": "actor-1"},
			"intent":        map[string]any{"action": "create", "target_type": "node", "target_id": "node:abc"},
			"identity": Identity{
				EntityKey:   "deadbeef",
				Type:        "event",
				Scope:       "executiongraph.v2",
				CollapseKey: stringPtr("node:abc"),
				AliasKeys:   []string{},
			},
			"causality": map[string]any{
				"parent_event_ids": []any{},
				"causal_chain_id":  "",
				"trace_depth":      0,
				"ordered":          true,
			},
			"artifact_refs": []string{"node:abc"},
			"state_delta": []StateDelta{
				{
					ArtifactID: "node:abc",
					BeforeHash: nil,
					AfterHash:  "abcdef",
					Patch:      map[string]any{"status": "created"},
				},
			},
			"payload": map[string]any{
				"type": "structured",
				"data": map[string]any{},
			},
			"compression": map[string]any{
				"strategy":           "full",
				"lossless":           true,
				"compression_version": 1,
			},
			"signature": map[string]any{
				"hash":      "deadbeef",
				"signed_by": nil,
			},
		},
		{
			"event_id":      "test-002",
			"event_version": 1,
			"ccnf_version":  1,
			"system":        "nexus",
			"domain":        "specification",
			"timestamp":     int64(1700000001),
			"actor":         map[string]any{"id": "actor-2"},
			"intent":        map[string]any{"action": "update", "target_type": "", "target_id": ""},
			"identity": Identity{
				EntityKey:   "cafebabe",
				Type:        "event",
				Scope:       "specification.v1",
				CollapseKey: nil,
				AliasKeys:   nil,
			},
			"causality": map[string]any{
				"parent_event_ids": []any{},
				"causal_chain_id":  "",
				"trace_depth":      0,
				"ordered":          true,
			},
			"artifact_refs": []string{},
			"state_delta":   []StateDelta{},
			"payload": map[string]any{
				"type": "structured",
				"data": map[string]any{},
			},
			"compression": map[string]any{
				"strategy":           "full",
				"lossless":           true,
				"compression_version": 1,
			},
			"signature": map[string]any{
				"hash":      "cafebabe",
				"signed_by": nil,
			},
		},
	}

	for _, tt := range inputs {
		cer := &CER{
			EventID:      tt["event_id"].(string),
			EventVersion: tt["event_version"].(int),
			CCNFVersion:  tt["ccnf_version"].(int),
			System:       tt["system"].(string),
			Domain:       tt["domain"].(string),
			Timestamp:    tt["timestamp"].(int64),
			Actor:        tt["actor"].(map[string]any),
			Intent:       tt["intent"].(map[string]any),
			Identity:     tt["identity"].(Identity),
			Causality:    tt["causality"].(map[string]any),
			ArtifactRefs: tt["artifact_refs"].([]string),
			StateDelta:   tt["state_delta"].([]StateDelta),
			Payload:      tt["payload"].(map[string]any),
			Compression:  tt["compression"].(map[string]any),
			Signature:    tt["signature"].(map[string]any),
		}

		name := cer.EventID
		t.Run(name, func(t *testing.T) {
			serialized := SerializeCER(cer)

			rehydrated, err := ParseCER(serialized)
			if err != nil {
				t.Fatalf("ParseCER: %v", err)
			}

			originalJSON := string(serialized)
			rehydratedJSON := string(SerializeCER(rehydrated))

			if originalJSON != rehydratedJSON {
				t.Fatalf("round-trip mismatch:\noriginal:\n%s\nrehydrated:\n%s", originalJSON, rehydratedJSON)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
