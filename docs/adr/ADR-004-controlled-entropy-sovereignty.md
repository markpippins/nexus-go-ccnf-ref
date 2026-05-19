# ADR-004: Controlled Entropy Sovereignty (CES)

**Status:** Draft — Sovereignty Layer

**Entropy Class:** Shaper (strong regulatory component)
**ΔH(I):** Zero (identity remains sealed by ADR-003)
**ΔH(T):** Negative (controlled expansion only)
**ΔH(C):** Negative (budget-constrained)
**ΔH(R):** Negative (bounded exploration)

## 0. Purpose

ADR-004 defines who is allowed to introduce entropy into the system, under
what constraints, and with what traceability guarantees.

ADR-003 collapsed identity entropy permanently. ADR-004 governs all
**non-identity entropy** — topology expansion, semantic mutation, and
runtime exploration — under a permissioned authority model.

The key transition:

```
Before: entropy is accidental
After:  entropy is authorized
```

## Context

ADR-001–003 established:

- Structural governance (ADR-001)
- Identity physics — StableID, OriginSeed, invariants (ADR-002)
- Entropy collapse — three laws sealing identity space (ADR-003)

What remains ungoverned is the **permission to introduce new entropy**
into topology (T), commitment (C), and runtime (R) domains. Without this
layer, entropy can re-enter silently through new abstraction layers, agent
policies, or topology extensions — even though identity remains frozen.

## Decision

Entropy is no longer a free variable. It is a **scarce resource**
governed by sovereign authority, budget allocation, and reversibility
discipline.

## 1. Sovereign Actor Model

Only three entity classes may authorize entropy injection:

| Sovereign | Scope | Authority Source |
|-----------|-------|-----------------|
| System (default baseline) | Automatic entropy within defined budgets | PGV kernel invariants |
| ADR-authorized Agents | Explicit entropy operations per ADR declaration | Approved ADR with EntropyClass=Generator |
| Governance Kernel (PGV core) | Budget reallocation, phase transitions | ADR-001 protocol authority |

### Sovereignty Constraint

```
Only sovereign actors may create Entropy Generators or modify
Generator-class ADRs.
```

This is a hard boundary. No non-sovereign entity may introduce entropy
into topology, commitment, or runtime layers.

## 2. Entropy Injection Types

ADR-004 formalizes three controlled injection categories. Note the
exclusion: **identity entropy is forbidden** (sealed by ADR-003).

### 2.1 Structural Entropy Injection (ΔH(T) > 0)

New topology edges, abstraction layers, or coordination patterns.

Effect: increases H_T under controlled bounds.

Governance: requires ADR-authorized Agent or Governance Kernel approval.

### 2.2 Semantic Entropy Injection (ΔH(C) > 0)

New meaning domains or interpretation layers attached to frozen identities.

Effect: increases H_C (interpretation space), NOT H_I (identity remains
sealed).

Governance: requires explicit EntropyClass declaration in authorizing ADR.

### 2.3 Runtime Entropy Injection (ΔH(R) > 0)

Non-deterministic behavior zones, adaptive execution policies, or
exploration agents.

Effect: increases H_R but must be bounded by commitment gates.

Governance: runtime entropy requires a defined budget cap and reversibility
plan.

### Forbidden injection

Identity entropy injection (any change to H_I) remains forbidden by ADR-003.
ADR-004 does not reopen this.

## 3. Sovereign Control Law

```
All entropy injections must be explicitly classified, bounded, and
reversible at the commitment layer unless marked as non-reversible with
an approved ADR Collapser dependency.
```

### Classification requirement

Every ADR that introduces entropy must declare, in its metadata:

- EntropyClass: Generator | Shaper | Collapser
- DomainImpact: T | C | R | I (forbidden)
- ΔH_estimate: High | Medium | Low
- Reversible: Yes | No (if No, requires paired Collapser ADR)

### Bounding requirement

Each injection must specify its maximum entropy contribution. This may be
expressed as a maximum branching factor, maximum latency, or maximum
interpretation count for the injected construct.

### Reversibility requirement

Unless explicitly marked non-reversible (with paired Collapser ADR), every
entropy injection must be removable without affecting the identity lattice.

## 4. Entropy Budget Allocation Authority (EBAA)

### 4.1 Budget model

Each governance phase has:

- A fixed entropy capacity per domain
- Allocation rules per domain

| Domain | Phase 2 (current) | Phase 3 (dual) | Phase 4 (normalized) |
|--------|-------------------|----------------|----------------------|
| Identity (I) | 0 | 0 | 0 |
| Topology (T) | unrestricted | controlled | strict budget |
| Commitment (C) | unrestricted | budgeted | strict budget |
| Runtime (R) | unrestricted | adaptive | adaptive with cap |

### 4.2 Sovereign action rule

```
No entropy injection is valid unless it consumes or reallocates from an
approved entropy budget pool.
```

This prevents uncontrolled growth. Budget pools are defined per phase by
the Governance Kernel. Reallocation requires an ADR-004 amendment or phase
transition.

## 5. Entropy Flow Graph Update (EFG v2)

EFG edges are recategorized into two classes:

### 5.1 Free-flow edges (pre-ADR-004)

Uncontrolled entropy movement. Still valid for Shaper-class ADRs that
redistribute existing entropy without net increase.

### 5.2 Sovereign edges (post-ADR-004)

Each edge now requires:

- Authorizing entity (sovereign actor)
- Budget source (which pool is consumed)
- Reversibility flag (whether the edge can be removed)

EFG becomes a **licensed flow system**, not a natural system.

## 6. Guardrails

### Guardrail 1 — ADR Metadata Requirement

A CI check (`make check-adr-classification`) verifies that every ADR
committed after ADR-004 declares its EntropyClass, DomainImpact, and
ΔH_estimate. Missing metadata blocks merge.

### Guardrail 2 — Identity Entropy Block

A CI check (`make check-identity-entropy` or extended
`check-identity-boundary`) verifies that no Generator-class ADR may target
the Identity (I) domain. Identity is permanently sealed by ADR-003.

### Guardrail 3 — Budget Compliance

A CI check (`make check-entropy-budget`) verifies that the total entropy
budget per domain per phase has not been exceeded. This uses a budget
tracking file (`.tools/entropy_budget.json`) updated by each ADR commit.

## 7. Relationship to Other ADRs

- **ADR-001**: Governance protocol. ADR-004 guardrails plug into the
  existing Protected Surface and Controlled Change Protocol.
- **ADR-002**: Identity physics. ADR-004 does not reopen identity.
- **ADR-003**: Entropy collapse. ADR-004 governs all entropy outside
  identity space. The pair forms: collapse (ADR-003) + control (ADR-004).
- **ADR-00Z** (future): Policy trace. Requires ADR-004's sovereign edge
  model to build a causal entropy ledger.

## 8. Consequences

### Positive

- Entropy becomes a scarce resource with explicit governance
- Identity remains permanently sealed (ADR-003 guarantee preserved)
- EFG becomes a licensed flow system — traceable and auditable
- Runtime exploration bounded by commitment gates

### Negative

- Entropy injection requires ADR-level ceremony
- Budget tracking adds bookkeeping overhead
- Sovereign actor model introduces access control complexity
- New CI surface (3 guardrails)

## 9. Non-Negotiable Rules

- Identity entropy injection is permanently forbidden (ADR-003 supremacy)
- Only sovereign actors may create Generator-class ADRs
- All entropy injections must be classified, bounded, and reversible
  unless paired with a Collapser ADR
- EFG edges must carry authorizing entity, budget source, and
  reversibility flag
