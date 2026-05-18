package account_test

import (
	"testing"

	"github.com/anomalyco/nexus-ccnf-ref/projection/account"
	"github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/snapshot"
	replay "github.com/anomalyco/nexus-ccnf-ref/runtime/replay"
)

func TestBuildAccountProjection_RoundTrip(t *testing.T) {
	state := replay.RuntimeState{
		Data: map[replay.StateKey]replay.StateValue{
			"acct:alice": []byte("100"),
			"acct:bob":   []byte("200"),
			"acct:carol": []byte("300"),
		},
		Version: 1,
	}

	snap := snapshot.NewFromRuntimeState(state)
	proj := account.BuildAccountProjection(snap)

	if proj.Count() != 3 {
		t.Fatalf("expected 3 accounts, got %d", proj.Count())
	}

	bal, ok := proj.GetBalance("alice")
	if !ok || bal != 100 {
		t.Fatalf("expected alice=100, got %d", bal)
	}

	bal, ok = proj.GetBalance("bob")
	if !ok || bal != 200 {
		t.Fatalf("expected bob=200, got %d", bal)
	}

	all := proj.AllBalances()
	if len(all) != 3 {
		t.Fatalf("expected 3 all balances, got %d", len(all))
	}
}

func TestBuildAccountProjection_Empty(t *testing.T) {
	state := replay.RuntimeState{
		Data:    map[replay.StateKey]replay.StateValue{},
		Version: 0,
	}

	snap := snapshot.NewFromRuntimeState(state)
	proj := account.BuildAccountProjection(snap)

	if proj.Count() != 0 {
		t.Fatalf("expected 0 accounts, got %d", proj.Count())
	}

	_, ok := proj.GetBalance("nonexistent")
	if ok {
		t.Fatal("expected false for nonexistent account")
	}
}

func TestBuildAccountProjection_MalformedValue(t *testing.T) {
	state := replay.RuntimeState{
		Data: map[replay.StateKey]replay.StateValue{
			"acct:good":  []byte("50"),
			"acct:bad":   []byte("not-a-number"),
			"acct:good2": []byte("75"),
		},
		Version: 1,
	}

	snap := snapshot.NewFromRuntimeState(state)
	proj := account.BuildAccountProjection(snap)

	// Malformed values should be gracefully skipped
	if proj.Count() != 2 {
		t.Fatalf("expected 2 valid accounts, got %d", proj.Count())
	}

	bal, ok := proj.GetBalance("good")
	if !ok || bal != 50 {
		t.Fatalf("expected good=50, got %d", bal)
	}

	_, ok = proj.GetBalance("bad")
	if ok {
		t.Fatal("expected false for malformed account")
	}
}

func TestBuildAccountProjection_NonAccountKeysIgnored(t *testing.T) {
	state := replay.RuntimeState{
		Data: map[replay.StateKey]replay.StateValue{
			"acct:alice": []byte("100"),
			"other:foo":  []byte("irrelevant"),
			"metadata":   []byte("also ignored"),
		},
		Version: 1,
	}

	snap := snapshot.NewFromRuntimeState(state)
	proj := account.BuildAccountProjection(snap)

	if proj.Count() != 1 {
		t.Fatalf("expected 1 account, got %d", proj.Count())
	}
}

func TestBuildAccountProjection_RebuildFromSameSnapshot(t *testing.T) {
	state := replay.RuntimeState{
		Data: map[replay.StateKey]replay.StateValue{
			"acct:alice": []byte("100"),
			"acct:bob":   []byte("200"),
		},
		Version: 1,
	}

	snap := snapshot.NewFromRuntimeState(state)

	p1 := account.BuildAccountProjection(snap)
	p2 := account.BuildAccountProjection(snap)

	if p1.Count() != p2.Count() {
		t.Fatal("rebuild from same snapshot should produce identical count")
	}

	b1, _ := p1.GetBalance("alice")
	b2, _ := p2.GetBalance("alice")
	if b1 != b2 {
		t.Fatal("rebuild from same snapshot should produce identical balances")
	}
}
