# ExecutionReceipt: Egress Boundary Contract

**Phase**: R9.2  
**Status**: Ratified  
**Immutability**: This spec is frozen at v0.1.0-ccnf. Changes require CCNF version increment.

## 1. Purpose

`ExecutionReceipt` is the only legal egress object from the CCNF pipeline. It
provides a formal proof-of-execution record linking a specific `ExecutionRequest`
to its resulting CER, trace commitment, status, and timing. Every invocation of
the pipeline produces exactly one `ExecutionReceipt`.

## 2. Schema

| Field | Type | Required | Description |
|---|---|---|---|---|
| `request_id` | `string` | yes | Links back to the originating `ExecutionRequest.request_id`. |
| `ccnf_hash` | `string` | conditional | SHA256 hex digest of the canonicalized CER (excluding signature). Required for `SUCCESS` and `PARTIAL`; empty for `FAILURE`. |
| `cer_root_hash` | `string` | conditional | Root hash of the CER chain. For R9.2 (single-event), equals `trace_root_hash`. Reserved for Merkle chain aggregation in future versions. |
| `trace_root_hash` | `string` | conditional | Domain-separated Merkle commitment over the execution trace (see EXECUTION_TRACE_SPEC.md). Required for `SUCCESS` and `PARTIAL`; empty for `FAILURE`. |
| `trace_event_count` | `uint64` | yes | Number of events appended to the trace. Must be > 0 for `SUCCESS` and `PARTIAL`; must be 0 for `FAILURE`. |
| `replay_binding_hash` | `string` | conditional | Domain-separated binding of (cer_root, trace_root, semantics_version, ccnf_version). Required for `SUCCESS` and `PARTIAL`; optional for `FAILURE` (see REPLAY_BINDING_SPEC.md). |
| `status` | `ExecutionStatus` | yes | One of `SUCCESS`, `FAILURE`, `PARTIAL`. |
| `failure` | `FailureNode` | no | Present only when `status` is `FAILURE` or `PARTIAL`. Must be absent when `status` is `SUCCESS`. |
| `timing` | `Timing` | yes | Wall-clock timing for the execution. |
| `ccnf_version` | `int` | yes | CCNF version used during this execution. |

### 2.1. ExecutionStatus

| Value | Semantics |
|---|---|
| `SUCCESS` | Pipeline completed normally. CER emitted. Trace committed. All consistency checks passed. |
| `FAILURE` | Pipeline aborted before producing a CER. No trace was generated. No state was modified. |
| `PARTIAL` | CER and trace were emitted but a downstream consistency check failed (e.g. snapshot mismatch). The CER and trace may still be valid. |

### 2.2. FailureNode

| Field | Type | Required | Description |
|---|---|---|---|
| `code` | `string` | yes | Machine-readable error code. Uses CCNF error constants where applicable. |
| `message` | `string` | yes | Human-readable description of the failure. |
| `cause` | `FailureNode` | no | Recursive cause chain for nested failures. |

### 2.3. Timing

| Field | Type | Required | Description |
|---|---|---|---|
| `started_at` | `int64` | yes | Unix epoch nanoseconds when execution began. |
| `completed_at` | `int64` | yes | Unix epoch nanoseconds when execution completed. |
| `duration_ms` | `int64` | yes | Wall-clock duration in milliseconds. Must be >= 0. |

## 3. Invariants

- **IR1**: `request_id` must be non-empty and must match the originating `ExecutionRequest.request_id`.
- **IR2**: `status` must be one of the three defined values.
- **IR3**: If `status` is `SUCCESS`, `failure` must be nil. If `status` is `FAILURE` or
  `PARTIAL`, `failure` must be non-nil.
- **IR4**: `duration_ms` must be >= 0.
- **IR5**: `ccnf_hash` must be a 64-character hex string (SHA256) when `status` is
  `SUCCESS` or `PARTIAL`. Must be empty when `status` is `FAILURE`.
- **IR6**: `timing.started_at` must be <= `timing.completed_at` (both set and non-zero).
- **TR1** (Trace Commitment): `trace_root_hash` must be non-empty for `SUCCESS` and
  `PARTIAL`. Must be a valid 64-character hex string.
