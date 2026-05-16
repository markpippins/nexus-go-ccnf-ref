# CCNF — Canonical Event Representation Format

**Version:** 1
**Status:** Immutable (v0.1.0-ccnf)
**Replaces:** SERIALIZATION_CONTRACT.md

---

## 1. Introduction

The Canonical Normalization Format (CCNF) is a deterministic,
cross-platform pipeline that transforms raw structured input into a
canonical Event Record (CER).  Every conforming implementation, given
the same input bytes and CCNF version, MUST produce bitwise-identical
CER output.

---

## 2. Scope

This document specifies the CCNF pipeline: the 8 required
transformation steps, the canonical JSON serialization rules, the
validation gates, and the error contract.  It does NOT specify:

- CER schema (see CER_SPEC.md)
- Replay semantics (see REPLAY_SPEC.md)
- Snapshot equivalence (see SNAPSHOT_SPEC.md)
- Version migration policy (see VERSIONING_MODEL.md)

---

## 3. Canonical Serialization

### 3.1 Byte Layout

Canonical CCNF output is a single line of JSON with zero trailing
newline.  The MIME type is `application/ccnf+json`.

Formally:

```
output := encode(root)
```

where `encode` is defined by the type-specific rules below and produces
UTF-8 bytes.  No whitespace, no indentation, no newline is emitted.

### 3.2 Type Encoding Rules

| Input Type | Output | Notes |
|---|---|---|
| `null` | `null` | Literal bytes `null` |
| `boolean` | `true` or `false` | Lowercase |
| `integer` | Base-10 digits | No leading zeros, no `.0` suffix |
| `float` | Fixed notation | See §3.3 |
| `string` | JSON string | See §3.4 |
| `array` | Bracket-delimited | See §3.5 |
| `object` | Brace-delimited | See §3.6 |

### 3.3 Float Encoding

```
encodeFloat(f):
  s ← formatFloat(f, 'f', -1, 64)
  if s contains 'e' or 'E':
    output null     // scientific notation rejected
  if s contains '.':
    output s        // preserve decimal point
  else:
    output s        // integer-like float (e.g. 3.0 → "3")
```

Integral-valued floats are promoted to integer representation: `3.0`
serializes as `3`.  Sub-normal floats (NaN, Inf) are NOT representable
in canonical form; an implementation SHOULD reject them at ingress.

### 3.4 String Encoding

Every string is wrapped in U+0022 QUOTATION MARK.  Within the string:

| Codepoint | Escape |
|---|---|
| U+0022 (quote) | `\"` |
| U+005C (backslash) | `\\\\` |
| U+000A (newline) | `\n` |
| U+000D (carriage return) | `\r` |
| U+0009 (tab) | `\t` |
| U+0000–U+001F (other control) | `\u00XX` (lowercase hex) |
| All other bytes | Pass through literally |

The encoder MUST NOT strip or escape non-BMP codepoints.  UTF-8 bytes
are reproduced faithfully.

### 3.5 Array Encoding

```
encodeArray(arr):
  if arr is null:
    output "null"
  output '['
  for i, elem in arr:
    if i > 0: output ','
    output encode(elem)
  output ']'
```

Null arrays encode as `null`, not `[]`.  Arrays retain the order
provided by the caller; sorting is the caller's responsibility.

### 3.6 Object Encoding

```
encodeObject(obj):
  keys ← sorted(obj.keys)   // lexicographic UTF-8 bytewise sort
  output '{'
  for i, k in enumerate(keys):
    if i > 0: output ','
    output encodeString(k)
    output ':'
    output encode(obj[k])
  output '}'
```

Object keys MUST be sorted lexicographically at every nesting level.
The sort is bytewise UTF-8 comparison (no Unicode collation).

---

## 4. Normalization Rules

### 4.1 String Normalization

Every string in the input MUST undergo:

1. **Zero-width removal:** Remove U+200B (ZERO WIDTH SPACE), U+200C
   (ZERO WIDTH NON-JOINER), U+200D (ZERO WIDTH JOINER)
2. **BOM removal:** Remove U+FEFF (BYTE ORDER MARK)
3. **NFC normalization:** Apply Unicode Normalization Form C via
   UAX #15

This applies to both map keys and string values.

### 4.2 Timestamp Normalization

Input timestamps may be:
- **epoch seconds** (int64): stored as-is
- **ISO-8601 string** (RFC 3339): parsed and converted to epoch seconds
- **float64**: truncated to int64

If the timestamp is unparseable or zero, the implementation MUST
substitute the current wall-clock time (`time.Now().Unix()`).

Output representation is always epoch seconds as a JSON number.

### 4.3 Numeric Normalization

A `float64` value that is integer-valued (`f == (int64)f`) is
converted to `int64`.  This ensures `3.0` → `3` in the canonical
output.

### 4.4 Field Presence Rules

All fields in the serialized CER MUST be present.  The rules are:

| Case | Representation |
|---|---|
| Value is null | `null` in JSON |
| Value is empty array | `[]` |
| Value is empty object | `{}` |
| Value is undefined/absent | INVALID — rejected at parse |

The following fields are explicitly nullable:
- `identity.collapse_key`
- `identity.external_id` (reserved)
- `signature`

### 4.5 Intent Normalization

The `intent` field MUST be a JSON object (not a free-text string).
Free-text intents are unconditionally rejected.

The normalized intent has the fixed shape:

```json
{
  "type": "normalized_verb",
  "action": "create",       // one of controlled vocabulary
  "target_type": "node",    // or empty string
  "target_id": "node:abc"   // or empty string
}
```

