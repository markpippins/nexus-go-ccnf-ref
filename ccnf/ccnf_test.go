package ccnf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoldenVectors(t *testing.T) {
	vectorDir := findVectorDir(t)
	vectorFiles := listVectorFiles(t, vectorDir)

	for _, vf := range vectorFiles {
		t.Run(vf, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(vectorDir, vf))
			if err != nil {
				t.Fatalf("failed to read vector %s: %v", vf, err)
			}

			var raw map[string]any
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to parse vector %s: %v", vf, err)
			}

			expected := raw["expected"].(map[string]any)
			ccnfVersion := 1
			if v, ok := raw["ccnf_version"].(float64); ok {
				ccnfVersion = int(v)
			}

			errStr, hasError := expected["error"].(string)

			process := func(input map[string]any, suffix string) {
				inputJSON, err := json.Marshal(input)
				if err != nil {
					t.Fatalf("failed to marshal input: %v", err)
				}

				cer, err := Run(inputJSON, ccnfVersion)

				if hasError && errStr != "" {
					if err == nil {
						t.Fatalf("expected error %s but got none", errStr)
					}
					if !strings.Contains(err.Error(), errStr) {
						t.Fatalf("expected error containing %q, got %q", errStr, err.Error())
					}
					return
				}

				if err != nil {
					if suffix == "_v2" {
						return
					}
					t.Fatalf("unexpected error: %v", err)
				}
				if cer == nil {
					t.Fatal("expected non-nil CER")
				}

				entityKeyField := "entity_key" + suffix
				canonHashField := "canonical_hash" + suffix

				if v, ok := expected[entityKeyField].(string); ok && v != "" {
					if cer.Identity.EntityKey != v {
						t.Errorf("%s mismatch:\n  want: %s\n  got:  %s", entityKeyField, v, cer.Identity.EntityKey)
					}
				}

				if v, ok := expected[canonHashField].(string); ok && v != "" {
					actualHash := ComputeHash(cer)
					if actualHash != v {
						t.Errorf("%s mismatch:\n  want: %s\n  got:  %s", canonHashField, v, actualHash)
					}
				}

				if suffix == "" {
					if cerObj, ok := expected["cer"].(map[string]any); ok {
						compareCERMap(t, cerObj, cer)
					}
				}
			}

			if input, ok := raw["input"].(map[string]any); ok && input != nil {
				process(input, "")

				if _, ok := expected["entity_key_v1"]; ok {
					process(raw["input"].(map[string]any), "_v1")
				}
				if _, ok := expected["entity_key_v2"]; ok {
					if _, ok := raw["input"]; ok {
						inputCopy := deepCopyMap(raw["input"].(map[string]any))
						inputCopy["ccnf_version"] = float64(2)
						inputJSON, _ := json.Marshal(inputCopy)
						_, err := Run(inputJSON, 2)
						if err == nil {
							t.Error("expected error for v2 but got none")
						}
					}
				}
			}

			if inputA, ok := raw["input_a"].(map[string]any); ok && inputA != nil {
				process(raw["input_a"].(map[string]any), "_a")
			}
			if inputB, ok := raw["input_b"].(map[string]any); ok && inputB != nil {
				process(raw["input_b"].(map[string]any), "_b")
			}
			if diffInput, ok := raw["different_input"].(map[string]any); ok && diffInput != nil {
				process(raw["different_input"].(map[string]any), "_b")
			}

			if assertNE, ok := expected["assert_not_equal"].(bool); ok && assertNE {
				if ekA, ok := expected["entity_key_a"].(string); ok {
					if ekB, ok := expected["entity_key_b"].(string); ok {
						if ekA == ekB {
							t.Errorf("assert_not_equal violated: entity_key_a == entity_key_b == %s", ekA)
						}
					}
				}
				if ekV1, ok := expected["entity_key_v1"].(string); ok {
					if ekV2, ok := expected["entity_key_v2"].(string); ok {
						if ekV1 == ekV2 {
							t.Errorf("assert_not_equal violated: entity_key_v1 == entity_key_v2 == %s", ekV1)
						}
					}
				}
			}
		})
	}
}

func compareCERMap(t *testing.T, expected map[string]any, cer *CER) {
	if v, ok := expected["event_id"].(string); ok && v != "" && v != cer.EventID {
		t.Errorf("cer.event_id: want %q, got %q", v, cer.EventID)
	}
	if v, ok := expected["event_version"].(float64); ok && v != 0 && int(v) != cer.EventVersion {
		t.Errorf("cer.event_version: want %d, got %d", int(v), cer.EventVersion)
	}
	if v, ok := expected["ccnf_version"].(float64); ok && v != 0 && int(v) != cer.CCNFVersion {
		t.Errorf("cer.ccnf_version: want %d, got %d", int(v), cer.CCNFVersion)
	}
	if v, ok := expected["domain"].(string); ok && v != "" && v != cer.Domain {
		t.Errorf("cer.domain: want %q, got %q", v, cer.Domain)
	}
	if v, ok := expected["timestamp"].(float64); ok && v != 0 && int64(v) != cer.Timestamp {
		t.Errorf("cer.timestamp: want %d, got %d", int64(v), cer.Timestamp)
	}
}

func findVectorDir(t *testing.T) string {
	wd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(wd, "..", "vectors", "v1"),
		filepath.Join(wd, "vectors", "v1"),
		"../vectors/v1",
		"vectors/v1",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	t.Fatal("could not find vectors/v1 directory")
	return ""
}

func listVectorFiles(t *testing.T, dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read vector directory %s: %v", dir, err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") && e.Name() != "expected-hashes.json" {
			files = append(files, e.Name())
		}
	}
	return files
}

func TestRunHappyPath(t *testing.T) {
	input := `{
		"actor": {"type": "system", "id": "test"},
		"intent": {"action": "create", "target_type": "node", "target_id": "test:001"},
		"payload": {"data": {}},
		"domain": "execution",
		"event_id": "test-001",
		"timestamp": 1713225600,
		"causality": {"parent_event_ids": [], "causal_chain_id": "chain-test", "trace_depth": 0}
	}`

	cer, err := Run([]byte(input), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cer == nil {
		t.Fatal("expected non-nil CER")
	}
	if cer.Identity.EntityKey == "" {
		t.Error("expected non-empty entity_key")
	}
	if cer.Intent == nil {
		t.Error("expected non-nil intent")
	}
	if cer.Signature == nil {
		t.Error("expected non-nil signature")
	}
	if cer.Signature["hash"] == "" {
		t.Error("expected non-empty signature hash")
	}
}

func TestRunVersionMismatch(t *testing.T) {
	input := `{"actor": {"type": "test", "id": "t"}, "intent": {"action": "create", "target_type": "n", "target_id": "t:1"}, "domain": "test", "event_id": "t"}`
	_, err := Run([]byte(input), 99)
	if err == nil {
		t.Fatal("expected error for version mismatch")
	}
}

func TestSerializerDeterminism(t *testing.T) {
	m1 := map[string]any{
		"z": "last",
		"a": "first",
		"m": "middle",
		"nested": map[string]any{
			"b": "two",
			"a": "one",
		},
	}
	m2 := map[string]any{
		"nested": map[string]any{
			"a": "one",
			"b": "two",
		},
		"m": "middle",
		"z": "last",
		"a": "first",
	}

	j1 := string(CanonicalJSON(m1))
	j2 := string(CanonicalJSON(m2))
	if j1 != j2 {
		t.Errorf("determinism violation:\n  %s\n  %s", j1, j2)
	}
}
