# ADR-003: Minimal Entropy Identity Normalization (MEIN)

**Status:** Draft — Entropy Collapser

**Entropy Class:** Collapser (Strong)
**ΔH(I):** Very High reduction
**ΔH(T):** High reduction (indirect)
**ΔH(C):** Medium reduction (ordering stabilization)
**ΔH(R):** Medium reduction (interpretation stabilization)

## 0. Purpose

ADR-003 defines the **irreducible identity law under system mutation**.

Its goal is to prevent identity drift, structural ambiguity, and
concurrency-induced re-interpretation of system state. It collapses three
entropy axes: semantic mutation over time, identity/structure conflict, and
concurrent ordering ambiguity.

## Context

ADR-002 established StableID as a registered LineageID with three invariants.
However, three unresolved tensions remain:

1. **SemanticSignature versioning**: can the definition of SemanticSignature
   evolve without breaking existing StableIDs?
2. **Identity vs structure precedence**: when StableID and structural
   equivalence disagree, which defines MOVE vs REWRITE?
3. **Registry ordering determinism**: what constitutes "first insertion"
   under concurrent or replayed ingestion?

These are not design choices. They are degrees of freedom that, left
unresolved, allow identity entropy to re-enter the system under evolution
pressure.

## Decision

Three laws, each collapsing one entropy axis.

---

## 1. SemanticSignature Versioning Law

**SemanticSignature is immutably versioned and structurally embedded, not
externally schema-evolved.**

### Law SSV1

```
A SemanticSignature is valid only if its interpretation is
self-contained at the point of evaluation.
```

### Constraint

- No external schema evolution may reinterpret an existing SemanticSignature.
- Versioning is embedded as a **forked identity**, not a mutable schema.
- A change to SemanticSignature derivation produces a new identity lineage,
  not a reinterpretation of existing StableIDs.

### Concrete rule

If the definition of SemanticSignature must change (e.g. Name normalization
changes, dependency model evolves), the new definition produces `StableID_v2`.
Existing `StableID_v1` values remain valid. They are not silently recomputed.

### Entropy justification

This eliminates:

- reinterpretation drift (high H_I → T coupling)
- retroactive meaning injection
- schema-layer ambiguity propagation

### EFG impact

- ↓ I → T ambiguity
- ↓ C → I reinterpretation loops
- ↓ long-range latency drift in identity resolution

---

## 2. Identity vs Structure Precedence Law

**Identity overrides structure when and only when identity is normalized via
ADR-003 rules. Otherwise, structure is authoritative.**

### Law ISPL1

```
When Identity and Topology conflict, the system resolves to the last
committed normalized Identity state.
```

### Constraint

- Structure cannot redefine identity post-normalization.
- Identity cannot be inferred from topology except through explicit
  normalization gates.
- Before normalization: structure dominates (existing Phase 2 behavior).
- After normalization: identity dominates.

### Key principle

Structure is observational. Identity is declarative.

This prevents:

- graph-induced identity hallucination
- runtime re-binding of entities
- agent-driven reinterpretation of system objects

### Entropy justification

Without this rule, the `T → I` feedback loop becomes uncontrolled and
topology becomes a semantic source (high entropy generator). With this rule,
topology becomes non-semantic infrastructure.

### EFG impact

- Breaks `T → I` entropy feedback loop
- Reduces cycle amplification potential
- Stabilizes runtime interpretation layer

---

## 3. Registry Ordering Determinism Law

**Registry ordering is defined by commit-time causal ordering, not insertion
time.**

### Law ROD1

```
"First insertion" is defined as the first successfully committed normalized
state in the global commitment lattice.
```

### Constraint

- Arrival order is not meaningful.
- Only commit order under normalized identity context is valid.

### Resolution mechanism

When concurrent ingestion occurs:

1. Identity normalization occurs first (ADR-003 gate).
2. Competing entries are reduced to normalized equivalence classes.
3. Only then is commit order assigned.

### Entropy justification

This removes:

- race-condition identity divergence
- phantom first writers
- temporal ambiguity in system history

Time becomes a projection of committed state, not a source of identity.

### EFG impact

- Removes `R → C` ordering ambiguity
- Stabilizes commit lattice structure
- Prevents runtime replay divergence

---

## 4. Unifying Principle

These three laws reduce to a single invariant:

```
Identity is not derived, inferred, or observed. It is only valid when it
has passed through a normalization gate and entered the commitment lattice.
```

Everything else is forbidden ambiguity.

---

## 5. Enforcement

### Guardrail 1 — SemanticSignature Frozen

A CI check (`make check-signature-immutable`) verifies that
`SemanticSignature` derivation in `runtime/identity/internal/canonicalize.go`
has not changed since the last ADR-003 declared version. If changed without
an ADR-003 version bump, CI fails.

### Guardrail 2 — Identity Precedence

The diff engine (`ComputeDiff`) must implement the identity-over-structure
rule: after normalization, StableID equality dominates
`STRUCTURE_SIGNATURE` equality in MOVE classification. `make
check-identity-precedence` verifies this.

### Guardrail 3 — Commit Lattice Ordering

Registry determinism tests (`TestRegistryDeterminism`) are extended to
validate that replay order matches commit order, not arrival order.
`make check-commit-ordering` verifies.

---

## 6. Phase Alignment

| Phase | Status | Identity Rule |
|-------|--------|---------------|
| 2 (current) | Frozen | Structure dominates (pre-normalization) |
| 3 (dual, ADR-002) | Active | Identity introduced, structure still authoritative |
| 4 (normalization, ADR-003) | This ADR | Identity dominates after normalization gate |
| 5 (deprecation) | Future | Legacy ID removed |

ADR-003 activates at the Phase 3 → 4 boundary. Prior to activation,
structure remains authoritative (existing Phase 2 behavior).

---

## 7. Relationship to Other ADRs

- **ADR-001**: Governance protocol. ADR-003 guardrails plug into the
  existing Protected Surface and Controlled Change Protocol.
- **ADR-002**: StableID definition and identity subsystem. ADR-003 provides
  the entropy-collapse layer that makes ADR-002 stable under evolution.
- **ADR-004** (future): Identity sovereignty. Requires ADR-003's
  normalization gate as a precondition.
- **ADR-00Z** (future): Policy trace. Lineage continuity under ADR-003
  provides deterministic anchor.

## 8. Consequences

### Positive

- Identity entropy is bounded under all system mutations
- Semantic reinterpretation is structurally impossible
- Concurrency cannot produce identity divergence
- Three-law system is minimal and checkable

### Negative

- SemanticSignature version changes require explicit forking
- Identity-over-structure rule requires diff engine modification
- Commit-lattice ordering adds serialization requirement
- New CI surface (3 guardrails)

## 9. Non-Negotiable Rules

- SemanticSignature MUST NOT be reinterpreted after assignment.
- Identity MUST dominate structure after normalization.
- Registry ordering MUST follow commit causal order, not arrival order.
- No identity MAY be derived or inferred outside a normalization gate.

---

*This ADR is the dominant entropy sink for identity coherence in PGV Phase C.
After adoption, identity stops being a fluid runtime concept, topology stops
being a semantic substrate, and concurrency stops being an ambiguity source.*
