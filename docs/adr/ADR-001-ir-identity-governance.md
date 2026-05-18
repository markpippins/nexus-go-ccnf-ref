# ADR-001: IR Identity Governance & Controlled Change Protocol

**Status:** Accepted — Phase 2 Freeze Active

## Context

The PGV system derives a deterministic Intermediate Representation (IR)
from graph inputs using:

- `deriveParent`
- `hashNode`
- `ToIR` projection

A golden snapshot test (`TestGoldenGraphToIR`) now locks the behavioral
output of this projection.

Prior to this ADR, IR identity evolution could occur implicitly through
implementation changes.

This created risk of:

- silent identity drift
- non-reproducible diffs
- unstable policy trace history
- accidental CLI overbuild during Phase 2

## Decision

Phase 2 establishes a Frozen IR Projection Contract.

Structural behavior affecting IR identity SHALL NOT change without
following the Controlled Change Protocol defined below.

### Phase Definitions

#### Phase 2 — Frozen Behavioral Truth

Allowed:
- tests
- validation wiring
- machine-mode plumbing
- non-structural refactors

Forbidden:
- changing `deriveParent`
- changing `hashNode`
- modifying IR schema
- altering diff identity semantics

Golden snapshot failure is intentional enforcement.

#### Phase 3 — Identity Evolution

Structural change becomes allowed only after:
1. Explicit intent declaration
2. Dual-run comparison
3. Reviewed rebaseline

### Controlled Change Protocol

#### Step 1 — Intent Declaration

Commit message:

```
PGV: IR Projection Contract Change — <component>
```

Must describe:
- change scope
- expected IR drift
- rebaseline expectation

#### Step 2 — Dual Implementation

Introduce parallel implementation:
- `deriveParent_v2`
- `hashNode_v2`
- `ToIR_v2`

Existing behavior MUST remain intact.

#### Step 3 — Dual Run Validation

System executes:
- old IR → baseline
- new IR → candidate

Produces:
- IRDelta comparison
- identity divergence report

No golden update allowed yet.

#### Step 4 — Explicit Rebaseline

Single commit:

```
PGV: IR Golden Snapshot Rebaseline (<reason>)
```

Includes:
- updated golden snapshot
- migration rationale

#### Step 5 — Contract Lock

New behavior becomes frozen Phase-2 truth.

### Non-Negotiable Rules

- No silent golden updates.
- No inline identity changes.
- No partial migrations.
- CI failure is governance, not error.

## Consequences

### Positive

- Deterministic identity evolution
- Stable policy trace lineage
- Safe CLI development separation
- Reviewable architectural history

### Negative

- Slower structural iteration
- Requires explicit migration commits

## Relationship to Future ADRs

Referenced by:
- ADR-00Z Policy Trace Overlay
- StableID introduction
- Machine-mode diff contract
- Identity normalization work
