# Versioning Model — Contract and Migration Policy

**Version:** 1
**Status:** Immutable (v0.1.0-ccnf)

---

## 1. Introduction

The system uses three independent version numbers to guard against
cross-deployment drift.  This document defines the semantics of each
version, the tri-version lock contract, and the migration rules.

---

## 2. Version Numbers

### 2.1 CCNF Version

| Name | Current | Scope |
|---|---|---|
| `ccnf_version` | 1 | CCNF pipeline + CER schema |

The CCNF version changes when:
- A field is added to, removed from, or renamed in the CER schema
- A serialization rule changes (§CCNF_SPEC §3)
- A normalization rule changes (§CCNF_SPEC §4)
- The controlled vocabulary changes (§CCNF_SPEC §4.5)
- The entity key derivation changes (§CCNF_SPEC §5)
- The hash algorithm changes (§CCNF_SPEC §7)

### 2.2 Collapse Version

| Name | Current | Scope |
|---|---|---|
| `collapse_version` | 1 | Identity collapse algorithm |

The collapse version changes when:
- Identity Rules 2 or 3 are modified (collapse semantics)
- The alias resolution strategy changes
- The semantic collapse boundary changes

### 2.3 Rehydration Version

| Name | Current | Scope |
|---|---|---|
| `rehydration_version` | 1 | Replay state derivation |

The rehydration version changes when:
- The fold algorithm changes (§REPLAY_SPEC §3)
- The delta merge semantics change (§REPLAY_SPEC §3.4)
- The entity isolation rules change
- The cursor model changes (§REPLAY_SPEC §4)

---

## 3. Tri-Version Lock

### 3.1 Contract

A snapshot is valid ONLY if:

```
ccnf_version == collapse_version == rehydration_version
```

AND none of the three is zero.

### 3.2 Enforcement

Validation (§SNAPSHOT_SPEC §4.2) proceeds in this order:

1. **Zero check:** If any version is 0 → `TRI_VERSION_LOCK_FAILURE`
2. **Equality check:** If any pair differs → `TRI_VERSION_LOCK_FAILURE` with detail

### 3.3 Rationale

The tri-version lock prevents silent inconsistency when components
are upgraded independently.  Consider this scenario:

| Component | Version |
|---|---|
| CCNF pipeline | 2 (new collapse boundary) |
| Snapshot on disk | ccnf: 2, collapse: 1, rehydration: 1 |

Without the lock, the old snapshot would be rehydrated using the new
collapse rules, producing incorrect state.  The lock detects this and
rejects the snapshot.

### 3.4 Invalid Lock

An implementation encountering a tri-version lock violation MUST:
1. Reject the snapshot
2. Return `TRI_VERSION_LOCK_FAILURE`
3. NOT fall back to any "best effort" reconstruction

---

## 4. Version Epochs

### 4.1 Epoch 1 (v0.1.0-ccnf)

| Version | Value |
|---|---|
| `ccnf_version` | 1 |
| `collapse_version` | 1 |
| `rehydration_version` | 1 |

This is the initial frozen epoch.  All golden vectors, collision
atlases, and test suites are locked to these versions.

### 4.2 Future Epochs

A future epoch bumps one or more version numbers.  The rules are:

- **Minor bump** (one version changes): the tri-version lock requires
  a coordinated deployment of the changed component.
- **Major bump** (all three change): a full system migration is
  required.  No cross-epoch compatibility is guaranteed.

---

## 5. Migration Rules

### 5.1 Cross-Version Comparison

```
It is invalid to compare a CER at version N with a CER at version M
where N != M.
```

Rationale: The entity key derivation includes the normalized intent
and scope, both of which may change between versions.  Entity keys
from different epochs are structurally incompatible.

### 5.2 Snapshot Rejection

A snapshot whose tri-version lock is invalid MUST be rejected.  The
only recovery path is:

1. Replay the original event log from the beginning
2. Build a new snapshot at the current version
3. Validate the new snapshot

### 5.3 Golden Vector Invalidation

When a version changes:
- All golden vectors for the new version MUST be generated from the
  reference oracle.
- Existing vectors for the old version are preserved but marked as
  superseded.
- The collision atlas MUST be regenerated.
- The hash-lock table MUST be regenerated.

### 5.4 CI Gate

Every version change requires:
1. Update version constants in the reference implementation
2. Regenerate golden vectors
3. Regenerate collision atlas
4. Update hash-lock table
5. Run full R1–R6 CI gate
6. Tag the new version

---

## 6. Immutability Guarantee

### 6.1 v0.1.0-ccnf

Version 1 is frozen at commit `bcb6200` (tag `v0.1.0-ccnf`).  No
changes will be made to:
- CER schema (15 top-level fields, all sub-fields)
- Serialization rules (9 immutable rules)
- Normalization rules (NFC, controlled vocabulary, timestamps)
- Entity key derivation (4-field SHA256)
- State delta computation (per-artifact SHA256)
- Hash algorithm (SHA256, hex-encoded)

### 6.2 v0.1.0-fuzz

The collision atlas is frozen at tag `v0.1.0-fuzz`.  It certifies
87,500 valid inputs with 0 collisions, 0 ambiguities, and 0
divergences across 8 mutation dimensions.

---

## 7. Constants Reference

| Constant | Current Value | Defined In |
|---|---|---|
| `CurrentCCNFVersion` | 1 | CCNF_SPEC.md, CER_SPEC.md |
| `CurrentEventVersion` | 1 | CER_SPEC.md |
| `CurrentCollapseVersion` | 1 | SNAPSHOT_SPEC.md |
| `CurrentRehydrationVersion` | 1 | SNAPSHOT_SPEC.md |
| `SystemName` | `"nexus"` | CER_SPEC.md §2.2 |

---

## 8. Invariants

### V1 — Cross-Version Incomparability
CERs at different `ccnf_version` values MUST NOT be compared or
equated.

### V2 — Lock Before Use
Every snapshot MUST pass tri-version lock validation before its state
may be used for reconstruction.

### V3 — No Fallback
A tri-version lock violation has no fallback or best-effort recovery.
The snapshot is unconditionally rejected.

### V4 — Epoch Atomicity
All three versions in a given epoch have the same numeric value.
They may diverge temporarily during migration but MUST be equal at
rest.
