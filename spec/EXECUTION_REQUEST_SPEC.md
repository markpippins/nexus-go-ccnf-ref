# ExecutionRequest: Ingress Boundary Contract

**Phase**: R9.1  
**Status**: Ratified  
**Immutability**: This spec is frozen at v0.1.0-ccnf. Changes require CCNF version increment.

## 1. Purpose

`ExecutionRequest` is the only legal ingress object for the CCNF pipeline. It wraps raw
input with a traceable, versioned envelope. No other object type may enter the pipeline
from outside the system boundary.

## 2. Schema

| Field | Type | Required | Description |
|---|---|---|---|
| `request_id` | `string` | yes | Globally unique, immutable trace anchor. Canonical format: UUID v4. |
| `version` | `VersionTriple` | yes | Tri-version lock specification (see VERSIONING_MODEL.md). |
| `timestamp` | `int64` | yes | Unix epoch seconds. Immutable after creation. |
| `source` | `string` | yes | Origin identifier (e.g. service name, host, client ID). |
| `payload` | `object` | yes | The actual CCNF input document. Content is validated by the CCNF pipeline (R1-R8). |
| `metadata` | `object` | no | Optional metadata bag. Not processed by CCNF pipeline. |

### 2.1. VersionTriple

| Field | Type | Required | Description |
|---|---|---|---|
| `ccnf` | `int` | yes | CCNF version this request targets. Must match the pipeline's `CurrentCCNFVersion`. |
| `collapse_engine` | `int` | yes | Collapse engine version for read-time semantic collapse. |
| `rehydration` | `int` | yes | Rehydration version for state reconstruction. |

## 3. Invariants

- **ER1**: `request_id` must be non-empty. Once set, it is immutable for the
  lifetime of the execution.
- **ER2**: `timestamp` must be > 0. It is set at creation time and never modified.
- **ER3**: All three `version` components must be > 0. Zero is not a valid version number
  in the CCNF system.
- **ER4**: `payload` must be present and non-null. The schema within `payload` is
  validated by the CCNF pipeline, not by the ingress boundary.

## 4. Validation Pseudocode

```
func ValidateRequest(req):
    if req.request_id == "":
        return error("INVALID_REQUEST: request_id must be non-empty")
    if req.timestamp <= 0:
        return error("INVALID_REQUEST: timestamp must be > 0")
    if req.version.ccnf <= 0 or req.version.collapse_engine <= 0 or req.version.rehydration <= 0:
        return error("INVALID_REQUEST: version triple components must all be > 0")
    return nil
```

## 5. Immutability Contract

`ExecutionRequest` is immutable after ingress. No field may be modified once the
request has been submitted to the CCNF pipeline. This is a spec-level contract;
type-level enforcement is deferred to R9.2+.

## 6. Example

```json
{
  "request_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "version": {
    "ccnf": 1,
    "collapse_engine": 1,
    "rehydration": 1
  },
  "timestamp": 1700000000,
  "source": "nexus-cli",
  "payload": {
    "ccnf_version": 1,
    "system": "nexus",
    "domain": "test",
    "event_id": "evt-001",
    "intent": { "action": "create", "target_id": "entity:test-001" },
    "actor": { "type": "service", "id": "orchestrator" },
    "timestamp": 1700000000
  }
}
```

## 7. Error Codes

Ingress validation produces codes prefixed with `INVALID_REQUEST`:

| Code | Condition |
|---|---|
| `INVALID_REQUEST` | Generic validation failure |
| `INVALID_REQUEST: request_id must be non-empty` | ER1 violation |
| `INVALID_REQUEST: timestamp must be > 0` | ER2 violation |
| `INVALID_REQUEST: version triple components must all be > 0` | ER3 violation |
