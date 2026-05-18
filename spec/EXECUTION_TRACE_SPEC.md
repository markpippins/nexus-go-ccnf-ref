# ExecutionTrace: Cryptographic Witness

**Phase**: R9.2  
**Status**: Ratified  
**Immutability**: This spec is frozen at v0.1.0-ccnf. Changes require CCNF version increment.

## 1. Purpose

The execution trace is a **cryptographic witness** that computation occurred. It is not
a log, not an audit trail, and not a record for human inspection. It is a mathematical
proof: given an `ExecutionRequest` and the trace's root hash, any verifier can confirm
that a specific sequence of CER hashes was produced by the pipeline.

The trace exists only to anchor the execution's output to a verifiable commitment.
It has no runtime function.

## 2. TraceBuilder API

`TraceBuilder` is the exclusive writer of trace data. It exposes three operations:

| Operation | Input | Visibility |
|---|---|---|
| `Append(cerHashHex)` | 64-char hex SHA256 | Write-only |
| `RootHash()` | — | Terminal read (after all writes) |
| `EventCount()` | — | Terminal read |

### 2.1. Append

Records a single CER hash into the trace. The hash must be a valid 64-character
hex-encoded SHA256 digest. Invalid input causes an immediate panic (fail-fast).

### 2.2. RootHash

Computes and returns the domain-separated Merkle commitment over all appended
event hashes. Called exactly once after all `Append` calls have completed.

### 2.3. EventCount

Returns the number of events appended. Used for completeness verification (TR3).

## 3. Passivity Rule

**The runtime must not read the trace while executing.**

The `TraceBuilder` enforces this by API shape:

- No method exposes individual events
- No method allows iteration
- No method allows indexed access
- `RootHash()` is the only read operation and collapses the entire trace

This prevents **observer effects** — the execution cannot branch on trace content,
which would introduce nondeterminism.

## 4. Hash Algorithm

The trace root hash SHALL be computed using a domain-separated hash context.

### 4.1. Domain Tag

```
TRACE_DOMAIN = SHA256("ccnf-domain:trace:v1")
```

The domain tag is a pre-hashed prefix that prevents cross-object hash confusion.
No other hash in the CCNF system may use this domain tag. Future trace format
changes use `ccnf-domain:trace:v2`, etc.

### 4.2. Trace Root Computation

```
trace_root = SHA256(
    TRACE_DOMAIN        // 32 bytes, pre-computed
    event_count_be64    // 8 bytes, big-endian uint64
    hash_1              // 32 bytes, raw SHA256
    hash_2              // 32 bytes, raw SHA256
    ...
    hash_n              // 32 bytes, raw SHA256
)
```

### 4.3. Properties

- **Collision isolation**: Domain tag prevents hash equivalence with CER hashes,
  snapshot hashes, entity keys, or any other hash in the system.
- **Determinism**: Same inputs always produce the same trace root, across
  languages, platforms, and compiler versions.
- **Extensibility**: Future event types (non-CER) can be added without changing
  the hash structure for existing traces.

### 4.4. Single-Event Case

For R9.2 (single CER per execution):

```
trace_root != cer_hash          // domain separation ensures this
trace_root == cer_root_hash     // reserved field, same value in single-chain
```

## 5. Invariants

- **TR1 (Trace Commitment)**: `ExecutionReceipt.trace_root_hash` must equal
  `TraceBuilder.RootHash()` computed from the same event sequence.
- **TR2 (Replay Equivalence)**: Reserved for R9.3+. `replay(trace)` must produce
  a runtime state whose `snapshot_hash` equals `receipt.final_state_hash`.
- **TR3 (Completeness)**: `trace_event_count` must be > 0 for `SUCCESS` and
  `PARTIAL` receipts (at least one CER was emitted). Must be 0 for `FAILURE`
  receipts (no CER was emitted).

## 6. Determinism Requirement

Given the same sequence of CER hashes (same count, same order, same values),
`TraceBuilder.RootHash()` MUST produce the same output on every invocation,
across all implementations (Go, Rust, future).

This is enforced by:

1. Fixed domain tag (single SHA256 pre-image, no variable encoding)
2. Fixed byte ordering (big-endian for count)
3. Fixed hash function (SHA256)
4. Fixed output encoding (lowercase hex)

## 7. Replay Binding (Reserved for R9.3+)

The trace root hash is the commitment that enables replay binding:

```
replay(trace) → RuntimeSnapshot
snapshot_hash == receipt.final_state_hash
```

This section is reserved and will be specified in R9.3.

## 8. Error Codes

| Code | Condition |
|---|---|
| `INVALID_RECEIPT: trace_root_hash must be non-empty for SUCCESS/PARTIAL` | TR1 violation |
| `INVALID_RECEIPT: trace_event_count must be > 0 for SUCCESS/PARTIAL` | TR3 violation (success/partial) |
| `INVALID_RECEIPT: trace_event_count must be 0 for FAILURE` | TR3 violation (failure) |

## 9. Example

Given two CER hashes:

```
hash_1 = "a000...001"
hash_2 = "a000...002"
```

The trace root is:

```
TRACE_DOMAIN = SHA256("ccnf-domain:trace:v1")
count        = 00 00 00 00 00 00 00 02
trace_root   = SHA256(TRACE_DOMAIN || count || hash_1_raw || hash_2_raw)
```
