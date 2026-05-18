package replay

import (
	"testing"
)

func TestInitialState(t *testing.T) {
	s := initialState()
	if s.Version != 0 {
		t.Fatalf("expected version 0, got %d", s.Version)
	}
	if len(s.Data) != 0 {
		t.Fatalf("expected empty data, got %d entries", len(s.Data))
	}
}

func TestApplySingleWrite(t *testing.T) {
	s := initialState()
	delta := StateDelta{
		Writes: map[StateKey]StateValue{
			"key1": []byte("value1"),
		},
	}
	s2 := Apply(s, delta)

	if s2.Version != 1 {
		t.Fatalf("expected version 1, got %d", s2.Version)
	}
	if string(s2.Data["key1"]) != "value1" {
		t.Fatalf("expected value1, got %s", string(s2.Data["key1"]))
	}

	// Original unchanged
	if len(s.Data) != 0 {
		t.Fatal("original state must not be mutated")
	}
}

func TestApplyOverwrite(t *testing.T) {
	s := initialState()
	s = Apply(s, StateDelta{
		Writes: map[StateKey]StateValue{"k": []byte("v1")},
	})
	s = Apply(s, StateDelta{
		Writes: map[StateKey]StateValue{"k": []byte("v2")},
	})

	if string(s.Data["k"]) != "v2" {
		t.Fatalf("expected v2, got %s", string(s.Data["k"]))
	}
	if s.Version != 2 {
		t.Fatalf("expected version 2, got %d", s.Version)
	}
}

func TestApplyMultipleKeys(t *testing.T) {
	s := initialState()
	s = Apply(s, StateDelta{
		Writes: map[StateKey]StateValue{
			"a": []byte("1"),
			"b": []byte("2"),
		},
	})

	if string(s.Data["a"]) != "1" || string(s.Data["b"]) != "2" {
		t.Fatal("multiple keys must be written")
	}
	if s.Version != 1 {
		t.Fatalf("expected version 1, got %d", s.Version)
	}
}

func TestFoldEmptyEvents(t *testing.T) {
	s := Fold([]ReplayEvent{})
	if s.Version != 0 {
		t.Fatalf("expected version 0, got %d", s.Version)
	}
}

func TestFoldSingleEvent(t *testing.T) {
	s := Fold([]ReplayEvent{
		{EventID: "e1", Delta: StateDelta{
			Writes: map[StateKey]StateValue{"x": []byte("10")},
		}},
	})
	if string(s.Data["x"]) != "10" {
		t.Fatalf("expected 10, got %s", string(s.Data["x"]))
	}
	if s.Version != 1 {
		t.Fatalf("expected version 1, got %d", s.Version)
	}
}

func TestFoldMultipleEvents(t *testing.T) {
	events := []ReplayEvent{
		{EventID: "e1", Delta: StateDelta{
			Writes: map[StateKey]StateValue{"counter": []byte("1")},
		}},
		{EventID: "e2", Delta: StateDelta{
			Writes: map[StateKey]StateValue{"counter": []byte("2")},
		}},
		{EventID: "e3", Delta: StateDelta{
			Writes: map[StateKey]StateValue{"status": []byte("done")},
		}},
	}
	s := Fold(events)

	if string(s.Data["counter"]) != "2" {
		t.Fatalf("expected 2, got %s", string(s.Data["counter"]))
	}
	if string(s.Data["status"]) != "done" {
		t.Fatalf("expected done, got %s", string(s.Data["status"]))
	}
	if s.Version != 3 {
		t.Fatalf("expected version 3, got %d", s.Version)
	}
}

func TestFoldDeterminism(t *testing.T) {
	events := []ReplayEvent{
		{EventID: "e1", Delta: StateDelta{
			Writes: map[StateKey]StateValue{"a": []byte("1"), "b": []byte("2")},
		}},
		{EventID: "e2", Delta: StateDelta{
			Writes: map[StateKey]StateValue{"c": []byte("3")},
		}},
	}

	s1 := Fold(events)
	for i := 0; i < 5; i++ {
		s2 := Fold(events)
		if s2.Version != s1.Version {
			t.Fatalf("run %d: version changed", i)
		}
		for k, v := range s1.Data {
			if string(s2.Data[k]) != string(v) {
				t.Fatalf("run %d: key %s diverged", i, k)
			}
		}
	}
}

