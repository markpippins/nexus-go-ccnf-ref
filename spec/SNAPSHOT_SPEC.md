# Snapshot Oracle — State Materialization and Verification

**Version:** 1
**Status:** Immutable (v0.1.0-ccnf)

---

## 1. Introduction

The Snapshot Oracle is a verification layer that answers one question:

```
snapshot(S_n) == fold(events[0..n])
```

A snapshot is a materialized checkpoint of runtime state at a point in
the event log.  The oracle ensures that the snapshot is consistent
with the events that produced it.  It is NOT a storage system — it is
a **consistency oracle**.

---

## 2. Core Types

### 2.1 Snapshot

```rust
struct Snapshot {
    state:               RuntimeState,    // §REPLAY_SPEC §2.2
    ccnf_version:        i64,
    collapse_version:    i64,
    rehydration_version: i64,
    timestamp:           i64,             // epoch seconds
}
```

| Field | Description |
|---|---|
| `state` | The frozen runtime state at the snapshot point |
| `ccnf_version` | CCNF protocol version that produced the source events |
| `collapse_version` | Collapse algorithm version used |
| `rehydration_version` | Rehydration algorithm version used |
| `timestamp` | Unix timestamp of the snapshot |

### 2.2 SnapshotContext

```rust
struct SnapshotContext {
    snapshot:     Snapshot,
    source_events: List<CEREvent>,
}
```

Used for validation only — binds a snapshot to the events that
produced it.

---

## 3. Builder

### 3.1 Build

```rust
fn build(state: RuntimeState, events: List<CEREvent>) -> Snapshot {
    let ts = if events.is_empty() { now() }
             else { events.last().timestamp };

    Snapshot {
        state:               deep_copy(state),
        ccnf_version:        1,
        collapse_version:    1,
        rehydration_version: 1,
        timestamp:           ts,
    }
}
```

Rules:
- The state is deep-copied (snapshot owns its own copy).
- The timestamp is taken from the last event's `timestamp`.  If the
  event list is empty or the last timestamp is 0, the current wall
  clock is used.
- Version fields are currently hardcoded to 1.

### 3.2 BuildFromReplay

```rust
fn build_from_replay(events: List<CEREvent>) -> Snapshot {
    let state = fold(events);   // §REPLAY_SPEC §3.3
    build(state, events)
}
```

This is the only permitted recomputation path — the builder may call
`fold` to derive state from events.  The validator path MUST NOT
recompute state.

---

## 4. Validation

### 4.1 Validate

```rust
fn validate(snapshot: Snapshot, events: List<CEREvent>) -> Result<(), Error> {
    validate_version_lock(snapshot)?;
    validate_state_equivalence(snapshot, events)?;
    Ok(())
}
```

Validation order: version lock first, then state equivalence.

### 4.2 ValidateVersionLock

```rust
fn validate_version_lock(s: Snapshot) -> Result<(), Error> {
    if s.ccnf_version == 0 || s.collapse_version == 0 || s.rehydration_version == 0 {
        return Err("TRI_VERSION_LOCK_FAILURE: zero version present");
    }
    if s.ccnf_version != s.collapse_version {
        return Err("TRI_VERSION_LOCK_FAILURE: ccnf_version != collapse_version");
    }
    if s.ccnf_version != s.rehydration_version {
        return Err("TRI_VERSION_LOCK_FAILURE: ccnf_version != rehydration_version");
    }
    Ok(())
}
```

### 4.3 ValidateStateEquivalence

```rust
fn validate_state_equivalence(snapshot: Snapshot, events: List<CEREvent>) -> Result<(), Error> {
    let replayed = fold(events);   // §REPLAY_SPEC §3.3
    if !equal_states(replayed, snapshot.state) {
        return Err("STATE_DIVERGENCE: Fold(events) != snapshot.State");
    }
    Ok(())
}
```

---

## 5. Structural Equality

### 5.1 Compare

```rust
fn compare(a: Snapshot, b: Snapshot) -> bool {
    a.ccnf_version        == b.ccnf_version        &&
    a.collapse_version    == b.collapse_version    &&
    a.rehydration_version == b.rehydration_version &&
    a.timestamp           == b.timestamp           &&
    equal_states(a.state, b.state)
}
```

### 5.2 EqualStates

```rust
fn equal_states(a: RuntimeState, b: RuntimeState) -> bool {
    if a.version != b.version { return false; }
    if len(a.entities) != len(b.entities) { return false; }

    for (key, ea) in a.entities {
        let eb = b.entities[key];
        if eb is None { return false; }
        if !equal_entity_state(ea, eb) { return false; }
    }
    return true;
}
```

### 5.3 EqualEntityState

```rust
fn equal_entity_state(a: EntityState, b: EntityState) -> bool {
    if a.last_event_seq != b.last_event_seq { return false; }
    if len(a.artifact_states) != len(b.artifact_states) { return false; }

    for (k, va) in a.artifact_states {
        let vb = b.artifact_states[k];
        if vb is None { return false; }
        if !deep_equal(va, vb) { return false; }
    }
    return true;
}
```

Equality semantics:
- Map key presence is symmetric (both maps must have the same keys).
- Values are compared structurally (deep value equality, not pointer
  identity).
- Map iteration order does not affect equality.

---

## 6. Tri-Version Lock Contract

### 6.1 Rule

A snapshot is valid ONLY if all three version numbers match:

```
ccnf_version == collapse_version == rehydration_version
```

Additionally, none may be zero.

### 6.2 Rationale

The three versions protect against cross-deployment drift:

| Version | Guards |
|---|---|
| `ccnf_version` | CER schema changes |
| `collapse_version` | Identity collapse algorithm changes |
| `rehydration_version` | Replay state derivation changes |

If any component is upgraded independently, the lock detects the
mismatch at snapshot validation time — preventing silent
inconsistency.

### 6.3 Error

A lock violation produces `TRI_VERSION_LOCK_FAILURE`.  The error
message identifies which field(s) are wrong.

---

## 7. API

### 7.1 SnapshotFromEvents

```rust
fn snapshot_from_events(events: List<CEREvent>) -> Snapshot
```

Convenience alias for `build_from_replay(events)`.

### 7.2 Verify

```rust
fn verify(snapshot: Snapshot, events: List<CEREvent>) -> Result<(), Error>
```

Alias for `validate(snapshot, events)`.

### 7.3 RoundTrip

```rust
fn round_trip(events: List<CEREvent>) -> Result<(), Error> {
    let snapshot = build_from_replay(events);
    validate(snapshot, events)
}
```

Self-consistency check: build a snapshot from events, then validate
it against the same events.

### 7.4 IsValidLock

```rust
fn is_valid_lock(s: Snapshot) -> bool
```

Returns `true` iff `validate_version_lock(s)` succeeds.

---

## 8. Invariants

### S1 — Fold Equivalence
```
snapshot.state == fold(events)
```

### S2 — Tri-Version Lock Integrity
```
snapshot.ccnf_version == snapshot.collapse_version == snapshot.rehydration_version
```
None may be zero.

### S3 — Snapshot is Derived Only
Snapshots MUST NOT influence replay or CER.  They are read-only
artifacts.

### S4 — No Semantic Interpretation
The snapshot oracle does NOT:
- Interpret state deltas
- Resolve aliases
- Modify state
- Recompute events

### S5 — Determinism
Same input events → identical snapshot (including all version fields
and timestamp, assuming deterministic wall clock seeding).

### S6 — Input State Isolation
`build(state, events)` deep-copies the input state.  Subsequent
mutations to the original state do not affect the snapshot.