**Controlled vocabulary** (closed set, version 1):
`create`, `update`, `delete`, `execute`, `validate`, `emit`

### 4.6 Artifact Resolution

Fields in `payload.data` whose keys match the `type:id` pattern are
extracted into `artifact_refs` and `state_delta`.  They are removed
from `payload.data`.

**Artifact ID format:** `type:id` (one colon) or `type:subtype:id`
(two colons).  No segment may be empty.  The pattern partitions on
`:` with a maximum of 3 segments.

---

## 5. Entity Key Derivation

### 5.1 Identity Type

The identity type is always the literal string `"event"`.

### 5.2 Entity Key

The entity key is `SHA256` of the canonical serialization of exactly
four fields, in sorted key order:

```
fields = {
  "actor":  <canonical(actor)>,
  "domain": <canonical(domain)>,
  "intent": <canonical(intent)>,
  "scope":  <canonical(scope)>,
}
```

After sorting keys lexicographically, the hash input is:

```
for each key in sorted(fields):
  write(key)
  write(0x00)
  write(canonical_json(fields[key]))
  write(0x00)
```

The digest is encoded as lowercase hex (`%x`).

### 5.3 Scope Derivation

The scope is derived from the domain:

| Domain | Scope |
|---|---|
| `"execution"` | `"executiongraph.v2"` |
| `"specification"` | `"specification.v1"` |
| `"system"` | `"system.v1"` |
| anything else | `"<domain>.v1"` |

### 5.4 Collapse Key

If both `intent.target_type` and `intent.target_id` are non-empty, the
collapse key is `"<target_type>:<target_id>"`.  Otherwise, it is
`null`.

### 5.5 Alias Keys

Alias keys are reserved for future identity linking.  Currently
always an empty array `[]`.

---

## 6. State Delta Computation

### 6.1 Delta per Artifact

Each artifact in `artifact_refs` produces one `StateDelta` entry:

| Field | Value |
|---|---|
| `artifact_id` | The artifact reference |
| `before_hash` | `null` (reserved for future use) |
| `after_hash` | `SHA256(canonical_json(patch))` (hex) |
| `patch` | The artifact value as a JSON object |

### 6.2 Scope Validation

If `intent.target_id` is present and non-empty, every
`state_delta[].artifact_id` MUST share the same prefix (text before
the first `:`).  A prefix mismatch is a scope violation.

---

## 7. Hash and Signature

### 7.1 Canonical Hash

```
hash ← SHA256(canonical_json(all_cer_fields_except_signature))
```

The hash is computed over every field of the CER EXCEPT `signature`.
The `signature` field is removed from the map before serialization.
The digest is lowercase hex.

### 7.2 Signature Block

```json
{
  "hash": "<hex hash>",
  "signed_by": null
}
```

The `signed_by` field is `null` (reserved for future signing
infrastructure).

---

## 8. Full Pipeline

The CCNF pipeline applies these steps in strict order:

1. **Version gate:** Reject if `ccnfVersion != CurrentCCNFVersion`
2. **Structural parse:** Validate JSON, check required top-level
   fields (`actor`, `intent`, `domain`, `event_id`), check embedded
   `ccnf_version` if present
3. **Canonicalize fields:** Deep-copy, normalize strings (NFC),
   convert timestamps, normalize number types, normalize null/absent
   fields
4. **Normalize intent:** Validate against controlled vocabulary,
   build normalized intent shape
5. **Check target_id:** Validate artifact reference syntax
6. **Resolve artifacts:** Extract artifacts from `payload.data`
7. **Derive identity:** Compute entity key, scope, collapse key
8. **Compute state deltas:** Build per-artifact delta entries
9. **Assemble CER:** Compose all fields into CER struct
10. **Compute signature:** Hash all fields except `signature`, attach
    signature block

---

## 9. Error Contract

### 9.1 Error Types

| Error Constant | HTTP Equivalent | Semantics |
|---|---|---|
| `STRUCTURAL_PARSE_FAILURE` | 400 | Invalid JSON, wrong root type, missing required field |
| `CCNF_VERSION_MISMATCH` | 400 | Version parameter != expected, or embedded version mismatch |
| `INTENT_NORMALIZATION_FAILURE` | 422 | Free-text intent, empty action, unknown action |
| `ARTIFACT_RESOLUTION_FAILURE` | 422 | Invalid artifact ID syntax, non-object artifact value |
| `DELTA_SCOPE_VIOLATION` | 422 | Artifact prefix outside target scope |

### 9.2 Required Fields

The CCNF pipeline requires these fields at the top level:

- `actor` (object)
- `intent` (object or string that will be rejected)
- `domain` (string)
- `event_id` (string)

---

## 10. Invariants

### I1 — Determinism
Given identical input bytes and CCNF version, every compliant
implementation produces identical CER bytes.

### I2 — Idempotency
Running the same input through the pipeline twice yields the same
output (same hash, same entity key, same CER).

### I3 — Order Independence
The entity key depends only on `actor`, `domain`, `intent`, and
`scope`.  It does NOT depend on:
- `timestamp`
- `event_id`
- `causality`
- `payload.data`
- `artifact_refs` ordering
- `state_delta` values

### I4 — Non-Collision
Syntactically different inputs that differ in any entity-key-relevant
field produce distinct entity keys.

### I5 — Serialization Stability
For any semantically equivalent input (differing only in key ordering,
whitespace, or character encoding forms), the pipeline produces
identical CER bytes.
