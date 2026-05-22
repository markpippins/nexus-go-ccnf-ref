# ADR-002: StableID Introduction & Identity Normalization

**Status:** Draft — Phase 3 Intent Declaration

## Context

The PGV system currently derives IR node identity from `ImportPath`:

```
IRNode.ID = ImportPath
```

This is structurally deterministic but semantically fragile. A package rename
(e.g. `auth/service` → `identity/service`) produces DELETE+ADD in the diff
engine instead of MOVE. The system cannot distinguish "same thing, new
location" from "old thing gone, new thing appeared."

ADR-001 established the governance protocol for identity changes. This ADR
introduces the identity model that ADR-001 protects.

## Decision

StableID is a **registered LineageID**, not a derived content hash.
It represents semantic continuity, not structural coincidence.

```
StableID = HASH(OriginSeed + SemanticSignature)
```

### Components

| Component | Type | Source | Mutability |
|-----------|------|--------|------------|
| OriginSeed | UUIDv7 | identity registry, first insertion | immutable |
| SemanticSignature | SHA256(Name + sorted(DirectDeps)) | computed by `ToIR` | recomputed each run |
| StableID | SHA256(OriginSeed \|\| SemanticSignature) | assembled at registry boundary | evolves only via versioned migration |

#### OriginSeed

A UUIDv7 assigned at **first SemanticSignature observation** in the identity
registry layer. Never recomputed. Never derived again. Answers "where did this
identity begin?", not "what does this node contain?"

#### SemanticSignature

A deterministic structural fingerprint computed by `ToIR` from the node's
semantic content only:

```
SemanticSignature = SHA256(Name + sorted(DirectDeps))
```

`ToIR` remains pure and stateless — it computes this field without calling the
registry. The registry layer assembles the full StableID.

## Subsystem: runtime/identity/

StableID ownership lives in a new isolated subsystem:

```
runtime/identity/
├── doc.go             // Package documentation and invariants
├── registry.go        // Public interface: Resolve(SemanticSignature) StableID
├── store.go           // Append-only persistence: Get, Insert (no Update/Delete/Replace)
├── stableid.go        // Assemble(seed OriginSeed, sig SemanticSignature) StableID — pure function
├── replay.go          // Replay(events []IRSnapshot) error — historical validation
├── errors.go          // Sentinel errors (Ambiguous, NotFound, AlreadyExists)
└── internal/
    ├── canonicalize.go // SemanticSignature normalization
    └── hash.go         // StableID hashing internals
```

### Architectural role

`runtime/identity/` owns exactly three responsibilities:

| Responsibility | Allowed |
|----------------|---------|
| OriginSeed assignment | ✅ |
| StableID assembly | ✅ |
| Historical replay validation | ✅ |

Everything else is forbidden:

- ❌ IR construction
- ❌ Dependency graph traversal
- ❌ Runtime execution / rehydration
- ❌ Diff engine logic
- ❌ Policy evaluation

### Dependency direction

```
ToIR (pure)
  └── emits SemanticSignature
        └── Identity Registry (stateful, deterministic)
              ├── assigns OriginSeed
              └── assembles StableID
                    └── Diff / MOVE / lineage engine
                          └── Runtime / Rehydrate (may consume StableID)
```

**One-way only.** Identity must never depend on rehydration, execution, or
graph traversal. The identity registry is the temporal root of truth.

### Import firewall

`runtime/identity/` must be an inward leaf package. Allowed imports:

- `runtime/identity` → `tools/pgv/ir` (for `SemanticSignature` if shared)
- `runtime/identity` → standard library (`crypto/sha256`, `encoding/hex`, `uuid`)

Forbidden imports:

- `runtime/identity` → `runtime/rehydrate`
- `runtime/identity` → `tools/pgv/diff`
- `runtime/identity` → `tools/pgv/graph`

Enforced by CI guardrail (Guardrail 1).

### Write-once store

The identity store is append-only. Allowed operations:

- `Get(sig SemanticSignature) (OriginSeed, bool)` — read
- `Insert(sig SemanticSignature, seed OriginSeed)` — write-once

Forbidden operations (CI-enforced):

- `Update`, `Delete`, `Replace`, `Overwrite`

## Three Invariants

These are non-negotiable system laws, not guidelines.

### Invariant I — Immutability

**StableID MUST NOT change once observed.**

A StableID, once assigned and persisted, is frozen for the lifetime of the
system. No refactor, rename, or reorganization may alter it.

This forbids:

- hash recomposition on field changes
- OriginSeed reassignment
- SemanticSignature redefinition without version bump
- silent golden snapshot updates

Enforcement: CI golden replay test fails if any StableID changes between runs.

### Invariant II — Determinism

**Identical semantic inputs MUST always produce identical StableIDs.**

Given:

- same `SemanticSignature`
- same `OriginSeed`

