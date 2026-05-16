package ccnf

import (
	"math/rand"
	"testing"
)

func TestFuzzDeterminism(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	iterations := 10000
	seenHashes := make(map[string]string)

	for i := 0; i < iterations; i++ {
		input := generateFuzzInput(rng, i)
		inputJSON := CanonicalJSON(input)

		cer, err := Run(inputJSON, 1)
		if err != nil {
			continue
		}

		canonHash := ComputeHash(cer)

		key := cer.Domain + ":" + cer.Identity.EntityKey
		if prev, ok := seenHashes[key]; ok {
			if prev != canonHash {
				t.Errorf("iteration %d: I8 VIOLATION: same key %q produced different hash\n  prev: %s\n  curr: %s", i, key, prev, canonHash)
			}
		} else {
			seenHashes[key] = canonHash
		}
	}
}

func generateFuzzInput(rng *rand.Rand, seed int) map[string]any {
	actions := []string{"create", "update", "delete", "execute", "validate", "emit"}
	targetTypes := []string{"node", "task", "graph", "workflow", "artifact"}
	domains := []string{"execution", "specification", "system", "test"}

	action := actions[rng.Intn(len(actions))]
	targetType := targetTypes[rng.Intn(len(targetTypes))]
	targetID := targetType + ":" + targetType + "-" + formatInt(seed)
	domain := domains[rng.Intn(len(domains))]

	input := map[string]any{
		"actor": map[string]any{
			"type":       "system",
			"id":         "fuzzer-" + formatInt(rng.Intn(10)),
			"session_id": "sess-" + formatInt(seed),
		},
		"intent": map[string]any{
			"action":      action,
			"target_type": targetType,
			"target_id":   targetID,
		},
		"payload": map[string]any{
			"data": map[string]any{},
		},
		"domain":   domain,
		"event_id": "fuzz-" + formatInt(seed),
		"timestamp": float64(1713225600 + rng.Intn(86400*30)),
		"causality": map[string]any{
			"parent_event_ids": []any{},
			"causal_chain_id":  "chain-" + domain + "-" + formatInt(seed%100),
			"trace_depth":      rng.Intn(10),
		},
	}

	if rng.Float64() < 0.3 {
		payloadData := input["payload"].(map[string]any)["data"].(map[string]any)
		payloadData[targetType+":"+targetID] = map[string]any{
			"state":     "active",
			"iteration": seed,
		}
	}

	return input
}

func formatInt(n int) string {
	if n < 10 {
		return string(rune('0'+n))
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