- **TR2** (Replay Equivalence): Reserved for R9.3+. `replay(trace)` must produce a
  runtime state whose `snapshot_hash` equals `receipt.final_state_hash`.
- **TR3** (Completeness): `trace_event_count` must be > 0 for `SUCCESS` and `PARTIAL`
  (at least one CER was emitted). Must be 0 for `FAILURE` (no CER was emitted).
- **RR1** (Replay Binding Presence): `replay_binding_hash` must be non-empty for
  `SUCCESS` and `PARTIAL`.
- **RR2** (Failure Weakening): `replay_binding_hash` is optional for `FAILURE`.
  May be empty or absent. MUST NOT be validated.
- **RR3** (Re-computability): Reserved for R10.2+. A verifier can recompute
  `replay_binding_hash` from the receipt's other fields and inputs.

## 4. Validation Pseudocode

```
func ValidateReceipt(rec):
    if rec.request_id == "":
        return error("INVALID_RECEIPT: request_id must be non-empty")
    if rec.timing.duration_ms < 0:
        return error("INVALID_RECEIPT: duration_ms must be >= 0")

    switch rec.status:
        case SUCCESS:
            if rec.failure != nil:
                return error("INVALID_RECEIPT: failure node must be nil for SUCCESS")
            if rec.ccnf_hash == "":
                return error("INVALID_RECEIPT: ccnf_hash must be non-empty for SUCCESS")
            if rec.trace_root_hash == "":
                return error("INVALID_RECEIPT: trace_root_hash must be non-empty for SUCCESS")
            if rec.trace_event_count == 0:
                return error("INVALID_RECEIPT: trace_event_count must be > 0 for SUCCESS")
            if rec.replay_binding_hash == "":
                return error("INVALID_RECEIPT: replay_binding_hash must be non-empty for SUCCESS")

        case FAILURE:
            if rec.failure == nil:
                return error("INVALID_RECEIPT: failure node must be non-nil for FAILURE")
            if rec.trace_event_count != 0:
                return error("INVALID_RECEIPT: trace_event_count must be 0 for FAILURE")

        case PARTIAL:
            if rec.failure == nil:
                return error("INVALID_RECEIPT: failure node must be non-nil for PARTIAL")
            if rec.ccnf_hash == "":
                return error("INVALID_RECEIPT: ccnf_hash must be non-empty for PARTIAL")
            if rec.trace_root_hash == "":
                return error("INVALID_RECEIPT: trace_root_hash must be non-empty for PARTIAL")
            if rec.trace_event_count == 0:
                return error("INVALID_RECEIPT: trace_event_count must be > 0 for PARTIAL")
            if rec.replay_binding_hash == "":
                return error("INVALID_RECEIPT: replay_binding_hash must be non-empty for PARTIAL")

    return nil
```

## 5. Construction Pseudocode

```
func BuildReceipt(req, status, ccnfHash, trace, failure, timing, ccnfVersion):
    traceRootHash = trace.RootHash()

    replayBindingHash = ""
    if status == SUCCESS or status == PARTIAL:
        replayBindingHash = ComputeReplayBinding(
            traceRootHash,            // cer_root_hash (single-chain = trace)
            traceRootHash,            // trace_root_hash
            CurrentRuntimeSemanticsVersion,
            ccnfVersion,
        )

    return ExecutionReceipt {
        request_id:          req.request_id,
        ccnf_hash:           ccnfHash,
        cer_root_hash:       traceRootHash,    // single-chain phase
        trace_root_hash:     traceRootHash,
        trace_event_count:   trace.EventCount(),
        replay_binding_hash: replayBindingHash,
        status:              status,
        failure:             failure,
        timing:              timing,
        ccnf_version:        ccnfVersion
    }
```

## 6. Trace Completeness Proof (Sketch)

For any execution:

1. A single `ExecutionRequest` produces exactly one `ExecutionReceipt`.
2. The receipt's `request_id` anchors the trace to its origin.
3. The `trace_root_hash` cryptographically commits to every CER produced.
4. The `trace_event_count` proves how many CERs were produced.
5. If `status == SUCCESS`, the `ccnf_hash` identifies the final CER.
6. If `status == FAILURE`, no CER exists and `trace_event_count == 0`.
7. If `status == PARTIAL`, the CER exists but may be inconsistent with downstream state.

