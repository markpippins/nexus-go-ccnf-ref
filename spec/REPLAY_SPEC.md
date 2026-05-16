# Replay Oracle — State Reconstruction from CER Events

**Version:** 1
**Status:** Immutable (v0.1.0-ccnf)

---

## 1. Introduction

The Replay Oracle is a pure function that reconstructs runtime state
from an ordered sequence of CER events.  It is a deterministic fold
over an append-only event log — the mathematical foundation for all
state reconstruction in the system.

---

## 2. Core Types

### 2.1 CEREvent (Replay Projection)

```rust
struct CEREvent {
    event_id:        String,
    causal_chain_id: String,
    sequence:        i64,
    timestamp:       i64,   // epoch seconds
    entity_key:      String,
    artifact_id:     String,
    state_delta:     Map<String, JsonValue>,
}
```

This is a minimal projection of the full CER (§CER_SPEC.md).  Fields
not relevant to state reconstruction (e.g., `actor`, `intent`,
`signature`) are intentionally excluded.

### 2.2 RuntimeState

```rust
struct RuntimeState {
    entities: Map<String, EntityState>,
    version:  i64,
}
```

| Field | Description |
|---|---|
| `entities` | Map from entity key to entity state |
| `version` | Monotonically increasing counter, incremented on each mutation |

### 2.3 EntityState

```rust
struct EntityState {
    artifact_states: Map<String, JsonValue>,
    last_event_seq:  i64,
}
```

| Field | Description |
|---|---|
| `artifact_states` | Flat key-value map of artifact state |
| `last_event_seq` | The `sequence` of the most recent event applied |

---

## 3. Fold Semantics

### 3.1 Initial State

```rust
fn initial_state() -> RuntimeState {
    RuntimeState {
        entities: {},
        version:  0,
    }
}
```

### 3.2 ApplyEvent

```rust
fn apply_event(state: RuntimeState, event: CEREvent) -> RuntimeState {
    let entity = get_entity(state, event.entity_key)
        .unwrap_or(EntityState { artifact_states: {}, last_event_seq: 0 });

    let updated = apply_delta(entity, event.artifact_id, event.state_delta, event.sequence);

    return update_entity(state, event.entity_key, updated);
}
```

Rules:
- The entity is looked up by `entity_key`.  If absent, a zero-value
  entity is created.
- The event's `state_delta` map is merged into the entity's
  `artifact_states` map.
- The entity's `last_event_seq` is unconditionally set to the event's
  `sequence`.
- The state's `version` is incremented by exactly 1.
- The input state is NEVER mutated.

### 3.3 Fold

```rust
fn fold(events: List<CEREvent>) -> RuntimeState {
    let mut state = initial_state();
    for event in events {
        state = apply_event(state, event);
    }
    return state;
}
```

Rules:
- Events are processed in list order.
- Processing an empty list returns `initial_state()`.
- The fold is a pure function: same events → same state.

### 3.4 Delta Merge (apply_delta)

```rust
fn apply_delta(
    entity: EntityState,
    artifact_id: String,     // currently informational, not used for scoping
    delta: Map<String, JsonValue>,
    seq: i64,
) -> EntityState {
    let mut merged = copy(entity.artifact_states);
    for (k, v) in delta {
        merged[k] = v;       // last-write-wins
    }
    return EntityState {
        artifact_states: merged,
        last_event_seq: seq,
    };
}
```

Merge rules:
1. The delta map keys are merged into the entity's artifact state map.
2. Existing keys are overwritten (last-write-wins).
3. Keys not in the delta are preserved.
4. A new map is always allocated (no mutation of input).
5. Currently, `artifact_id` is not used for scoping — all deltas for
   an entity merge into the same flat `artifact_states` map.

### 3.5 Entity Update (update_entity)

```rust
fn update_entity(
    state: RuntimeState,
    entity_key: String,
    updated: EntityState,
) -> RuntimeState {
    let mut entities = copy(state.entities);
    entities[entity_key] = updated;
    return RuntimeState {
        entities: entities,
        version: state.version + 1,
    };
}
```

Rules:
- The entity is replaced entirely for the given key.
- Version is always incremented by 1, regardless of whether this is a
  new entity or an update to an existing one.
- A new map is always allocated.

---

## 4. Cursor Model

### 4.1 Cursor

```rust
struct Cursor {
    index: i64,
}
```

The cursor identifies a position in the event log.  It is an
immutable value type — all operations return a new `Cursor`.

### 4.2 Operations

| Operation | Signature | Semantics |
|---|---|---|
| `new_cursor` | `() -> Cursor` | Returns `Cursor { index: 0 }` |
| `step` | `(Cursor) -> Cursor` | Returns `Cursor { index: c.index + 1 }` |
| `jump` | `(Cursor, i64) -> Cursor` | If `i < 0`, clamps to 0; otherwise `Cursor { index: i }` |
| `get_index` | `(Cursor) -> i64` | Returns `c.index` |
| `event` | `(Cursor, List<CEREvent>) -> (CEREvent, bool)` | Returns `events[c.index]` if in bounds; `false` otherwise |

### 4.3 Bounds Behavior

| Condition | `event()` returns |
|---|---|
| `index < 0` | `(_, false)` |
| `index >= len(events)` | `(_, false)` |
| `0 <= index < len(events)` | `(events[index], true)` |

---

## 5. Replay API

### 5.1 Replay (full)

```rust
fn replay(events: List<CEREvent>) -> RuntimeState {
    fold(events)
}
```

### 5.2 ReplayFromCursor

```rust
fn replay_from_cursor(events: List<CEREvent>, cursor: Cursor) -> RuntimeState {
    if cursor.index < 0:
        fold(events)           // negative → full replay
    if cursor.index >= len(events):
        initial_state()        // past end → empty state
    fold(events[cursor.index..])
}
```

### 5.3 ReplayRange

```rust
fn replay_range(events: List<CEREvent>, start: i64, end: i64) -> RuntimeState {
    let s = max(0, start);
    let e = min(len(events), end);
    if s >= e:
        initial_state()
    fold(events[s..e])
}
```

Bounds are clamped: `start` is clamped to `0`, `end` is clamped to
`len(events)`.  An empty range returns `initial_state()`.

---

## 6. Invariants

### R1 — Determinism
Given the same event list, every invocation of `fold` produces
identical `RuntimeState`.

### R2 — Idempotency
```
replay(events) == replay(events)
```

### R3 — Equivalence
```
replay(events) == fold(events)
replay_range(events, 0, len(events)) == fold(events)
```

### R4 — Cursor Composition
```
replay_from_cursor(events, new_cursor()) == fold(events)
fold(events[0..n]) + replay_from_cursor(events, jump(n)) == fold(events)
```

### R5 — Entity Isolation
Events targeting different entity keys produce isolated entity states.
Mutating `entity:a` does not affect `entity:b`.

### R6 — Delta Monotonicity
Later deltas for the same entity key override earlier deltas for
matching keys within `artifact_states`.  Keys not present in the
later delta are preserved.

### R7 — Pure Function
`fold`, `apply_event`, `apply_delta`, and all cursor operations have
no side effects, no global state, and no randomness.
