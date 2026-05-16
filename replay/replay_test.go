package replay

import (
	"reflect"
	"testing"
)

func makeEvents() []CEREvent {
	return []CEREvent{
		{EventID: "e1", CausalChainID: "chain-a", Sequence: 1, EntityKey: "entity:x", ArtifactID: "node:1", StateDelta: map[string]any{"status": "created", "value": 1}},
		{EventID: "e2", CausalChainID: "chain-a", Sequence: 2, EntityKey: "entity:x", ArtifactID: "node:1", StateDelta: map[string]any{"value": 2}},
		{EventID: "e3", CausalChainID: "chain-b", Sequence: 1, EntityKey: "entity:y", ArtifactID: "node:2", StateDelta: map[string]any{"status": "created", "color": "blue"}},
		{EventID: "e4", CausalChainID: "chain-a", Sequence: 3, EntityKey: "entity:x", ArtifactID: "node:1", StateDelta: map[string]any{"value": 3, "active": true}},
		{EventID: "e5", CausalChainID: "chain-b", Sequence: 2, EntityKey: "entity:y", ArtifactID: "node:2", StateDelta: map[string]any{"color": "red"}},
	}
}

func TestFoldDeterminism(t *testing.T) {
	events := makeEvents()

	state1 := Fold(events)
	state2 := Fold(events)

	if !reflect.DeepEqual(state1, state2) {
		t.Fatal("Fold is not deterministic: same events produced different states")
	}
}

func TestReplayEquivalence(t *testing.T) {
	events := makeEvents()

	foldState := Fold(events)
	replayState := Replay(events)

	if !reflect.DeepEqual(foldState, replayState) {
		t.Fatal("Replay != Fold")
	}
}

func TestCursorReplay(t *testing.T) {
	events := makeEvents()
	fullState := Fold(events)

	t.Run("from cursor zero", func(t *testing.T) {
		cursor := NewCursor()
		state := ReplayFromCursor(events, cursor)
		if !reflect.DeepEqual(state, fullState) {
			t.Fatal("ReplayFromCursor(0) != Fold(all)")
		}
	})

	t.Run("from cursor mid", func(t *testing.T) {
		cursor := Cursor{Index: 2}
		state := ReplayFromCursor(events, cursor)

		if len(state.Entities) != 2 {
			t.Fatalf("expected 2 entities (entity:x from e4, entity:y from e3), got %d", len(state.Entities))
		}
		entityX, ok := state.Entities["entity:x"]
		if !ok {
			t.Fatal("expected entity:x (from e4)")
		}
		if entityX.ArtifactStates["value"] != 3 {
			t.Fatal("expected entity:x value == 3 (from e4)")
		}
		entityY, ok := state.Entities["entity:y"]
		if !ok {
			t.Fatal("expected entity:y (from e3)")
		}
		if entityY.ArtifactStates["color"] != "red" {
			t.Fatal("expected entity:y color == red (from e5)")
		}
	})

	t.Run("from cursor end", func(t *testing.T) {
		cursor := Cursor{Index: len(events)}
		state := ReplayFromCursor(events, cursor)
		if len(state.Entities) != 0 {
			t.Fatal("expected empty state from cursor at end")
		}
	})

	t.Run("step cursor", func(t *testing.T) {
		cursor := NewCursor()
		cursor = cursor.Step()
		state := ReplayFromCursor(events, cursor)
		e, ok := cursor.Event(events)
		if !ok || e.EventID != "e2" {
			t.Fatal("Step cursor should point to e2")
		}
		if state.Entities["entity:x"].ArtifactStates["value"] != 3 {
			t.Fatal("expected entity:x value=3 (overwritten by e4)")
		}
	})

	t.Run("jump cursor", func(t *testing.T) {
		cursor := NewCursor().Jump(3)
		state := ReplayFromCursor(events, cursor)
		e, ok := cursor.Event(events)
		if !ok || e.EventID != "e4" {
			t.Fatal("Jump cursor should point to e4 (index 3)")
		}
		if state.Entities["entity:x"].ArtifactStates["value"] != 3 {
			t.Fatal("expected entity:x value=3 from e4")
		}
		entityY, okY := state.Entities["entity:y"]
		if !okY {
			t.Fatal("entity:y should exist (from e5)")
		}
		if entityY.ArtifactStates["color"] != "red" {
			t.Fatal("expected entity:y color=red from e5")
		}
	})

	t.Run("negative jump clamped", func(t *testing.T) {
		cursor := NewCursor().Jump(-5)
		if cursor.GetIndex() != 0 {
			t.Fatal("negative Jump should clamp to 0")
		}
	})
}

