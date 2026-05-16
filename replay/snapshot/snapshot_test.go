package snapshot

import (
	"testing"

	"github.com/anomalyco/nexus-ccnf-ref/replay"
)

func makeEvents() []replay.CEREvent {
	return []replay.CEREvent{
		{EventID: "e1", CausalChainID: "chain-a", Sequence: 1, Timestamp: 1000, EntityKey: "entity:x", ArtifactID: "node:1", StateDelta: map[string]any{"status": "created", "value": 1}},
		{EventID: "e2", CausalChainID: "chain-a", Sequence: 2, Timestamp: 1001, EntityKey: "entity:x", ArtifactID: "node:1", StateDelta: map[string]any{"value": 2}},
		{EventID: "e3", CausalChainID: "chain-b", Sequence: 1, Timestamp: 1002, EntityKey: "entity:y", ArtifactID: "node:2", StateDelta: map[string]any{"status": "created", "color": "blue"}},
		{EventID: "e4", CausalChainID: "chain-a", Sequence: 3, Timestamp: 1003, EntityKey: "entity:x", ArtifactID: "node:1", StateDelta: map[string]any{"value": 3, "active": true}},
		{EventID: "e5", CausalChainID: "chain-b", Sequence: 2, Timestamp: 1004, EntityKey: "entity:y", ArtifactID: "node:2", StateDelta: map[string]any{"color": "red"}},
	}
}

func TestSnapshotEquivalence(t *testing.T) {
	events := makeEvents()

	foldState := replay.Fold(events)
	snapshot := BuildFromReplay(events)

	if !EqualStates(foldState, snapshot.State) {
		t.Fatal("Fold(events) != snapshot.State")
	}
}

func TestTriVersionLock(t *testing.T) {
	events := makeEvents()
	state := replay.Fold(events)

	t.Run("valid lock", func(t *testing.T) {
		s := Build(state, events)
		if err := ValidateTriVersionLock(s); err != nil {
			t.Fatalf("expected valid lock, got: %v", err)
		}
		if !IsValidLock(s) {
			t.Fatal("IsValidLock should be true")
		}
	})

	t.Run("mismatched ccnf version", func(t *testing.T) {
		s := Build(state, events)
		s.CCNFVersion = 2
		if err := ValidateTriVersionLock(s); err == nil {
			t.Fatal("expected lock error for mismatched CCNFVersion")
		}
	})

	t.Run("mismatched collapse version", func(t *testing.T) {
		s := Build(state, events)
		s.CollapseVersion = 2
		if err := ValidateTriVersionLock(s); err == nil {
			t.Fatal("expected lock error for mismatched CollapseVersion")
		}
	})

	t.Run("mismatched rehydration version", func(t *testing.T) {
		s := Build(state, events)
		s.RehydrationVersion = 2
		if err := ValidateTriVersionLock(s); err == nil {
			t.Fatal("expected lock error for mismatched RehydrationVersion")
		}
	})

	t.Run("zero ccnf version", func(t *testing.T) {
		s := Build(state, events)
		s.CCNFVersion = 0
		if err := ValidateTriVersionLock(s); err == nil {
			t.Fatal("expected lock error for zero CCNFVersion")
		}
	})

	t.Run("all zero versions", func(t *testing.T) {
		s := Snapshot{State: state}
		if err := ValidateTriVersionLock(s); err == nil {
			t.Fatal("expected lock error for all zero versions")
		}
	})
}

