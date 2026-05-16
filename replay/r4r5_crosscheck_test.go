package replay_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anomalyco/nexus-ccnf-ref/ccnf"
	"github.com/anomalyco/nexus-ccnf-ref/replay"
	"github.com/anomalyco/nexus-ccnf-ref/replay/snapshot"
)

func cerToReplayEvents(cer *ccnf.CER) []replay.CEREvent {
	if cer == nil {
		return nil
	}

	cc := cer.Causality
	chainID, _ := cc["causal_chain_id"].(string)
	var seq int64
	if s, ok := cc["trace_depth"].(float64); ok {
		seq = int64(s)
	}

	if len(cer.StateDelta) == 0 {
		return []replay.CEREvent{
			{
				EventID:       cer.EventID,
				CausalChainID: chainID,
				Sequence:      seq,
				Timestamp:     cer.Timestamp,
				EntityKey:     cer.Identity.EntityKey,
				StateDelta:    map[string]any{},
			},
		}
	}

	events := make([]replay.CEREvent, 0, len(cer.StateDelta))
	for _, d := range cer.StateDelta {
		events = append(events, replay.CEREvent{
			EventID:       cer.EventID + "/" + d.ArtifactID,
			CausalChainID: chainID,
			Sequence:      seq,
			Timestamp:     cer.Timestamp,
			EntityKey:     cer.Identity.EntityKey,
			ArtifactID:    d.ArtifactID,
			StateDelta:    d.Patch,
		})
	}
	return events
}

func findVectorDir() string {
	wd, _ := os.Getwd()
	for _, c := range []string{
		filepath.Join(wd, "..", "vectors", "v1"),
		filepath.Join(wd, "vectors", "v1"),
		"../vectors/v1",
		"vectors/v1",
	} {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	return ""
}

func TestR4R5CrossCheck(t *testing.T) {
	vectorDir := findVectorDir()
	if vectorDir == "" {
		t.Fatal("could not find vectors/v1 directory")
	}

	entries, err := os.ReadDir(vectorDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || e.Name() == "expected-hashes.json" {
			continue
		}

		name := strings.TrimSuffix(e.Name(), ".json")
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(vectorDir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}

			var raw map[string]any
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatal(err)
			}

			ccnfVersion := 1
			if v, ok := raw["ccnf_version"].(float64); ok {
				ccnfVersion = int(v)
			}

			expected, _ := raw["expected"].(map[string]any)
			errStr, hasError := expected["error"].(string)

			process := func(input map[string]any, label string) {
				inputJSON, err := json.Marshal(input)
				if err != nil {
					t.Fatal(err)
				}

				cer, err := ccnf.Run(inputJSON, ccnfVersion)
				if err != nil {
					if hasError && errStr != "" {
						return
					}
					t.Fatalf("%s: ccnf.Run: %v", label, err)
				}
				if cer == nil {
					t.Fatalf("%s: expected non-nil CER", label)
				}

				events := cerToReplayEvents(cer)
				foldState := replay.Fold(events)

				s := snapshot.BuildFromReplay(events)

				if err := snapshot.ValidateTriVersionLock(s); err != nil {
					t.Fatalf("%s: version lock: %v", label, err)
				}
				if err := snapshot.ValidateStateEquivalence(s, events); err != nil {
					t.Fatalf("%s: state divergence: %v", label, err)
				}
				if !snapshot.EqualStates(foldState, s.State) {
					t.Fatalf("%s: Fold(events) != Snapshot.State", label)
				}
			}

			if input, ok := raw["input"].(map[string]any); ok {
				process(input, "input")
			}
			if inputA, ok := raw["input_a"].(map[string]any); ok {
				process(inputA, "input_a")
			}
			if inputB, ok := raw["input_b"].(map[string]any); ok {
				process(inputB, "input_b")
			}
		})
	}
}

func TestR4R5EmptyEvents(t *testing.T) {
	events := []replay.CEREvent{}
	foldState := replay.Fold(events)
	s := snapshot.BuildFromReplay(events)

	if err := snapshot.ValidateTriVersionLock(s); err != nil {
		t.Fatalf("empty lock: %v", err)
	}
	if err := snapshot.ValidateStateEquivalence(s, events); err != nil {
		t.Fatalf("empty divergence: %v", err)
	}
	if len(foldState.Entities) != 0 {
		t.Fatal("empty fold should have 0 entities")
	}
	if len(s.State.Entities) != 0 {
		t.Fatal("empty snapshot should have 0 entities")
	}
}

func TestR4R5Determinism(t *testing.T) {
	events := []replay.CEREvent{
		{EventID: "e1", EntityKey: "entity:x", ArtifactID: "art:1", StateDelta: map[string]any{"a": 1}},
		{EventID: "e2", EntityKey: "entity:y", ArtifactID: "art:2", StateDelta: map[string]any{"b": 2}},
		{EventID: "e3", EntityKey: "entity:x", ArtifactID: "art:1", StateDelta: map[string]any{"c": 3}},
	}

	foldState := replay.Fold(events)

	s1 := snapshot.BuildFromReplay(events)
	s2 := snapshot.BuildFromReplay(events)

	if err := snapshot.Validate(s1, events); err != nil {
		t.Fatalf("s1 validation: %v", err)
	}
	if err := snapshot.Validate(s2, events); err != nil {
		t.Fatalf("s2 validation: %v", err)
	}
	if !snapshot.Compare(s1, s2) {
		t.Fatal("R4=R5 determinism: same events produced different snapshots")
	}
	if !snapshot.EqualStates(foldState, s1.State) {
		t.Fatal("R4=R5: Fold(events) != snapshot.State")
	}
}

func TestR4R5MultiDeltaEvent(t *testing.T) {
	cers := []*ccnf.CER{
		{
			EventID:   "multi-1",
			Timestamp: 1000,
			Identity: ccnf.Identity{
				EntityKey: "entity:x",
			},
			StateDelta: []ccnf.StateDelta{
				{ArtifactID: "art:a", Patch: map[string]any{"x": 1}},
				{ArtifactID: "art:b", Patch: map[string]any{"y": 2}},
			},
		},
	}

	var events []replay.CEREvent
	for _, cer := range cers {
		events = append(events, cerToReplayEvents(cer)...)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 replay events for multi-delta CER, got %d", len(events))
	}
	if events[0].ArtifactID != "art:a" {
		t.Fatalf("event[0] artifact: want art:a, got %s", events[0].ArtifactID)
	}
	if events[1].ArtifactID != "art:b" {
		t.Fatalf("event[1] artifact: want art:b, got %s", events[1].ArtifactID)
	}

	foldState := replay.Fold(events)
	s := snapshot.BuildFromReplay(events)

	if err := snapshot.ValidateStateEquivalence(s, events); err != nil {
		t.Fatalf("multi-delta divergence: %v", err)
	}
	if len(foldState.Entities["entity:x"].ArtifactStates) != 2 {
		t.Fatalf("expected 2 artifact keys, got %d", len(foldState.Entities["entity:x"].ArtifactStates))
	}
}

func TestR4R5RoundTrip(t *testing.T) {
	events := []replay.CEREvent{
		{EventID: "e1", EntityKey: "e:x", ArtifactID: "a:1", StateDelta: map[string]any{"v": 1}},
		{EventID: "e2", EntityKey: "e:y", ArtifactID: "a:2", StateDelta: map[string]any{"v": 2}},
	}

	if err := snapshot.RoundTrip(events); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
}
