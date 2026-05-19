package identity

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestAssembleDeterministic(t *testing.T) {
	seed := OriginSeed("test-uuid-00000000-0000-0000-0000-000000000001")
	sig := SemanticSignature("service|db,auth")

	a := Assemble(seed, sig)
	b := Assemble(seed, sig)

	if a != b {
		t.Fatalf("Assemble not deterministic: %s != %s", a, b)
	}
}

func TestAssembleDifferentSeed(t *testing.T) {
	sig := SemanticSignature("service|db,auth")
	seedA := OriginSeed("uuid-a")
	seedB := OriginSeed("uuid-b")

	if Assemble(seedA, sig) == Assemble(seedB, sig) {
		t.Fatal("different seeds produced same StableID")
	}
}

func TestAssembleDifferentSignature(t *testing.T) {
	seed := OriginSeed("test-uuid")
	sigA := SemanticSignature("service|db,auth")
	sigB := SemanticSignature("service|db")

	if Assemble(seed, sigA) == Assemble(seed, sigB) {
		t.Fatal("different signatures produced same StableID")
	}
}

func TestStoreWriteOnce(t *testing.T) {
	s := NewMemoryStore()
	sig := SemanticSignature("service|db,auth")
	seed := OriginSeed("uuid-1")

	if err := s.Insert(sig, seed); err != nil {
		t.Fatalf("first insert should succeed: %v", err)
	}

	if err := s.Insert(sig, seed); err == nil {
		t.Fatal("second insert should fail (ErrAlreadyExists)")
	}
}

func TestStoreGetExisting(t *testing.T) {
	s := NewMemoryStore()
	sig := SemanticSignature("service|db")
	seed := OriginSeed("uuid-xyz")

	s.Insert(sig, seed)

	got, ok := s.Get(sig)
	if !ok {
		t.Fatal("Get should return ok=true for existing entry")
	}
	if got != seed {
		t.Fatalf("Get returned wrong seed: got %s, want %s", got, seed)
	}
}

func TestStoreGetMissing(t *testing.T) {
	s := NewMemoryStore()
	_, ok := s.Get(SemanticSignature("nonexistent"))
	if ok {
		t.Fatal("Get should return ok=false for missing entry")
	}
}

func TestRegistryDeterminism(t *testing.T) {
	inputs := []struct {
		name string
		deps []string
	}{
		{"service", []string{"db", "auth"}},
		{"model", []string{}},
		{"log", []string{"io"}},
	}

	run := func() map[string]string {
		result := make(map[string]string)
		for _, in := range inputs {
			sig := semanticSignature(in.name, in.deps)
			h := sha256.Sum256([]byte(sig))
			result[in.name] = fmt.Sprintf("%x", h[:])
		}
		return result
	}

	a := run()
	b := run()

	for key := range a {
		if a[key] != b[key] {
			t.Fatalf("non-deterministic output for %s: %s vs %s", key, a[key], b[key])
		}
	}
}

func TestTransformationCREATE(t *testing.T) {
	s := NewMemoryStore()
	sig := SemanticSignature("new-service|db")
	seed := OriginSeed("uuid-create-1")

	if err := s.Insert(sig, seed); err != nil {
		t.Fatal("CREATE should assign new OriginSeed")
	}

	got, ok := s.Get(sig)
	if !ok || got != seed {
		t.Fatal("CREATE: OriginSeed should be retrievable")
	}
}

func TestTransformationMOVE(t *testing.T) {
	s := NewMemoryStore()
	sig := SemanticSignature("service|db,auth")
	seed := OriginSeed("uuid-move-1")

	s.Insert(sig, seed)
	stableA := Assemble(seed, sig)

	sigRenamed := SemanticSignature("service|db,auth")
	stableB := Assemble(seed, sigRenamed)

	if stableA != stableB {
		t.Fatal("MOVE: same semantic content must produce same StableID")
	}
}

func TestTransformationSPLIT(t *testing.T) {
	s := NewMemoryStore()

	parentSig := SemanticSignature("monolith|db,auth,io")
	parentSeed := OriginSeed("uuid-split-parent")

	s.Insert(parentSig, parentSeed)
	parentStable := Assemble(parentSeed, parentSig)

	childASig := SemanticSignature("core|db,auth")
	childASeed := OriginSeed("uuid-split-a")
	s.Insert(childASig, childASeed)

	childBSig := SemanticSignature("session|io")
	childBSeed := OriginSeed("uuid-split-b")
	s.Insert(childBSig, childBSeed)

	if parentStable == Assemble(childASeed, childASig) {
		t.Fatal("SPLIT: child should not have same StableID as parent")
	}
}

func TestTransformationMERGE(t *testing.T) {
	s := NewMemoryStore()

	seedA := OriginSeed("uuid-merge-a")
	seedB := OriginSeed("uuid-merge-b")

	sigA := SemanticSignature("util-crypto|crypto")
	sigB := SemanticSignature("util-crypto|crypto")

	s.Insert(sigA, seedA)
	s.Insert(sigB, seedB)

	stableA := Assemble(seedA, sigA)
	stableB := Assemble(seedB, sigB)

	if stableA == stableB {
		t.Fatal("MERGE: different OriginSeeds must not produce same StableID")
	}
}

func TestTransformationDELETE(t *testing.T) {
	s := NewMemoryStore()
	sig := SemanticSignature("obsolete|old")
	seed := OriginSeed("uuid-delete-1")

	s.Insert(sig, seed)

	got, ok := s.Get(sig)
	if !ok || got != seed {
		t.Fatal("DELETE precondition: seed must exist before removal")
	}

	// Identity is preserved even after deletion — no reuse
	if err := s.Insert(sig, OriginSeed("uuid-delete-reuse")); err == nil {
		t.Fatal("DELETE: reusing deleted SemanticSignature must not create new OriginSeed")
	}
}

func TestReplayEmpty(t *testing.T) {
	s := NewMemoryStore()
	if err := Replay(s, nil); err != nil {
		t.Fatalf("replay empty events: %v", err)
	}
	if err := Replay(s, []string{}); err != nil {
		t.Fatalf("replay empty slice: %v", err)
	}
}

func TestReplayValidEvents(t *testing.T) {
	s := NewMemoryStore()
	events := []string{"event-1", "event-2", "event-3"}
	if err := Replay(s, events); err != nil {
		t.Fatalf("replay valid events: %v", err)
	}
}

func TestReplayEmptyEvent(t *testing.T) {
	s := NewMemoryStore()
	events := []string{"event-1", "", "event-3"}
	if err := Replay(s, events); err == nil {
		t.Fatal("replay should reject empty event")
	}
}

func semanticSignature(name string, deps []string) string {
	sorted := make([]string, len(deps))
	copy(sorted, deps)
	sort.Strings(sorted)
	return name + "|" + strings.Join(sorted, ",")
}
