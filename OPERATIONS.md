# Operations Guide — nexus-ccnf-ref

## CI/CD

### Workflow: CCNF Conformance (R1-R10.3)

Triggered on push/PR to `main` or `master`.

### Job Matrix

| Job | OS | Go | Description |
|-----|----|-----|-------------|
| R1 | 3×2 | 1.22/1.23 | CCNF determinism + golden vectors |
| R2 | 3×2 | 1.22/1.23 | Equivalence closure + collision atlas |
| R3 | 3×2 | 1.22/1.23 | CER round-trip |
| R4 | 3×2 | 1.22/1.23 | Replay oracle |
| R5 | 3×2 | 1.22/1.23 | Snapshot oracle |
| R4=R5 | 3×2 | 1.22/1.23 | Fold == Snapshot cross-check |
| R6 | 3×2 | 1.22/1.23 | Cross-platform build + golden vectors |
| R10.3 | 3×2 | 1.22/1.23 | Rehydration: purity, isolation, registry |
| R10.3B | 3×2 | 1.22/1.23 | Projection: no back-edge, import isolation |
| Phase A | 3×2 | 1.22/1.23 | PGV Go + Rust dependency topology |
| Phase B | 3×2 | 1.22/1.23 | PGV ↔ LegacyOracle parity (observational) |

### Required Checks

All jobs except Phase B are required. Phase B has `continue-on-error: true`
during the observational window. On `PDTD_PHASE_B_ACTIVATE`, Phase B becomes
required and replaces the legacy grep-based enforcement.

### Run Status

```
make pdtp-window-status
```

Shows: frozen surface integrity, IR hash match, local parity, CI counter.

## PDTD Phase B Window

### Counter

The window counter tracks consecutive full-matrix (6/6) Phase B passes
from the most recent run backward. It resets to 0 if any frozen component
changes (extractor semantics, IR schema, validation rules, parity comparison
logic).

### Progression

| State | Action |
|-------|--------|
| Counter < 7 | Push new commits to main/master |
| Counter == 7 | Trigger `PDTD_PHASE_B_ACTIVATE` |
| Frozen component changed | Counter resets automatically |

### Activation

At 7/7 consecutive passes:

1. Create `PDTD_PHASE_B_ACTIVATE` commit that:
   - Removes legacy grep-based enforcement commands from CI
   - Collapses Phase B from observational to required
   - Collapses `make r6` from 32 → 29 phases
2. Push and verify CI is green
3. Update this document

## Branch Management

Two branches are kept in sync:
- `master` — default branch
- `main` — CI trigger branch (workflow runs on both)

Always push to `master` first, then sync:
```
git push origin master
git push origin master:main
```

## Versioning

| Tag | Trigger | Freeze |
|-----|---------|--------|
| v0.1.0-ccnf | R1 gate | Serialization semantics |
| v0.1.0-fuzz | R2 gate | Equivalence class boundaries |

Tags are created by maintainers when the corresponding gate passes.
Tags are immutable once pushed.

## Checking CI Status

```
# Latest run
gh run list --repo markpippins/nexus-go-ccnf-ref --workflow conformance.yml --limit 1

# View failed jobs
gh run view --repo markpippins/nexus-go-ccnf-ref <run-id> --log-failed

# Get job logs
gh api /repos/markpippins/nexus-go-ccnf-ref/actions/jobs/<job-id>/logs
```

## What to Check When CI Fails

1. **go.mod errors**: Run `go mod tidy -go=1.22` locally
2. **Golden vector mismatch**: Something changed canonical output — check serializer
3. **Phase B parity break**: PGV validator diverged from LegacyOracle — compare outputs
4. **Rust verifier failure**: Not runnable in this repo (rust code in monorepo)
5. **Cross-platform failure**: Usually nondeterminism in test (map iteration, timestamps)