func TestReplayIdempotency(t *testing.T) {
	events := makeEvents()

	stateA := Replay(events)
	stateB := Replay(events)

	if !reflect.DeepEqual(stateA, stateB) {
		t.Fatal("Replay is not idempotent: same input gave different output")
	}
}

func TestReplayRange(t *testing.T) {
	events := makeEvents()

	t.Run("empty range", func(t *testing.T) {
		state := ReplayRange(events, 0, 0)
		if len(state.Entities) != 0 {
			t.Fatal("expected empty state for empty range")
		}
	})

	t.Run("first two events only", func(t *testing.T) {
		state := ReplayRange(events, 0, 2)
		if len(state.Entities) != 1 {
			t.Fatalf("expected 1 entity, got %d", len(state.Entities))
		}
		if state.Entities["entity:x"].ArtifactStates["value"] != 2 {
			t.Fatal("expected value=2 after events e1,e2")
		}
	})

	t.Run("clamped range", func(t *testing.T) {
		state := ReplayRange(events, 0, 100)
		full := Replay(events)
		if !reflect.DeepEqual(state, full) {
			t.Fatal("ReplayRange clamped to len(events) != Replay(all)")
		}
	})

	t.Run("invalid start", func(t *testing.T) {
		state := ReplayRange(events, -1, 2)
		if len(state.Entities) != 1 {
			t.Fatal("negative start should clamp to 0")
		}
	})
}

func TestEntityIsolation(t *testing.T) {
	events := []CEREvent{
		{EventID: "e1", EntityKey: "entity:a", ArtifactID: "art:1", StateDelta: map[string]any{"x": 1}},
		{EventID: "e2", EntityKey: "entity:b", ArtifactID: "art:2", StateDelta: map[string]any{"y": 2}},
	}

	state := Fold(events)

	if len(state.Entities) != 2 {
		t.Fatalf("expected 2 isolated entities, got %d", len(state.Entities))
	}

	v, ok := state.Entities["entity:a"].ArtifactStates["x"]
	if !ok || v != 1 {
		t.Fatal("entity:a.x should be 1")
	}
	v, ok = state.Entities["entity:b"].ArtifactStates["y"]
	if !ok || v != 2 {
		t.Fatal("entity:b.y should be 2")
	}

	if _, exists := state.Entities["entity:a"].ArtifactStates["y"]; exists {
		t.Fatal("entity:a should not have artifact y")
	}
}

func TestDeltaMergeSemantics(t *testing.T) {
	events := []CEREvent{
		{EventID: "e1", EntityKey: "entity:x", ArtifactID: "art:1", StateDelta: map[string]any{"a": 1, "b": 2}},
		{EventID: "e2", EntityKey: "entity:x", ArtifactID: "art:1", StateDelta: map[string]any{"b": 3, "c": 4}},
	}

	state := Fold(events)

	artifacts := state.Entities["entity:x"].ArtifactStates
	if artifacts["a"] != 1 {
		t.Fatal("delta merge: a should remain 1")
	}
	if artifacts["b"] != 3 {
		t.Fatal("delta merge: b should be overwritten to 3")
	}
	if artifacts["c"] != 4 {
		t.Fatal("delta merge: c should be added as 4")
	}
	if len(artifacts) != 3 {
		t.Fatalf("expected 3 artifact keys, got %d", len(artifacts))
	}
}

func TestSequenceTracking(t *testing.T) {
	events := []CEREvent{
		{EventID: "e1", EntityKey: "entity:x", Sequence: 1, StateDelta: map[string]any{}},
		{EventID: "e2", EntityKey: "entity:x", Sequence: 5, StateDelta: map[string]any{}},
		{EventID: "e3", EntityKey: "entity:y", Sequence: 10, StateDelta: map[string]any{}},
	}

	state := Fold(events)

	if state.Entities["entity:x"].LastEventSeq != 5 {
		t.Fatalf("entity:x last seq should be 5, got %d", state.Entities["entity:x"].LastEventSeq)
	}
	if state.Entities["entity:y"].LastEventSeq != 10 {
		t.Fatalf("entity:y last seq should be 10, got %d", state.Entities["entity:y"].LastEventSeq)
	}
}

func TestEmptyEvents(t *testing.T) {
	state := Fold(nil)

	if len(state.Entities) != 0 {
		t.Fatal("empty events should produce empty state")
	}
	if state.Version != 0 {
		t.Fatal("empty events version should be 0")
	}
}

func TestNewCursor(t *testing.T) {
	c := NewCursor()
	if c.GetIndex() != 0 {
		t.Fatal("NewCursor should start at 0")
	}
}
