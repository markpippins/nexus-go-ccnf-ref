# Engineer's Guide — nexus-ccnf-ref

## Three-Phase Truth Model

```
Phase 1: Generation (CER pipeline, stateless)
Phase 2: Reconstruction (replay + rehydration)
Phase 3: Compression (snapshot engine, async)
```

The control plane produces an ExecutionState, then pipeline stages begin.
normalize-intent is the exclusive owner of ExecutionState derivation.

## Repository Architecture

### Core Pipeline (ccnf/)

8-step CCNF pipeline — the ground-truth oracle. Every step is a pure function.
The serializer (`ccnf/serializer.go`) is the highest-risk component; it is the
only code path that produces canonical bytes. No `encoding/json` escape hatch.

### Replay (replay/)

Pure fold over event log. `Fold(events) == Snapshot(state(events))` is the
R4=R5 cross-check invariant enforced across all golden vectors.

### Runtime (runtime/)

Typed ingress/egress boundaries: `ExecutionRequest` / `ExecutionReceipt`.
The replay binding (`ComputeReplayBinding`) is a domain-separated hash.
`BuildReceipt` auto-populates it; `ValidateReceipt` checks format only.

### Replay Sealed ABI (runtime/replay/)

One-way semantic membrane. `ReplayEvent` contains `EventID`, `PrevEventID`,
`Delta`, `DeltaHash` — no `ArtifactID`. Compile-time seal via
`internal/replayseal.SemanticType` marker interface.
Once `CEREventsToReplay()` converts CER events, meaning is intentionally
destroyed. No back-edges from replay to CCNF.

### Rehydration (runtime/rehydrate/)

R10.3A — produces VIEWS, never MODELS. Value receivers only. Zero pointer
receivers. No domain words in rehydrate. ViewRegistry is write-once at boot
— no `Register()`, no dynamic registration, no `init()`.

### Projection (projection/)

R10.3B — always rebuildable from ReplaySnapshot. Cache must not be observable.
Projection depends on rehydrate (one direction only).

### PDTD / PGV (tools/pgv/)

Dependency topology enforcement. Extractor plugin model:
- GoExtractor: `go list -json` for Go packages
- RustExtractor: file-scanner for Rust modules (no cargo metadata)
- CommentExtractor: `// DependsOn:` annotations (Phase A advisory only)

PGV IR is a pure dependency fact graph: nodes with import paths and direct deps.
IR hashing is deterministic. The validator checks P6–P10 rules.

## Development Workflow

### Make targets

```
make test           # unit tests
make r6             # full 32-phase proof gate
make pdtp           # PGV dependency topology
make pdtp-phase-b-verify   # PGV ↔ LegacyOracle parity
make pdtp-window-status    # Phase B window counter
make ci             # everything CI runs
```

### Adding new Go code

1. Create package in the appropriate directory
2. Add `// DependsOn:` comment for Phase A advisory
3. Run `make r6` — all 32 phases must pass
4. Update the golden vector set if CCNF behavior changes
5. Push — CI verifies across 3 OS × 2 Go versions

### Go version compatibility

The module targets `go 1.22`. When tidying locally with a newer Go:
```
go mod tidy -go=1.22
```
Without `-go=1.22`, Go will bump the directive to your local version,
breaking CI (which runs Go 1.22 and 1.23).

### Rust code

Rust source lives outside this repository (in the broader nexus monorepo
at `rust/wrp/ccnf-verifier/`). The R8 CI job is defined here but cannot
run in this repo — it executes in the monorepo CI.

## Invariants

Each invariant is enforced at a specific phase in `make r6`:

| Code | Rule | Enforced At |
|------|------|-------------|
| I8   | CCNF cross-host determinism | R1 |
| R2   | Equivalence closure | R2 |
| R3   | CER round-trip | R3 |
| R4   | Replay fold determinism | R4 |
| R5   | Tri-version snapshot lock | R5 |
| R4=R5 | Fold == Snapshot cross-check | R4=R5 |
| R6   | Cross-platform build | R6 |
| P6   | Max dependency depth ≤ 0 (projection isolation) | PDTD |
| P7   | No projection-to-projection edges | PDTD |
| P8   | No cycles in dependency graph | PDTD |
| P9   | No observed backflow (replay → CCNF, rehydrate → replay) | PDTD |
| P10a | Forbidden import paths (architectural law) | PDTD |
| P10b | Allowlist enforcement (projection-only constraints) | PDTD |

## Common Failure Modes

| Symptom | Cause | Fix |
|---------|-------|-----|
| `go: updates to go.mod needed` | Go version mismatch | `go mod tidy -go=1.22` |
| Golden vector hash mismatch | Serializer divergence | Check `ccnf/serializer.go` — no `encoding/json` |
| R2 collision | Spec ambiguity | Investigate, document as spec clarification |
| PGV hash changed | Topology evolved | Update `tools/pgv/baseline.go` if intentional |
| Phase B parity broken | PGV ≠ LegacyOracle union | Check PGV validator rules vs grep guards |

## Window Management (PDTD Phase B)

The 7-pass observational window certifies toolchain invariance:
PGV validity is already proven; the window confirms cross-platform
determinism and CI stability under real development pressure.

- Counter resets if frozen components change (extractors, IR, validators, parity logic)
- At 7/7: trigger `PDTD_PHASE_B_ACTIVATE` → remove legacy grep guards
- Check counter: `make pdtp-window-status`
