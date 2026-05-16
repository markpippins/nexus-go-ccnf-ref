# CER — Canonical Event Record

**Version:** 1
**Status:** Immutable (v0.1.0-ccnf)

---

## 1. Introduction

The Canonical Event Record (CER) is the output format of the CCNF
pipeline (§CCNF_SPEC.md) and the input format for the Replay Oracle
(§REPLAY_SPEC.md).  Every CER is a self-contained, hash-anchored
record of a single event in a causally-ordered distributed system.

---

## 2. Schema

### 2.1 Top-Level Fields

All fifteen fields are top-level JSON object keys.  The canonical
serialization order is lexicographic (§CCNF_SPEC.md §3.6), but the
logical schema is presented below in functional groupings.

### 2.2 Identity Fields

```json
{
  "event_id": "string",
  "event_version": 1,
  "ccnf_version": 1,
  "system": "nexus",
  "domain": "string",
  "timestamp": 1713225600
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `event_id` | string | yes | Unique event identifier |
| `event_version` | integer | yes | Event schema version (currently 1) |
| `ccnf_version` | integer | yes | CCNF protocol version (currently 1) |
| `system` | string | yes | System of origin (always `"nexus"`) |
| `domain` | string | yes | Business domain |
| `timestamp` | integer | yes | Epoch seconds (int64) |

### 2.3 Actor and Intent

```json
{
  "actor": {
    "type": "system",
    "id": "orchestrator-1"
  },
  "intent": {
    "type": "normalized_verb",
    "action": "create",
    "target_type": "node",
    "target_id": "node:abc"
  }
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `actor` | object | yes | Actor metadata (opaque to CCNF) |
| `intent` | object | yes | Normalized intent (§CCNF_SPEC §4.5) |

### 2.4 Identity

```json
{
  "identity": {
    "entity_key": "a1b2c3d4e5f6...",
    "type": "event",
    "scope": "executiongraph.v2",
    "collapse_key": "node:abc",
    "alias_keys": []
  }
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `identity.entity_key` | string | yes | SHA256-derived hard identity (§CCNF_SPEC §5.2) |
| `identity.type` | string | yes | Always `"event"` |
| `identity.scope` | string | yes | Scoped domain (§CCNF_SPEC §5.3) |
| `identity.collapse_key` | string or null | yes | Stable human-readable collapse identifier (§CCNF_SPEC §5.4) |
| `identity.alias_keys` | array of strings | yes | Historical names for read-time resolution (§CCNF_SPEC §5.5) |

### 2.5 Causality

```json
{
  "causality": {
    "parent_event_ids": [],
    "causal_chain_id": "chain-001",
    "trace_depth": 0,
    "ordered": true
  }
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `causality.parent_event_ids` | array | yes | Parent event IDs (empty for root events) |
| `causality.causal_chain_id` | string | yes | Causal chain identifier |
| `causality.trace_depth` | integer | yes | Depth in causal chain |
| `causality.ordered` | boolean | yes | Whether ordering is enforced |

**Defaults** (when causality is absent or partial):

| Missing Field | Default |
|---|---|
| Entire causality block | `parent_event_ids: []`, `causal_chain_id: ""`, `trace_depth: 0`, `ordered: true` |
| `ordered` | `true` |
| `parent_event_ids` | `[]` |

### 2.6 Artifacts and State

```json
{
  "artifact_refs": ["node:abc"],
  "state_delta": [
    {
      "artifact_id": "node:abc",
      "before_hash": null,
      "after_hash": "deadbeef...",
      "patch": {
        "status": "created",
        "value": 42
      }
    }
  ]
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `artifact_refs` | array of strings | yes | List of artifact references |
| `state_delta` | array of objects | yes | Per-artifact state mutations |

**StateDelta entry:**

| Field | Type | Required | Description |
|---|---|---|---|
| `artifact_id` | string | yes | Type:id artifact reference |
| `before_hash` | string or null | yes | Previous state hash (`null` for creates) |
| `after_hash` | string | yes | SHA256 of canonical patch JSON |
| `patch` | object | yes | State mutation payload |

### 2.7 Payload and Metadata

```json
{
  "payload": {
    "type": "structured",
    "data": {}
  },
  "compression": {
    "strategy": "full",
    "lossless": true,
    "compression_version": 1
  },
  "signature": {
    "hash": "97cb25e0...",
    "signed_by": null
  }
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `payload` | object | yes | Event payload (artifact keys removed) |
| `compression` | object | yes | Compression metadata |
| `signature` | object | yes | Hash + signing info |

**Compression block** is hardcoded:

```json
{
  "strategy": "full",
  "lossless": true,
  "compression_version": 1
}
```

**Signature block** (§CCNF_SPEC §7):

```json
{
  "hash": "<hex sha256>",
  "signed_by": null
}
```

---

## 3. Canonical Key Order

When serialized, keys appear in this lexicographic order:

```
actor
artifact_refs
causality
ccnf_version
compression
domain
event_id
event_version
identity
intent
payload
signature
state_delta
system
timestamp
```

Within nested objects (`identity`, `causality`, `state_delta[*]`,
`compression`, `signature`, `payload`), keys are also sorted
lexicographically.

Identity sub-key order: `alias_keys`, `collapse_key`, `entity_key`,
`scope`, `type`.

Causality sub-key order: `causal_chain_id`, `ordered`,
`parent_event_ids`, `trace_depth`.

---

## 4. Serialization Rules (Summary of CCNF_SPEC §3)

1. **Compact JSON only** — no whitespace, no indentation, no trailing
   newline
2. **Sorted object keys** — lexicographic bytewise at every nesting
   level
3. **Integers** — no `.0` suffix
4. **Floats** — fixed notation only, no scientific notation
5. **Strings** — UTF-8, NFC-normalized, BOM and zero-width chars
   stripped
6. **Null vs absent** — all fields MUST be present; nullable fields
   use `null`
7. **Arrays** — `ordered: true` preserves input order; otherwise
   lexicographic
8. **Hash** — `SHA256(canonical_UTF8_bytes)`, hex-encoded, no
   trailing newline

---

## 5. Structural Invariants

### C1 — Schema Fixed
The set of 15 top-level fields is fixed.  No field may be added,
removed, or renamed without incrementing `event_version`.

### C2 — Version-Anchored
`ccnf_version` determines the interpretation of every field.
Cross-version comparison is invalid (§VERSIONING_MODEL.md).

### C3 — Self-Describing
Every CER contains its own `ccnf_version` and `event_version`.
No out-of-band context is required for interpretation.

### C4 — Hash-Anchored
The `signature.hash` is a commitment to the canonical bytes of every
other field.  Tampering with any field (except `signature`) breaks
the hash.

### C5 — Causally Traceable
Every CER has a `causal_chain_id` and optional
`parent_event_ids`.  The chain forms a DAG.

---

## 6. Version History

| Version | Date | Changes |
|---|---|---|
| 1 | 2026-05-16 | Initial specification (v0.1.0-ccnf) |
