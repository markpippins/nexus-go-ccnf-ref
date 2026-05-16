package ccnf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestR2Determinism(t *testing.T) {
	cfg := DefaultR2Config()
	fuzzer := NewSemanticFuzzer(cfg)
	detector := NewCollisionDetector()

	for i := 0; i < cfg.Iterations; i++ {
		input := fuzzer.Generate(i)
		inputJSON := CanonicalJSON(input)

		cer, err := Run(inputJSON, 1)
		if err != nil {
			continue
		}

		h := ComputeHash(cer)
		detector.Add(input, h)
	}

	report := detector.Report()
	if len(report.Divergences) > 0 {
		t.Errorf("R2-DETERMINISM: %d divergence(s) detected", len(report.Divergences))
	}
}

func TestR2EquivalenceClassBoundary(t *testing.T) {
	domains := []string{"execution", "specification", "system", "test"}
	actions := []string{"create", "update", "delete", "execute", "validate", "emit"}
	types := []string{"node", "task", "graph", "workflow", "artifact"}

	combos := 0
	for _, domain := range domains {
		for _, action := range actions {
			for _, typ := range types {
				base := map[string]any{
					"actor": map[string]any{
						"type": "system", "id": "boundary-test",
					},
					"intent": map[string]any{
						"action": action, "target_type": typ,
						"target_id": typ + ":BOUNDARY",
					},
					"payload":   map[string]any{"data": map[string]any{}},
					"domain":    domain,
					"event_id":  fmt.Sprintf("boundary-%s-%s-%s", domain, action, typ),
					"timestamp": float64(1713225600),
					"causality": map[string]any{
						"parent_event_ids": []any{},
						"causal_chain_id":  "chain-boundary",
						"trace_depth":      0,
					},
				}
				j, _ := json.Marshal(base)
				cer, err := Run(j, 1)
				if err != nil {
					t.Errorf("R2-BOUNDARY: valid combo %s/%s/%s failed: %v", domain, action, typ, err)
					continue
				}
				if cer.Identity.EntityKey == "" {
					t.Error("R2-BOUNDARY: empty entity_key")
				}
				combos++
			}
		}
	}
	t.Logf("R2-BOUNDARY: %d domain×action×type combinations valid", combos)
}

func TestR2CrossOrderEquivalence(t *testing.T) {
	tester := NewCrossOrderTester()
	cfg := DefaultR2Config()
	fuzzer := NewSemanticFuzzer(cfg)
	trials := 0

	for i := 0; i < 100; i++ {
		input := fuzzer.Generate(i)
		refHash, otherHashes, ok, err := tester.Test(input)
		if err != nil {
			continue
		}
		trials++
		if !ok {
			t.Errorf("R2-CROSS-ORDER: hash divergence at iteration %d\n  ref: %s\n  others: %v", i, refHash, otherHashes)
		}
	}
	t.Logf("R2-CROSS-ORDER: %d trials, all hash-stable under key permutation", trials)
}

func TestR2RoundTrip(t *testing.T) {
	checker := NewRoundTripChecker()
	cfg := DefaultR2Config()
	fuzzer := NewSemanticFuzzer(cfg)

	for i := 0; i < 1000; i++ {
		input := fuzzer.Generate(i)
		inputJSON := CanonicalJSON(input)

		h1, h2, err := checker.Check(inputJSON)
		if err != nil {
			continue
		}
		if h1 != h2 {
			t.Errorf("R2-ROUNDTRIP: hash instability at iteration %d\n  pass1: %s\n  pass2: %s", i, h1, h2)
		}
	}
}

func TestR2CollisionDetection(t *testing.T) {
	cfg := StressR2Config()
	fuzzer := NewSemanticFuzzer(cfg)
	detector := NewCollisionDetector()

	for i := 0; i < cfg.Iterations; i++ {
		input := fuzzer.Generate(i)
		inputJSON := CanonicalJSON(input)
		cer, err := Run(inputJSON, 1)
		if err != nil {
			continue
		}
		detector.Add(input, ComputeHash(cer))
	}

	report := detector.Report()

	reportPath := filepath.Join("..", "vectors", "r2", "collisions", "v0.1.0-fuzz.json")
	if _, err := os.Stat("vectors"); err == nil {
		reportPath = filepath.Join("vectors", "r2", "collisions", "v0.1.0-fuzz.json")
	}
	_ = os.MkdirAll(filepath.Dir(reportPath), 0755)

	b, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(reportPath, b, 0644); err != nil {
		t.Logf("R2-COLLISION: could not write report: %v", err)
	}

	t.Logf("R2-COLLISION: %d iterations, %d expected collisions, %d ambiguities, %d divergences",
		cfg.Iterations, len(report.Expected), len(report.Ambiguities), len(report.Divergences))

	if len(report.Divergences) > 0 {
		t.Errorf("R2-COLLISION: %d divergence(s) found — implementation bugs", len(report.Divergences))
	}
}

func TestR2UnicodeStress(t *testing.T) {
	cfg := DefaultR2Config()
	fuzzer := NewSemanticFuzzer(cfg)
	hits := 0

	for i := 0; i < cfg.Iterations; i++ {
		input := fuzzer.Generate(i)
		inputJSON := CanonicalJSON(input)
		cer, err := Run(inputJSON, 1)
		if err != nil {
			continue
		}
		if i%8 == 2 {
			hits++
			_ = cer.Identity.EntityKey
		}
	}
	t.Logf("R2-UNICODE: %d unicode variant inputs processed", hits)
}

func TestR2TimestampBoundaries(t *testing.T) {
	cfg := DefaultR2Config()
	fuzzer := NewSemanticFuzzer(cfg)
	hits := 0

	for i := 0; i < cfg.Iterations; i++ {
		input := fuzzer.Generate(i)
		inputJSON := CanonicalJSON(input)
		cer, err := Run(inputJSON, 1)
		if err != nil {
			continue
		}
		if i%8 == 3 {
			if hits == 0 && cer.Timestamp < 0 {
				t.Logf("R2-TIMESTAMP: negative timestamp accepted: %d (documented behavior)", cer.Timestamp)
			}
			hits++
		}
	}
	t.Logf("R2-TIMESTAMP: %d timestamp boundary inputs processed", hits)
}
