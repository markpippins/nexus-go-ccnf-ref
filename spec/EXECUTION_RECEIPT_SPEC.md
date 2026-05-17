# ExecutionReceipt: Egress Boundary Contract

**Phase**: R9.1  
**Status**: Ratified  
**Immutability**: This spec is frozen at v0.1.0-ccnf. Changes require CCNF version increment.

## 1. Purpose

`ExecutionReceipt` is the only legal egress object from the CCNF pipeline. It
provides a formal proof-of-execution record linking a specific `ExecutionRequest`
to its resulting CER, status, and timing. Every invocation of the pipeline produces
exactly one `ExecutionReceipt`.

## 2. Schema

| Field | Type | Required | Description |
|---|---|---|---|
| `request_id` | `string` | yes | Links back to the originating `ExecutionRequest.request_id`. |
| `ccnf_hash` | `string` | conditional | SHA256 hex digest of the canonicalized CER (excluding signature). Required for `SUCCESS` and `PARTIAL`; empty for `FAILURE`. |
| `cer_root_hash` | `string` | conditional | Root hash of the CER chain. For R9.1 (single-event), equals `ccnf_hash`. Reserved for Merkle chain aggregation in future versions. |
| `status` | `ExecutionStatus` | yes | One of `SUCCESS`, `FAILURE`, `PARTIAL`. |
| `failure` | `FailureNode` | no | Present only when `status` is `FAILURE` or `PARTIAL`. Must be absent when `status` is `SUCCESS`. |
| `timing` | `Timing` | yes | Wall-clock timing for the execution. |
| `ccnf_version` | `int` | yes | CCNF version used during this execution. |

### 2.1. ExecutionStatus

| Value | Semantics |
|---|---|
| `SUCCESS` | Pipeline completed normally. CER emitted. All consistency checks passed. |
| `FAILURE` | Pipeline aborted before producing a CER. No state was modified. |
| `PARTIAL` | CER was emitted but a downstream consistency check failed (e.g. snapshot mismatch). The CER may still be valid. |

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
        case FAILURE:
            if rec.failure == nil:
                return error("INVALID_RECEIPT: failure node must be non-nil for FAILURE")
        case PARTIAL:
            if rec.failure == nil:
                return error("INVALID_RECEIPT: failure node must be non-nil for PARTIAL")
            if rec.ccnf_hash == "":
                return error("INVALID_RECEIPT: ccnf_hash must be non-empty for PARTIAL")

    return nil
```

## 5. Construction Pseudocode

```
func BuildReceipt(req, status, ccnfHash, cerRootHash, failure, timing, ccnfVersion):
    return ExecutionReceipt {
        request_id:    req.request_id,
        ccnf_hash:     ccnfHash,
        cer_root_hash: cerRootHash,
        status:        status,
        failure:       failure,
        timing:        timing,
        ccnf_version:  ccnfVersion
    }
```

## 6. Trace Completeness Proof (Sketch)

For any execution:

1. A single `ExecutionRequest` produces exactly one `ExecutionReceipt`.
2. The receipt's `request_id` anchors the trace to its origin.
3. If `status == SUCCESS`, the `ccnf_hash` identifies the resulting CER.
4. If `status == FAILURE`, no CER exists (receipt is the terminal record).
5. If `status == PARTIAL`, the CER exists but may be inconsistent with downstream state.

This forms a **causal chain**: `Request → Receipt → (CER | FailureNode)` — no
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

## 8. Example: SUCCESS

```json
{
  "request_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "ccnf_hash": "abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
  "cer_root_hash": "abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
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
  "cer_root_hash": "def789abc012def789abc012def789abc012def789abc012def789abc012def7",
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
