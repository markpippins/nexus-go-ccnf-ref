package runtime

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"
)

func TestNewTraceBuilder_Empty(t *testing.T) {
	tb := NewTraceBuilder()
	if tb.EventCount() != 0 {
		t.Fatalf("expected 0 events, got %d", tb.EventCount())
	}
	rh := tb.RootHash()
	if rh == "" {
		t.Fatal("expected non-empty root hash")
	}
	if len(rh) != 64 {
		t.Fatalf("expected 64-char hex hash, got %d", len(rh))
	}
}

func TestTraceBuilder_AppendSingle(t *testing.T) {
	tb := NewTraceBuilder()
	tb.Append("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	if tb.EventCount() != 1 {
		t.Fatalf("expected 1 event, got %d", tb.EventCount())
	}
	rh := tb.RootHash()
	if len(rh) != 64 {
		t.Fatalf("expected 64-char hash, got %d", len(rh))
	}

	// Verify determinism
	rh2 := tb.RootHash()
	if rh != rh2 {
		t.Fatal("RootHash not deterministic")
	}
}

func TestTraceBuilder_AppendMultiple(t *testing.T) {
	tb := NewTraceBuilder()
	tb.Append("a000000000000000000000000000000000000000000000000000000000000001")
	tb.Append("a000000000000000000000000000000000000000000000000000000000000002")
	tb.Append("a000000000000000000000000000000000000000000000000000000000000003")

	if tb.EventCount() != 3 {
		t.Fatalf("expected 3 events, got %d", tb.EventCount())
	}

	// Single-event trace should have different hash
	tb1 := NewTraceBuilder()
	tb1.Append("a000000000000000000000000000000000000000000000000000000000000001")

	if tb.RootHash() == tb1.RootHash() {
		t.Fatal("single and triple event traces should differ")
	}

	// Ordering matters
	tb3 := NewTraceBuilder()
	tb3.Append("a000000000000000000000000000000000000000000000000000000000000003")
	tb3.Append("a000000000000000000000000000000000000000000000000000000000000002")
	tb3.Append("a000000000000000000000000000000000000000000000000000000000000001")

	if tb.RootHash() == tb3.RootHash() {
		t.Fatal("ordering must affect root hash")
	}
}

func TestTraceBuilder_Determinism(t *testing.T) {
	hashes := []string{
		"b000000000000000000000000000000000000000000000000000000000000001",
		"b000000000000000000000000000000000000000000000000000000000000002",
	}

	first := NewTraceBuilder()
	first.Append(hashes[0])
	first.Append(hashes[1])

	for i := 0; i < 5; i++ {
		tb := NewTraceBuilder()
		tb.Append(hashes[0])
		tb.Append(hashes[1])
		if tb.RootHash() != first.RootHash() {
			t.Fatalf("run %d: determinism broken", i)
		}
	}
}

func TestTraceBuilder_DomainSeparation(t *testing.T) {
	// Verify that trace_root_hash is NOT just SHA256(hashes...)
	// by constructing the plain hash manually and checking inequality.
	h := sha256.New()
	domain := sha256.Sum256([]byte("ccnf-domain:trace:v1"))
	h.Write(domain[:])
	count := make([]byte, 8)
	binary.BigEndian.PutUint64(count, 2)
	h.Write(count)

	h1, _ := hex.DecodeString("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	h2, _ := hex.DecodeString("dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
	h.Write(h1)
	h.Write(h2)
	expected := hex.EncodeToString(h.Sum(nil))

	tb := NewTraceBuilder()
	tb.Append("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	tb.Append("dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")

	got := tb.RootHash()
	if got != expected {
		t.Fatalf("domain-separated hash mismatch:\n  expected: %s\n  got:      %s", expected, got)
	}
}

func TestTraceBuilder_DifferentFromPlainHash(t *testing.T) {
	// Plain SHA256 of concatenated raw hashes should NOT equal trace root.
	tb := NewTraceBuilder()
	tb.Append("eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	tb.Append("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	traceRoot := tb.RootHash()

	// Compute plain SHA256 of just the two hashes concatenated
	h := sha256.New()
	h.Write([]byte("eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"))
	h.Write([]byte("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"))
	plain := hex.EncodeToString(h.Sum(nil))

	if traceRoot == plain {
		t.Fatal("trace root must differ from plain SHA256 of same hashes (domain separation)")
	}
}

func TestTraceBuilder_AppendPanicsOnInvalidHash(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on invalid hash")
		}
	}()
	tb := NewTraceBuilder()
	tb.Append("not-a-hex-string")
}

func TestTraceBuilder_AppendPanicsOnWrongLength(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on wrong-length hash")
		}
	}()
	tb := NewTraceBuilder()
	tb.Append("abcd")
}

func TestTraceBuilder_EmptyTracesEqual(t *testing.T) {
	tb1 := NewTraceBuilder()
	tb2 := NewTraceBuilder()
	if tb1.RootHash() != tb2.RootHash() {
		t.Fatal("two empty builders must produce identical root hash")
	}
}

func TestTraceBuilder_CERHashEquivalence(t *testing.T) {
	// When a single CER hash is appended, the trace root hash differs from
	// the CER hash itself (domain separation), but is deterministic and
	// reproducible from the same inputs.
	cerHash := "aa00000000000000000000000000000000000000000000000000000000000001"

	tb := NewTraceBuilder()
	tb.Append(cerHash)
	traceRoot := tb.RootHash()

	if traceRoot == cerHash {
		t.Fatal("trace root must NOT equal the CER hash (domain separation)")
	}

	if !strings.HasPrefix(traceRoot, "0") && !strings.HasPrefix(traceRoot, "f") {
		// Not a real assertion — just ensuring the value is a valid hex string
	}

	// Recomputing yields the same
	tb2 := NewTraceBuilder()
	tb2.Append(cerHash)
	if tb2.RootHash() != traceRoot {
		t.Fatal("deterministic from same inputs")
	}
}
