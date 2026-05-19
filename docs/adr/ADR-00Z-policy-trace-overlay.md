# ADR-00Z: Policy Trace Overlay — Causal Entropy Ledger

**Status:** Draft — Accountability Layer

**Entropy Class:** Shaper (zero net ΔH — records, does not inject)
**ΔH(I):** Zero
**ΔH(T):** Zero
**ΔH(C):** Zero
**ΔH(R):** Zero

## 0. Purpose

ADR-00Z defines a fully causal, append-only trace of all entropy injections,
collapses, and redistributions across the system.

It is not logging. It is a **causal reconstruction substrate** for why the
system is in its current entropy state.

## Context

The governance stack now has:

- ADR-003: entropy collapse — identity sealed
- ADR-004: entropy control — sovereign permissioned injection

Neither guarantees **explainability** of why entropy exists where it does.
ADR-00Z adds post-hoc and forward-traceable causality of entropy movement.
It completes the triad: collapse → control → accountability.

## Decision

All entropy transformations in the system are recorded as verifiable,
append-only EntropyEvents linked in a Causal Entropy Graph (CEG). No
entropy may exist in the system without an originating event chain.

## 1. Core Primitive: EntropyEvent

Everything in ADR-00Z reduces to one primitive:

```
EntropyEvent {
    event_id:          StableID        // unique event identifier
    timestamp:         int64           // monotonic commit time
    actor:             string          // sovereign entity (System | ADR Agent | Kernel)
    domain:            Domain          // I | T | C | R
    delta_h:           EntropyDelta    // signed ΔH per sub-domain
    adr_reference:     string          // optional ADR-NNN
    sovereign_authority: string        // authority source (ADR-004 budget)
    pre_state_hash:    string          // SHA256 of entropy state before
    post_state_hash:   string          // SHA256 of entropy state after
    parent_event_id:   StableID        // causal predecessor (optional)
}
```

Each event answers:

- what changed
- who authorized it
- which ADR allowed it
- what entropy moved
- what state it produced

This converts system evolution into a verifiable causal chain of entropy
transformations.

## 2. Append-Only Constraint

```
Once an EntropyEvent is committed, it cannot be modified or removed without
generating a compensating Collapser event.
```

This enforces:

- historical integrity
- audit invariance
- adversarial resistance

A compensating Collapser event has:

- domain = same domain as the original
- delta_h = opposite sign
- adr_reference = ADR-003 (or other Collapser ADR)
- parent_event_id = original event ID

## 3. Causal Entropy Graph (CEG)

ADR-00Z introduces a second graph layered on top of the Entropy Flow Graph.

| Graph | Purpose |
|-------|---------|
| EFG | Physical flow — how entropy moves |
| CEG | Causal justification — why entropy moved |

### CEG structure

Nodes are `EntropyEvent`s. Edges are causal dependencies:

```
event_A → event_B  means "event_A enabled event_B"
```

The CEG is built from `parent_event_id` references in each event.

### Properties

- Directed acyclic (causal time flows forward)
- Append-only (events never removed)
- Replayable (full chain reconstructable from genesis)

## 4. Sovereignty Trace Requirement

Every ADR-004 sovereign action must now emit a `SovereignTrace` alongside
its `EntropyEvent`:

```
SovereignTrace {
    actor:              string    // sovereign entity
    authority_source:   string    // ADR-NNN clause
    budget_source:      string    // which budget pool consumed
    justification_hash: string    // SHA256 of the ADR text or rationale
}
```

This binds permission, budget, action, and outcome into a single
verifiable chain.

## 5. Entropy Accountability Law

```
No entropy may exist in the system without an originating EntropyEvent chain.
```

This is the enforcement rule that closes the system. It prevents:

- orphan entropy
- undocumented drift
- silent complexity accumulation

Enforced by CI guardrail: `make check-entropy-provenance` verifies that
all current entropy state is reachable from genesis in the CEG.

## 6. System Property: Reconstructability

With ADR-00Z, the system gains:

```
The ability to reconstruct system state from entropy history alone.
```

This implies:

- no hidden state
- no implicit governance decisions
- no untraceable drift

This property is what makes Phase C systems auditable under adversarial
pressure.

## 7. Guardrails

### Guardrail 1 — EntropyEvent Append-Only

A CI check (`make check-event-immutable`) verifies that no committed
EntropyEvent has been modified or deleted without a compensating Collapser
event. The check scans the event log and validates hash chain integrity.

### Guardrail 2 — Entropy Provenance

A CI check (`make check-entropy-provenance`) verifies that all current
entropy state in every domain (I, T, C, R) is reachable from a genesis
event via a valid CEG path. Orphan entropy is a hard failure.

### Guardrail 3 — Sovereignty Trace Binding

A CI check (`make check-sovereign-trace`) verifies that every
EntropyEvent with a sovereign actor has a corresponding SovereignTrace
referencing a valid ADR-004 budget source. Untraced sovereign actions
are a hard failure.

## 8. Phase Alignment

| Phase | ADR-00Z Status |
|-------|----------------|
| 2 (current) | Not active — no event recording |
| 3 (dual) | Active — CEG initialized, events recorded |
| 4 (normalized) | Active — full provenance enforcement |
| 5 (deprecation) | Required — reconstructability essential |

## 9. Relationship to Other ADRs

- **ADR-001**: Governance protocol. ADR-00Z guardrails plug into the
  existing Protected Surface.
- **ADR-002**: Identity physics. `EntropyEvent.event_id` uses StableID.
- **ADR-003**: Entropy collapse. Produces Collapser-class EntropyEvents.
  Compensating events reference ADR-003.
- **ADR-004**: Sovereignty. Produces Generator/Shaper-class EntropyEvents.
  `SovereignTrace` binds to ADR-004 budget pools.
- **EFG/CEG**: EFG is physical flow. CEG is causal justification. Both
  are required for complete governance.

## 10. Consequences

### Positive

- Full causal reconstruction of system state from entropy history
- Orphan entropy is structurally impossible
- Sovereignty actions are traceable and auditable
- Adversarial evolution produces detectable provenance gaps

### Negative

- Event recording introduces storage and computational overhead
- CEG maintenance adds complexity to the governance layer
- Append-only constraint complicates schema evolution
- New CI surface (3 guardrails)

## 11. Non-Negotiable Rules

- EntropyEvents MUST be append-only; modification or deletion requires a
  compensating Collapser event
- All entropy MUST have an originating EntropyEvent chain (no orphan
  entropy)
- Every sovereign action MUST carry a SovereignTrace with valid
  budget source
- The CEG MUST be reconstructable from genesis to current state

---

*With ADR-00Z, the system is no longer just stable, governed, and
deterministic — it becomes causally reconstructable under bounded entropy
evolution. Collapse → Control → Accountability. The triad is closed.*