This forms a **causal chain**: `Request → Trace → Receipt → (CER | FailureNode)` — no
execution can produce an untracked CER, and no receipt exists without a request.

## 7. Error Codes

| Code | Condition |
|---|---|
| `INVALID_RECEIPT: request_id must be non-empty` | IR1 violation |
| `INVALID_RECEIPT: duration_ms must be >= 0` | IR4 violation |
| `INVALID_RECEIPT: failure node must be nil for SUCCESS` | IR3 violation (success case) |
| `INVALID_RECEIPT: failure node must be non-nil for FAILURE` | IR3 violation (failure case) |
| `INVALID_RECEIPT: failure node must be non-nil for PARTIAL` | IR3 violation (partial case) |
| `INVALID_RECEIPT: ccnf_hash must be non-empty for SUCCESS` | IR5 violation (success case) |
| `INVALID_RECEIPT: ccnf_hash must be non-empty for PARTIAL` | IR5 violation (partial case) |
| `INVALID_RECEIPT: trace_root_hash must be non-empty for SUCCESS` | TR1 violation (success case) |
| `INVALID_RECEIPT: trace_root_hash must be non-empty for PARTIAL` | TR1 violation (partial case) |
| `INVALID_RECEIPT: trace_event_count must be > 0 for SUCCESS` | TR3 violation (success case) |
| `INVALID_RECEIPT: trace_event_count must be > 0 for PARTIAL` | TR3 violation (partial case) |
| `INVALID_RECEIPT: trace_event_count must be 0 for FAILURE` | TR3 violation (failure case) |
| `INVALID_RECEIPT: replay_binding_hash must be non-empty for SUCCESS` | RR1 violation (success case) |
| `INVALID_RECEIPT: replay_binding_hash must be non-empty for PARTIAL` | RR1 violation (partial case) |

## 8. Example: SUCCESS

```json
{
  "request_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "ccnf_hash": "abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
  "cer_root_hash": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1",
  "trace_root_hash": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1",
  "trace_event_count": 1,
  "replay_binding_hash": "d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5",
  "status": "SUCCESS",
  "timing": {
    "started_at": 1700000000000000000,
    "completed_at": 1700000000500000000,
    "duration_ms": 5
  },
  "ccnf_version": 1
}
```

## 9. Example: FAILURE

```json
{
  "request_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
  "ccnf_hash": "",
  "cer_root_hash": "",
  "trace_root_hash": "",
  "trace_event_count": 0,
  "replay_binding_hash": "",
  "status": "FAILURE",
  "failure": {
    "code": "INTENT_NORMALIZATION_FAILURE",
    "message": "unknown intent action 'destroy'",
    "cause": {
      "code": "STRUCTURAL_PARSE_FAILURE",
      "message": "field 'intent.action' has invalid value"
    }
  },
  "timing": {
    "started_at": 1700000000000000000,
    "completed_at": 1700000000300000000,
    "duration_ms": 3
  },
  "ccnf_version": 1
}
```

## 10. Example: PARTIAL

```json
{
  "request_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
  "ccnf_hash": "def789abc012def789abc012def789abc012def789abc012def789abc012def7",
  "cer_root_hash": "b2c3d4e5f6a7b2c3d4e5f6a7b2c3d4e5f6a7b2c3d4e5f6a7b2c3d4e5f6a7",
  "trace_root_hash": "b2c3d4e5f6a7b2c3d4e5f6a7b2c3d4e5f6a7b2c3d4e5f6a7b2c3d4e5f6a7",
  "trace_event_count": 1,
  "replay_binding_hash": "e5f6a7b2c3d4e5f6a7b2c3d4e5f6a7b2c3d4e5f6a7b2c3d4e5f6a7b2c3d4e5f6a7",
  "status": "PARTIAL",
  "failure": {
    "code": "DOWNSTREAM_CONSISTENCY_FAILURE",
    "message": "snapshot verification failed: tri-version lock mismatch"
  },
  "timing": {
    "started_at": 1700000000000000000,
    "completed_at": 1700000001000000000,
    "duration_ms": 10
  },
  "ccnf_version": 1
}
```