func TestFoldPreservesOriginalState(t *testing.T) {
	events := []ReplayEvent{
		{EventID: "e1", Delta: StateDelta{
			Writes: map[StateKey]StateValue{"key": []byte("original")},
		}},
	}

	// Before fold, there is no state to preserve, but the fold itself
	// should not be affected by external changes to the events slice.
	s := Fold(events)
	if string(s.Data["key"]) != "original" {
		t.Fatalf("expected original, got %s", string(s.Data["key"]))
	}
}

func TestCursorNavigation(t *testing.T) {
	events := []ReplayEvent{
		{EventID: "e1"},
		{EventID: "e2"},
		{EventID: "e3"},
	}

	c := NewCursor()
	e, ok := c.Event(events)
	if !ok || e.EventID != "e1" {
		t.Fatalf("expected e1 at index 0")
	}

	c = c.Step()
	e, ok = c.Event(events)
	if !ok || e.EventID != "e2" {
		t.Fatalf("expected e2 at index 1")
	}

	c = c.Step()
	e, ok = c.Event(events)
	if !ok || e.EventID != "e3" {
		t.Fatalf("expected e3 at index 2")
	}

	c = c.Step()
	_, ok = c.Event(events)
	if ok {
		t.Fatal("expected out of bounds")
	}
}

func TestCursorJump(t *testing.T) {
	events := []ReplayEvent{
		{EventID: "e1"}, {EventID: "e2"}, {EventID: "e3"},
	}

	c := NewCursor().Jump(2)
	e, ok := c.Event(events)
	if !ok || e.EventID != "e3" {
		t.Fatalf("expected e3 at index 2")
	}
}

func TestCursorJumpNegative(t *testing.T) {
	c := NewCursor().Jump(-5)
	if c.Index() != 0 {
		t.Fatalf("negative jump should clamp to 0")
	}
}

func TestValidateReplayInput_Valid(t *testing.T) {
	input := ReplayInput{
		Events: []ReplayEvent{
			{EventID: "e1"},
		},
		CerRootHash:       "abc",
		TraceRootHash:     "def",
		ReplayBindingHash: "ghi",
		CCNFVersion:       1,
		SemanticsVersion:  1,
		EventCount:        1,
	}
	if err := ValidateReplayInput(input); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateReplayInput_EmptyEvents(t *testing.T) {
	input := ReplayInput{
		Events:      []ReplayEvent{},
		CerRootHash: "abc",
		EventCount:  0,
	}
	if err := ValidateReplayInput(input); err != ErrEmptyEventList {
		t.Fatalf("expected ErrEmptyEventList, got %v", err)
	}
}

func TestValidateReplayInput_CountMismatch(t *testing.T) {
	input := ReplayInput{
		Events: []ReplayEvent{
			{EventID: "e1"},
			{EventID: "e2"},
		},
		CerRootHash: "abc",
		TraceRootHash: "def",
		EventCount: 3,
	}
	if err := ValidateReplayInput(input); err != ErrEventCountMismatch {
		t.Fatalf("expected ErrEventCountMismatch, got %v", err)
	}
}

func TestValidateReplayInput_EmptyEventID(t *testing.T) {
	input := ReplayInput{
		Events: []ReplayEvent{
			{EventID: ""},
		},
		CerRootHash:   "abc",
		TraceRootHash: "def",
		EventCount:    1,
	}
	if err := ValidateReplayInput(input); err == nil {
		t.Fatal("expected error for empty EventID")
	}
}

func TestAdapterRoundTrip(t *testing.T) {
	input := ReplayInput{
		Events: []ReplayEvent{
			{EventID: "e1", Delta: StateDelta{
				Writes: map[StateKey]StateValue{"k": []byte("v")},
			}},
		},
		CerRootHash:       "abc",
		TraceRootHash:     "def",
		ReplayBindingHash: "ghi",
		CCNFVersion:       1,
		SemanticsVersion:  1,
		EventCount:        1,
	}

	s := Fold(input.Events)
	if string(s.Data["k"]) != "v" {
		t.Fatalf("expected v, got %s", string(s.Data["k"]))
	}
	if s.Version != 1 {
		t.Fatalf("expected version 1, got %d", s.Version)
	}
}
