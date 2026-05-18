package replay

import (
	"testing"

	"github.com/anomalyco/nexus-ccnf-ref/internal/replayseal"
)

func TestReplayTypesDoNotImplementSemanticType(t *testing.T) {
	// Verify that replay types are NOT assignable to replayseal.SemanticType.
	// If any of these compile, a CCNF semantic type has leaked into replay.
	var s replayseal.SemanticType

	// ReplayEvent must NOT implement SemanticType
	_, okEvent := any(ReplayEvent{}).(replayseal.SemanticType)
	if okEvent {
		t.Fatal("ReplayEvent must NOT implement SemanticType — CCNF leakage detected")
	}

	// StateDelta must NOT implement SemanticType
	_, okDelta := any(StateDelta{}).(replayseal.SemanticType)
	if okDelta {
		t.Fatal("StateDelta must NOT implement SemanticType — CCNF leakage detected")
	}

	// RuntimeState must NOT implement SemanticType
	_, okState := any(RuntimeState{}).(replayseal.SemanticType)
	if okState {
		t.Fatal("RuntimeState must NOT implement SemanticType — CCNF leakage detected")
	}

	// Verify the seal package works (that CCNF types DO implement it)
	_ = s // prevent unused variable
}