func TestSnapshotDeepEquality(t *testing.T) {
	events := makeEvents()

	s1 := BuildFromReplay(events)
	s2 := BuildFromReplay(events)

	if !Compare(s1, s2) {
		t.Fatal("same events should produce identical snapshots")
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	events := makeEvents()

	if err := RoundTrip(events); err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
}

func TestSnapshotDivergence(t *testing.T) {
	events := makeEvents()

	t.Run("different entity state", func(t *testing.T) {
		badState := replay.RuntimeState{
			Entities: map[string]replay.EntityState{
				"entity:x": {
					ArtifactStates: map[string]any{"value": 999},
					LastEventSeq:   3,
				},
			},
			Version: 1,
		}
		snapshot := Build(badState, events)
		if err := ValidateStateEquivalence(snapshot, events); err == nil {
			t.Fatal("expected STATE_DIVERGENCE for different entity state")
		}
	})

	t.Run("wrong version count", func(t *testing.T) {
		differentState := replay.RuntimeState{
			Entities: map[string]replay.EntityState{},
			Version:  999,
		}
		snapshot := Build(differentState, events)
		if err := ValidateStateEquivalence(snapshot, events); err == nil {
			t.Fatal("expected STATE_DIVERGENCE for wrong version")
		}
	})

	t.Run("nil entities mismatch", func(t *testing.T) {
		emptyState := replay.RuntimeState{}
		snapshot := Build(emptyState, events)
		if err := ValidateStateEquivalence(snapshot, events); err == nil {
			t.Fatal("expected STATE_DIVERGENCE for nil vs populated entities")
		}
	})
}

func TestValidateFullPipeline(t *testing.T) {
	events := makeEvents()

	t.Run("valid snapshot and events", func(t *testing.T) {
		snapshot := BuildFromReplay(events)
		if err := Validate(snapshot, events); err != nil {
			t.Fatalf("Validate failed: %v", err)
		}
	})

	t.Run("verify alias", func(t *testing.T) {
		snapshot := BuildFromReplay(events)
		if err := Verify(snapshot, events); err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
	})
}

func TestSnapshotFromEvents(t *testing.T) {
	events := makeEvents()

	snapshot := SnapshotFromEvents(events)
	foldState := replay.Fold(events)

	if !EqualStates(foldState, snapshot.State) {
		t.Fatal("SnapshotFromEvents state != Fold(events)")
	}
	if err := ValidateTriVersionLock(snapshot); err != nil {
		t.Fatalf("SnapshotFromEvents has invalid lock: %v", err)
	}
}

func TestBuildPreservesInputState(t *testing.T) {
	events := makeEvents()
	state := replay.Fold(events)

	snapshot := Build(state, events)

	if !EqualStates(state, snapshot.State) {
		t.Fatal("Build should preserve input state")
	}

	state.Entities["entity:x"] = replay.EntityState{
		ArtifactStates: map[string]any{"hacked": true},
	}

	if EqualStates(state, snapshot.State) {
		t.Fatal("Build must copy state, not retain reference")
	}
}

func TestEmptyEvents(t *testing.T) {
	snapshot := BuildFromReplay(nil)

	if len(snapshot.State.Entities) != 0 {
		t.Fatal("empty events should produce empty entities")
	}
	if err := ValidateTriVersionLock(snapshot); err != nil {
		t.Fatalf("empty snapshot should have valid lock: %v", err)
	}
	if err := ValidateStateEquivalence(snapshot, nil); err != nil {
		t.Fatalf("empty snapshot should validate against nil events: %v", err)
	}
	if err := RoundTrip(nil); err != nil {
		t.Fatalf("RoundTrip(nil) failed: %v", err)
	}
}

func TestMetadataPreservation(t *testing.T) {
	events := makeEvents()
	snapshot := BuildFromReplay(events)

	if snapshot.CCNFVersion != CurrentCCNFVersion {
		t.Fatalf("expected CCNFVersion %d, got %d", CurrentCCNFVersion, snapshot.CCNFVersion)
	}
	if snapshot.CollapseVersion != CurrentCollapseVersion {
		t.Fatalf("expected CollapseVersion %d, got %d", CurrentCollapseVersion, snapshot.CollapseVersion)
	}
	if snapshot.RehydrationVersion != CurrentRehydrationVersion {
		t.Fatalf("expected RehydrationVersion %d, got %d", CurrentRehydrationVersion, snapshot.RehydrationVersion)
	}
	if snapshot.Timestamp != 1004 {
		t.Fatalf("expected timestamp 1004 (last event), got %d", snapshot.Timestamp)
	}
}
