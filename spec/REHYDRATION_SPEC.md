# R10.3 Rehydration Specification

**Status:** Draft  
**Version:** v0.1.0-ccnf  
**Prerequisite:** R10.2 Sealed Replay  
**Next:** R10.4+ Domain Views

---

## 1. The One Law

Rehydration produces VIEWS, never MODELS.

If any rehydrated object can influence replay or CCNF, the architecture has failed.

**Layer Power Gradient:**

| Layer | Authority | Action |
|-------|-----------|--------|
| CCNF | Decide | Semantic execution |
| Replay | Remember | Structural fold |
| Rehydrate | Describe | Read-only projection |

Rehydrate MUST NOT decide. Rehydrate MUST NOT remember across calls.

---

## 2. The Observer Wall

Views are disposable projections. They behave like JSON decoded from disk:

- No identity (no ID treated as identity handle)
- No equality semantics (not used as map keys)
- No mutation lifecycle (never persisted or diffed)
- No caching assumptions (safe to drop at any time)

**Enforcement:**
- Views use value receivers only — zero pointer receivers
- Views may not return `bool` decisions
- Views are non-identity value objects

---

## 3. Core Types

### ReplaySnapshot Interface

```go
type ReplaySnapshot interface {
    Get(key []byte) ([]byte, bool)
    Scan(prefix []byte) Iterator
    Height() uint64
}
```

**Forbidden methods:** `Apply`, `Execute`, `Validate`, `Commit`, `State`, `Runtime`

Snapshot exposes memory, not behavior.

### Iterator Interface

```go
type Iterator interface {
    Next() bool
    Key() []byte
    Value() []byte
    Close()
}
```

### View Interface

```go
type View interface {
    Kind() string
}
```

No methods returning `bool` or `error`. No pointer receivers.

### Decoder Interface

```go
type Decoder interface {
    Decode(key, value []byte) (View, error)
}
```

### ViewRegistry

```go
type ViewRegistry struct { /* unexported routes */ }
func New(specs ...RouteSpec) *ViewRegistry
func (r *ViewRegistry) Decode(k, v []byte) (View, bool)
```

**NO `Register()` method.** Registry is write-once at process boot.

---

## 4. Three Immutable Registry Laws

### Law 1: Prefix Ownership Is Absolute

A decoder owns a storage prefix, not a concept.

```
Correct:  contracts/v1/  → DecoderV1
Correct:  accounts/      → AccountDecoder

Wrong:    active_contracts/   (encodes meaning in prefix)
Wrong:    expired_contracts/  (encodes lifecycle in prefix)
```

If prefixes encode meaning, corruption begins.

### Law 2: Registry Is Write-Once at Boot

After startup:
- NO dynamic registration
- NO plugin injection
- NO runtime mutation
- NO `init()` registration

Mutable registries always become semantic routing layers.

### Law 3: Registry Cannot Observe Replay

Registry receives only `(key, value bytes)`.

It NEVER sees:
- Events
- Commands
- History
- Previous values
- Other keys

---

## 5. Decoder Rules

### Allowed

- Decode bytes into a typed view
- Rename fields
- Version adapt (e.g., `v1` vs `v2` prefix)
- Format transform

### Forbidden

- Query the snapshot again
- Aggregate keys
- Compute balances or derived values
- Validate invariants
- Resolve cross-key references
- Call helpers that inspect other keys

A decoder touching anything except its own KV pair is already illegal.

---

## 6. Versioned Prefix Pattern

Long-lived systems evolve views safely via versioned prefixes:

```
contracts/v1/  → DecoderV1
contracts/v2/  → DecoderV2
```

No migrations required. Replay truth remains unchanged. Both versioned snapshots coexist in the same key space.

---

## 7. Reader API

### Correct

```go
type Reader struct { /* snap + reg */ }
func New(snap ReplaySnapshot, reg *ViewRegistry) *Reader
func (r *Reader) Scan(prefix []byte) []View
```

Reader is stateless, derived, disposable.

### Forbidden

```go
// These are execution queries wearing observer clothing:
Reader.GetContract(···)
Reader.TotalStake()
Reader.ActiveValidators()
```

---

## 8. Three Failure Modes

### Failure Mode 1: Entity Resurrection

Bad: `ReplayState → Contract struct`  
Good: `ReplayState → ContractView`

Views are read-only, derived, disposable, non-authoritative.
Entities are mutable, authoritative, semantic.

Never recreate entities.

### Failure Mode 2: Helper Query Creep

Bad: `GetActiveContracts()`  
Good: `Scan(prefix)`, `Decode(key, value)`

Queries must remain mechanical — storage-shaped, never domain-shaped.

### Failure Mode 3: Validation During Rehydration

Bad: `if !contract.IsValid() { panic }`

This silently reintroduces execution semantics.
Replay must assume validity. Validation belongs only to R10.1.

---

## 9. Invariants

### H1: Zero Pointer Receivers
View types MUST NOT have pointer receiver methods in `view/`.  
*Enforcement: grep `func (.*\*)` → FAIL*

### H2: Zero Domain Words
Rehydrate MUST NOT contain domain-specific identifiers (`contract`, `account`, `stake`, `validator`, `tx`, `execute`, `apply`, `validate`, `state machine`).  
*Enforcement: grep for domain words → FAIL*

### H3: Zero Semantic Queries
Reader MUST NOT expose domain-shaped query methods. Only `Scan(prefix)`.  
*Enforcement: manual code review*

### H4: Registry Immutability
ViewRegistry MUST NOT have a `Register()` method. Constructed at boot via `New()`.  
*Enforcement: grep `func Register(` → FAIL*

### H5: Import Isolation
Rehydrate MUST NOT import `ccnf/`, `execution/`, `engine/`, `replay/apply`, `replay/core`.  
*Enforcement: grep for forbidden imports → FAIL*

### H6: Build Isolation
`go build ./runtime/rehydrate/...` MUST succeed without CCNF packages.  
*Enforcement: CI build test*

### H7: Cross-View Isolation
Decoders MUST NOT import other view packages.  
*Enforcement: grep `import runtime/rehydrate/view/` in decode/ → FAIL*

---

## 10. Long-Term Proof

After R10.3 is correct:

```
rm -rf ccnf/
go build ./runtime/rehydrate/...   # PASS
go test ./runtime/rehydrate/...    # PASS
```

Rehydration survives without the brain.