`Assemble(seed, sig)` must return the same `StableID` across:

- process restarts
- OS differences
- build environments
- execution ordering
- cache warm states

This is enforced by the Registry Determinism Check (Guardrail 6) which runs
identity assignment twice and asserts hash equality.

### Invariant III — Lineage Preservation

**MOVE, MERGE, and SPLIT MUST preserve ancestry graph continuity.**

When a node transitions between IR snapshots, the transformation MUST maintain
traceable StableID lineage:

- MOVE: same StableID, different Path
- SPLIT: one StableID maps to multiple descendant StableIDs
- MERGE: multiple StableIDs map to one descendant StableID

DELETE does not preserve lineage (terminal). CREATE introduces new lineage
root.

This invariant is what makes policy trace (ADR-00Z) feasible.

## Transformation Classification (Declared)

These are defined behaviorally only. Lifecycle governance is deferred to
ADR-003.

| Class | Identity | Structure | Description |
|-------|----------|-----------|-------------|
| CREATE | new StableID | — | First observation of a new entity |
| MOVE | same StableID | overlap ≥ 0.9 | Same entity, new location |
| SPLIT | 1 → N StableIDs | distributed overlap | Single entity becomes multiple |
| MERGE | N → 1 StableID | convergent overlap | Multiple entities unify |
| DELETE | StableID absent | — | Entity removed from graph |

Structural overlap threshold computed via Jaccard index on `DirectDeps`.

## Guardrails

All guardrails are CI-enforced. No reviewer dependency.

### Guardrail 1 — Import Firewall

CI step: `make check-identity-boundary`

Parses Go AST of `runtime/identity/`. Fails if any forbidden import detected.

### Guardrail 2 — Write-Once Enforcement

CI static check on `runtime/identity/`. Disallows functions matching
`Update`, `Delete`, `Replace`, `Overwrite`. Only `Insert`, `Resolve`, `Replay`
permitted.

### Guardrail 3 — Golden Identity Replay

CI job: `make identity-replay`

Loads historical IR snapshots, recomputes StableIDs, compares against
`runtime/identity/testdata/golden_identity_replay.json`. Failure means history
changed — requires ADR reference.

### Guardrail 4 — ADR Reference Requirement

If any file under `runtime/identity/` changes, PR must contain `ADR-002:` in
description or commit message. Enforced by ADR-001 CI guardrails.

### Guardrail 5 — Dual-Run Validator Hook

During identity migration phases, CI executes:

```
identity replay --compare vN vN+1
```

Allowed outcomes: identical StableIDs ✅, or explicit version bump ✅.
Silent divergence ❌ BLOCK.

### Guardrail 6 — Registry Determinism Check

CI runs identity assignment twice:

```
run A → seeds
run B → seeds
```

Hashes must match. Prevents map iteration dependence, filesystem ordering
dependence, and concurrency races.

## Migration Phase Alignment

| Phase | ID Source | StableID | Diff Key | ADR Status |
|-------|-----------|----------|----------|------------|
| 2 (current) | ImportPath | absent | ID | Frozen |
| 3 (dual) | ImportPath + StableID | H(OriginSeed \|\| Sig) | ID | This ADR |
| 4 (switch) | StableID | H(OriginSeed \|\| Sig) | StableID + structure | Future ADR |
| 5 (deprecation) | StableID only | H(OriginSeed \|\| Sig) | StableID | Future ADR |

Phase 3 introduces StableID alongside legacy ID. Diff engine continues using
ID. Phase 4 switches diff to StableID. Phase 5 removes legacy ID.

## Relationship to Other ADRs

- **ADR-001**: Governs protocol boundary and CI enforcement. ADR-002
  guardrails plug into ADR-001's Protected Surface and Controlled Change
  Protocol.
- **ADR-003** (future): Identity lifecycle — Active / Historical / Tombstoned
  / Removed states.
- **ADR-004** (future): Identity sovereignty — roles, mint detection,
  authority enforcement.
- **ADR-00Z** (future): Policy trace overlay — StableID provides the
  continuity anchor for trace attachment.

## Consequences

### Positive

- MOVE detection becomes stable under rename and refactor
- Schema evolution stops breaking history
- Policy trace has a permanent anchor
- Dual-version migration path is explicit
- Repository gains temporal invariants alongside spatial ones

### Negative

- Introduces stateful subsystem (identity registry)
- Migration from Phase 2 to Phase 3 requires rebaseline
- Identity creation becomes architecturally significant
- New CI surface (6 guardrails) must be maintained

## Non-Negotiable Rules

- StableID MUST NOT depend on ImportPath, traversal order, serialization, or
  build environment.
- StableID MUST NOT change once observed.
- Identity registry MUST be append-only (no Update, Delete, Replace, Overwrite).
- `runtime/identity/` MUST be an inward leaf package.
- CI guardrails MUST fail automatically — never reviewer-dependent.
