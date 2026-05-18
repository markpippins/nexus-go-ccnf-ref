# ReplayBinding: Cryptographic Execution Identity

**Phase**: R10.1  
**Status**: Ratified  
**Immutability**: This spec is frozen at v0.1.0-ccnf. Changes require CCNF version increment.

## 1. Purpose

`replay_binding_hash` is the field that converts an `ExecutionReceipt` from a proof of
execution into a proof of **reproducible** execution.

Before R10.1, a receipt certifies:
> This execution happened.

After R10.1, a receipt also certifies:
> This exact computation is replayable anywhere, given the same inputs.

This is the first field that makes execution **cryptographically addressable** across
independent runtimes, machines, and verifiers.

## 2. Field Definition

`replay_binding_hash` is a field on `ExecutionReceipt`:

| Field | Type | Required | Description |
|---|---|---|---|
| `replay_binding_hash` | `string` | conditional | SHA256 hex digest binding (cer_root, trace_root, version context). Required for `SUCCESS` and `PARTIAL`. Optional for `FAILURE`. |

### 2.1. Failure-State Weakening Rule

- `SUCCESS`: MUST be present. MUST be non-empty. MUST be a valid 64-char hex string.
- `PARTIAL`: MUST be present. MUST be non-empty. MUST be a valid 64-char hex string.
- `FAILURE`: MAY be absent or empty. MUST NOT be validated.

## 3. Hash Algorithm

`replay_binding_hash` SHALL be computed using a domain-separated hash context.

### 3.1. Domain Tag

```
REPLAY_DOMAIN = SHA256("ccnf-domain:replay-binding:v1")
```

Pre-computed once. Immutable. No other hash in the CCNF system may use this domain.

### 3.2. Input Ordering and Byte Encoding

```
replay_binding_hash = SHA256(
    REPLAY_DOMAIN                         // 32 bytes, pre-computed
    || BE64(len(cer_root_hash_raw))       // 8 bytes
    || cer_root_hash_raw                  // 32 bytes
    || BE64(len(trace_root_hash_raw))     // 8 bytes
    || trace_root_hash_raw                // 32 bytes
    || BE64(semantics_version)            // 8 bytes
    || BE64(ccnf_version)                 // 8 bytes
)
```

### 3.3. Encoding Rules

| Element | Encoding |
|---|---|
| `cer_root_hash` | raw bytes (not hex), length-prefixed with BE64 |
| `trace_root_hash` | raw bytes (not hex), length-prefixed with BE64 |
| `semantics_version` | big-endian uint64 |
| `ccnf_version` | big-endian uint64 |
| concatenation | strict ordering as specified |
| intermediate encoding | binary only, no JSON |

### 3.4. Injectivity Constraint

Any change in any input field MUST produce a different hash with overwhelming
probability. This includes:

- `cer_root_hash` (different MER root → different binding)
- `trace_root_hash` (different trace → different binding)
- `semantics_version` (different runtime semantics → different binding)
- `ccnf_version` (different pipeline version → different binding)

## 4. Invariants

- **RR1** (Presence): `replay_binding_hash` must be non-empty for `SUCCESS` and `PARTIAL`.
- **RR2** (Failure Weakening): `replay_binding_hash` is optional for `FAILURE`.
  May be empty or absent. Must not be validated.
- **RR3** (Re-computability): Reserved for R10.2+. A verifier given the receipt
  and the same inputs must be able to recompute `replay_binding_hash` and
  confirm equality.

## 5. Construction

`replay_binding_hash` MUST be produced by the single function:

```
ComputeReplayBinding(cerRootHash, traceRootHash, semanticsVersion, ccnfVersion) → string
```

`BuildReceipt` MUST call this function. The caller MUST NOT be able to inject
`replay_binding_hash` directly.

`ValidateReceipt` MUST NOT recompute the hash (R10.1 stage). Validation checks
format only: non-empty presence for `SUCCESS`/`PARTIAL`.

## 6. Version Sensitivity

Changing either `semantics_version` or `ccnf_version` produces a different
`replay_binding_hash`. This guarantees:

- Future replay engine changes (via `semantics_version` bump) cannot produce
  the same binding as the original execution.
- Pipeline version upgrades (via `ccnf_version` bump) cannot produce the
  same binding as the original.

This preserves replay isolation across versions without ambiguity.

## 7. Golden Test Vector

Both Go and Rust implementations MUST produce identical output for:

```
cer_root_hex     = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
trace_root_hex   = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
semantics_ver    = 1
ccnf_ver         = 1

Expected: replay_binding_hash = <hex output, identical in Go and Rust>
```

(Note: The exact expected value is defined by running the Go implementation
and hardcoding the result in a cross-runtime test.)

## 8. Cross-Runtime Equivalence

The Rust verifier MUST implement `compute_replay_binding` with byte-identical
encoding:

- Same domain tag string: `"ccnf-domain:replay-binding:v1"`
- Same SHA256 digest for the domain
- Same BE64 length-prefixed raw bytes
- Same hex output (lowercase)

Rust MUST NOT reinterpret, normalize, or "improve" the encoding rules.

## 9. Passivity

R10.1 introduces no behavioral change to execution:

- CCNF pipeline unchanged
- Trace construction unchanged
- CER construction unchanged
- Replay does not exist yet

`replay_binding_hash` is an opaque value on the receipt. No downstream system
reads or interprets it until R10.2.

## 10. Error Codes

| Code | Condition |
|---|---|
| `INVALID_RECEIPT: replay_binding_hash must be non-empty for SUCCESS/PARTIAL` | RR1 violation |

## 11. R10.2 Forward Reservation

`replay_binding_hash` exists at R10.1 only to stabilize the receipt identity.
At R10.2, it becomes the anchor point for replay verification:

```
replay(trace, cer) → RuntimeState
VerifyReplayBinding(receipt, replayOutput) → bool
```

The R10.2 replay engine MUST NOT recompute `replay_binding_hash`. It is an
opaque input consumed but not derived. See R10.2 boundary invariants.
