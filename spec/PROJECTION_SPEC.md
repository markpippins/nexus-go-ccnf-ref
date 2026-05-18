# R10.3B Projection Specification

**Status:** Draft
**Version:** v0.1.0-ccnf
**Prerequisite:** R10.3A Rehydration
**Next:** R10.4 Deterministic External Interfaces

---

## P0 — Projection Non-Authority Principle

**Projection is not a source of truth.**

It is a reproducible interpretation of `ReplaySnapshot` only.

This prevents:
- Projection becoming a "query authority"
- Projection becoming an indexing engine with independent logic
- Projection becoming a second state system

---

## 1. Core Architecture

### Layer Model

```
runtime/         → mechanical truth (CCNF + Replay + Rehydrate)
projection/      → semantic truth (domain interpretation)
```

Projection is the **first consumer** of the kernel — sibling to `runtime/`, never inside it.

### Dependency Flow

```
projection/                  ──imports──▶  runtime/rehydrate/snapshot  (ReplaySnapshot only)
runtime/                     ──NO BACK-EDGE──▶  projection/
runtime/rehydrate/           ──NO BACK-EDGE──▶  projection/
```

---

## 2. Core Types

### Projection Interface

```go
type Projection interface {
    Kind() string
}
```

Key design decision: `Projection` does NOT expose `Snapshot()`.

Snapshot is a **construction-time dependency only** — injected via `Build*` functions, never available as a runtime capability. This prevents consumers from traversing back into the structural layer.

### AccountBalance (Canonical Example)

```go
type AccountBalance struct {
    ID      string
    Balance uint64
}

func (AccountBalance) Kind() string { return "account_balance" }
```

This is the single illustrative projection type. It is **not extensible** — real domain projections arrive in production-specific layers.

### AccountProjection

```go
type AccountProjection struct {
    cache map[string]cacheEntry  // unexported
}

func BuildAccountProjection(snap snapshot.ReplaySnapshot) AccountProjection
func (p AccountProjection) GetBalance(id string) (uint64, bool)
func (p AccountProjection) AllBalances() []AccountBalance
func (p AccountProjection) Count() int
```

---

## 3. Construction Rule

Snapshot is a **construction-time dependency**, not a runtime capability.

```go
// CORRECT: snapshot injected at construction
func BuildAccountProjection(s snap.ReplaySnapshot) AccountProjection

// FORBIDDEN: snapshot exposed on interface
Projection.Snapshot()  // MUST NOT EXIST
```

---

## 4. Cache Rule (P4)

Projection caches are **implementation artifacts** and must not be observable or externally addressable.

**Allowed:**
- Unexported cache fields (`cache map[string]cacheEntry`)
- Internal indexes derived from snapshot data
- Precomputed maps for query performance

**Forbidden:**
- Exported cache types (`type Cache struct { ... }`)
- Cache getter methods that expose internal structure
- Cache mutation APIs
- Cache observability (size, hit rate, eviction policy)

**Rationale:** If cache is observable, consumers treat it as authoritative state. Disposable caches prevent semantic authority.

---

## 5. Projection State Semantics

### Allowed

Projection may maintain state, but only as **cache**:

- Caches
- Indexes
- Precomputed maps
- Materialized views
- Aggregation tables

Constraint: All projection state **must be derivable from ReplaySnapshot at any time**.

### Forbidden

- Hidden mutation sources
- Event ingestion
- Replay dependency
- Back-edges into `runtime/`
- Projection as incremental state machine
- Projection writing back into snapshot/replay
- Projection becoming authoritative state

### The Real Invariant

Projection is **disposable, but expensive to recompute**.

Not:

Projection is **persistent truth**.

---

## 6. Semantic Rule

Projection is allowed to know domain vocabulary:

- `account`
- `balance`
- `contract`
- `validator`

Rehydrate is **structurally blind** to all of these.

---

## 7. External Interface Rule

**R10.4 may depend ONLY on `projection/`.**

```
CORRECT:   R10.4 ──▶  projection/ ──▶  runtime/rehydrate/snapshot
FORBIDDEN: R10.4 ──▶  runtime/rehydrate/
```

If R10.4 touches `runtime/rehydrate/`, semantic leakage risk is reintroduced.

---

## 8. Invariants

### P0 — Non-Authority Principle
Projection is not a source of truth. It is a reproducible interpretation of ReplaySnapshot only.

### P1 — No Snapshot on Interface
`Projection` interface MUST NOT expose `Snapshot()` or any method that returns `ReplaySnapshot`.

### P2 — Dependency Direction
`projection/` MAY import `runtime/rehydrate/snapshot`. `runtime/` MUST NOT import `projection/`.

### P3 — Cache Sealing
Projection caches MUST be unexported. No cache getters, mutation APIs, or observability.

### P4 — Build Independence
`go build ./projection/...` MUST succeed without importing `runtime/replay/` or `ccnf/`.

### P5 — No Pointer Receivers
Projection types MUST use value receivers only. Zero pointer receivers.
*Enforcement: grep `func (.*\*)` in `projection/` → FAIL*
