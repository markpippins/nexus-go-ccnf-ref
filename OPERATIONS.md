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
| R10.3B | 3×2 | 1.22/1.23 | Projection: no back-edge, cache isolation |
| Phase A | 3×2 | 1.22/1.23 | PGV Go dependency topology + advisory |
| Phase B | 3×2 | 1.22/1.23 | PGV topology enforcement (required gate) |
| ADR-001 | 1 | — | IR identity governance (protected surface + phase lock) |
| R10.4 | 1 | 1.22 | Identity registry guardrails + tests |
| R10.5 | 1 | — | CEGL-A closed-world verification |

### Required Checks

All jobs are required. Phase B enforces PGV topology as the sole
dependency authority. R10.5 enforces CEGL-A governance state legality.

### Run Status

```
make pdtp-window-status
```

Shows: frozen surface integrity, IR hash match, CI counter.

## PGV Topology Enforcement

PGV is the sole topology enforcement authority. Run:
```
make pdtp
```
Or check baseline integrity:
```
make pdtp-window-status
```

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
3. **Phase B PGV failure**: Dependency topology violated — check PGV output
4. **Cross-platform failure**: Usually nondeterminism in test (map iteration, timestamps)
