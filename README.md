# CCNF Reference Implementation — nexus-ccnf-ref

## Purpose

Ground-truth oracle for the CER Canonical Normalization Function (CCNF).
Not production. Not optimized. Just correct.

Every other implementation of CCNF must prove bitwise equivalence against this module.

## Non-Goals

- NOT optimized for performance
- NOT a production runtime
- NOT extensible without version epoch change
- NOT a convenience library
- NOT backward-compatibility preserving across CCNF versions

## Why This Exists

Distributed systems fail when behavior depends on implementation details.
This repository exists to make correctness independent of language, runtime, or organization.

## Authority Model

| Authority | Source |
|---|---|
| Specification authority | `spec/*.md` + `SERIALIZATION_CONTRACT.md` + Golden Vectors |
| Behavioral authority | `ccnf-conformance` binary |
| Correctness | No other implementation defines it |

## Architecture

```
Raw Input
    ↓
CCNF — 8-step deterministic normalization
  ├── 1. structural_parse     raw → intermediate schema
  ├── 2. canonicalize_fields  key order, type norm, NFC, timestamps
  ├── 3. derive_identity      entity_key (pure over static fields)
  ├── 4. normalize_intent     controlled vocabulary
  ├── 5. resolve_artifacts    type:id references
  ├── 6. compute_state_delta  artifact-scoped patch
  ├── 7. serialize            fixed field order, compact JSON
  └── 8. hash + sign          SHA256
    ↓
CER Event
    ↓
Replay Engine (pure fold over rehydrated events)
    ↓
Snapshot Oracle (minimal, in-memory, synchronous)
```

## Repository Layout

```
ccnf-ref/
  SERIALIZATION_CONTRACT.md       ← immutable root commit — 9 serialization rules
  README.md                       ← this file
  Makefile                        ← R1–R6 CI targets
  go.mod                          ← module github.com/anomalyco/nexus-ccnf-ref

  spec/                           ← Formal RFC-style specifications
    CCNF_SPEC.md                  ← canonical serialization, normalization, pipeline
    CER_SPEC.md                   ← CER schema, field semantics, structural invariants
    REPLAY_SPEC.md                ← fold semantics, cursor model, delta merge rules
    SNAPSHOT_SPEC.md              ← snapshot builder, validation, tri-version lock
    VERSIONING_MODEL.md           ← version contract, migration policy

  vectors/
    v1/                           ← 32 golden vector files (input → expected hash)
    expected-hashes.json          ← master hash table for v1
    r2/collisions/                ← collision atlas (87.5k inputs, 0 collisions)

  ccnf/
    serializer.go                 ← THE ONLY canonical serializer in the system
    ccnf.go                       ← Run() entry point — 8-step pipeline orchestration
    structural_parse.go           ← step 1
    canonicalize.go               ← step 2
    identity.go                   ← step 3 — entity_key derivation
    intents.go                    ← step 4 — controlled vocabulary
    artifacts.go                  ← step 5 — type:id resolution
    deltas.go                     ← step 6 — artifact-scoped state_delta
    serialize.go                  ← step 7 — deterministic serialization
    hash.go                       ← step 8 — SHA256 + signature
    cer.go                        ← CER serialize/parse
    r2.go                         ← equivalence class fuzzer (8 mutation dimensions)
    ccnf_test.go                  ← table-driven golden vector runner
    fuzz_test.go                  ← R2 structured tests
    cer_test.go                   ← CER round-trip tests

  replay/
    types.go                      ← CEREvent, RuntimeState, EntityState
    fold.go                       ← ApplyEvent, Fold, initialState
    cursor.go                     ← cursor operations (step, jump, time-travel)
    replay.go                     ← Replay, ReplayFromCursor, ReplayRange
    state.go                      ← applyDelta, updateEntity, getEntity
    r4r5_crosscheck_test.go       ← R4=R5 invariant over all golden vectors
    snapshot/                     ← R5 snapshot oracle
      types.go                    ← Snapshot, SnapshotContext
      builder.go                  ← Build, BuildFromReplay
      snapshot.go                 ← Validate, Verify, RoundTrip
      compare.go                  ← Compare, EqualStates
      lock.go                     ← ValidateTriVersionLock, IsValidLock

  conformance/
    runner.go                     ← standalone binary oracle → ccnf-conformance
    fuzz.go                       ← semantic fuzzer (deterministic PRNG)
    multi_host.go                 ← cross-platform compilation + hash comparison

  .github/workflows/
    conformance.yml               ← PR gate: golden vectors + fuzz + 3-OS matrix
```

## Invariants Enforced

| Invariant | Enforced By | Failure Mode |
|---|---|---|
| I1 — replay(rehydrate(CER)) == snapshot_n | replay/fold_test.go | Round-trip divergence |
| I2 — no global state in events | ccnf/structural_parse.go rejects symbolic refs | DELTA_SCOPE_VIOLATION |
| I3 — identity immutable (no re-keying) | ccnf/identity.go excludes runtime + temporal data | Cross-host entity_key mismatch |
| I4 — snapshots never feed write path | snapshot.go has no write-path import | Compilation error |
| I5 — anti-collapse guard (requires causal index) | ccnf/identity.go + conformance/fuzz.go | Undetected without fuzz |
| I6 — delta chain integrity (ancestor_event_id) | cer/pipeline.go validates ancestor | Orphan DELTA reject |
| I7 — rehydration purity | cer/rehydrate.go deterministic by construction | Non-determinism caught by vectors |
| I8 — CCNF cross-host determinism | ccnf_test.go + conformance/multi_host.go | Golden vector hash mismatch |
| I9 — collapse versioning | Version triple in every entry point | Fuzz divergence |
| I10 — phase separation (Gen ≠ Recon ≠ Compress) | Module boundaries (ccnf/ vs cer/ vs replay/) | Circular import → compile error |
| I11 — identity epoch (CCNF version scoping) | ccnf/identity.go scoped to ccnf_version | CCNF_VERSION_MISMATCH |

## Serialization Contract (9 Rules)

| Rule | Value |
|---|---|
| **Key order** | Lexicographic bytewise UTF-8, all nesting levels |
| **Whitespace** | Compact — no spaces, no indent, no trailing newline |
| **Integers** | JSON number without `.0` |
| **Floats** | IEEE-754 double, fixed notation, locale-independent `.`, NO scientific notation |
| **Strings** | UTF-8 NFC normalized at ingress. BOM stripped. Zero-width chars stripped. Case preserved |
| **Null vs absent** | All fields present: nullable → `null`, arrays → `[]`, objects → `{}` |
| **Timestamps** | Epoch seconds int64. ISO-8601 converted at ingress. No timezone in canonical form |
| **Arrays** | `ordered:true` preserves input order; all others sorted lexicographically |
| **Hash** | `SHA256(canonical_UTF8_bytes)` — no trailing newline |

See [`SERIALIZATION_CONTRACT.md`](./SERIALIZATION_CONTRACT.md) for the full immutable specification.

## Things Frozen at v0.1.0-ccnf

These are one-way doors. Do not tag `v0.1.0-ccnf` until stable.

| Freeze | Rule |
|---|---|
| Float encoding | `strconv.AppendFloat(..., 'f', -1, 64)`, reject scientific notation |
| Unicode normalization | NFC only. No NFKC, no raw UTF-8 alternatives |
| Timestamp canonical form | Epoch seconds int64. No sub-second drift without version bump |
| Field presence | `null` / `[]` / `{}` — no optional omission |
| CCNF version isolation | Any change to frozen rules requires CCNF version increment |

## Golden Vectors

32 vectors across 9 categories, located in `vectors/v1/`.

| Category | Vectors | What It Tests |
|---|---|---|
| Core determinism | 001–004 | Full pipeline, delta, mutation, cross-host equivalence |
| Field canonicalization | 005–012 | Key order, bool/null normalization, NFC, whitespace, timestamps, arrays |
| Identity derivation | 013–018 | No runtime/timestamp in entity_key, scope sensitivity, no collision |
| Intent normalization | 019–020 | Controlled vocabulary; unmappable → error |
| Artifact resolution | 021–022 | Valid type:id; symbolic → error |
| Delta construction | 023–025 | before_hash null on create, multiple artifacts, scope violation |
| Serialization | 026–028 | Fixed field order, no optional omission, compact JSON |
| Hash | 029–030 | Deterministic, changes on any field |
| Identity epoch | 031–032 | CCNF version isolation; mismatch → error |

**Immutability rule**: Golden vectors are **append-only**. Once released into `vectors/v1/`, no existing vector is ever modified. Corrections require `vectors/v2/` and a CCNF version increment. Failure to respect this breaks the hash chain and invalidates every existing snapshot.

## Quickstart

```
go test ./ccnf/...                 # unit tests + golden vectors
make conformance                    # full golden vector suite
make cross-platform                 # local determinism check (cross-compile 3 OS targets)
make fuzz                           # deterministic PRNG fuzz (10k iterations)
make oracle                         # build conformance/runner.go → ./bin/ccnf-conformance
make ci                             # everything CI runs
```

## CCNF Conformance Binary Oracle

```
make oracle
./bin/ccnf-conformance run vectors/v1/          # reports pass/fail per vector
./bin/ccnf-conformance fuzz                      # runs determinism fuzzer
./bin/ccnf-conformance verify path/to/events.log # validates event stream against oracle
```

The binary is the **external oracle** — other systems (nexus-runtime, CI, third-party impls) invoke it to validate their output without importing Go packages.

## Implementation Phases

```
R1 — CCNF Oracle
  Gate: 32/32 vectors pass, cross-platform hash identity
  → tag v0.1.0-ccnf
  → FREEZES serialization semantics. Any change requires CCNF version increment.

R2 — Determinism Harness
  Gate: 10k fuzz runs zero divergence
  → tag v0.1.0-fuzz

R3 — CER Reference (FULL only)
  Gate: write → rehydrate → CCNF identical

R4 — Replay Oracle
  Gate: replay(events) == snapshot_n

R5 — Snapshot Oracle (minimal, in-memory)
  Gate: I4 enforced, snapshot never feeds write path

R6 — CI Freeze
  Gate: every PR gated on golden vectors + fuzz + cross-platform matrix
```

## Engineering Warnings

- **Serializer is the highest-risk component.** No `encoding/json` escape hatch. Only `ccnf/serializer.go` produces canonical bytes.
- **Go map iteration is hostile.** Always copy → sort → encode. Never serialize Go maps directly.
- **NFC normalization MUST be explicit.** Linux and macOS produce different pre-composed forms from the same input. Without explicit NFC normalization, hashes diverge.
- **Convert timestamps at ingress.** Never allow `time.Time` objects past the parse step. Epoch seconds (int64) only.
- **Floats use `strconv.AppendFloat(..., 'f', -1, 64)`** then reject scientific notation. Do not use `%g` or `%e` formatting.

## Cross-Platform Model

| Scope | Tool | Role |
|---|---|---|
| Local dev | `make cross-platform` | Fast feedback — catches developer mistakes before commit |
| CI | `conformance.yml` matrix | Constitutional truth — institutional enforcement, clean environments |

Both are required. Local checks without CI: no enforcement boundary. CI without local checks: discover divergence after context lost.

## Relationship to nexus-runtime

```
nexus-ccnf-ref                      nexus-runtime
  (oracle, separate Go module)        (production)
  vectors/ ────────────────────────→ reads vectors via path convention
  SERIALIZATION_CONTRACT.md ←──────── pins same contract
  ccnf-conformance binary ──────────→ invoked as external validator
```

Data flow: one direction only. Oracle → Production. Never Production → Oracle.

## Glossary

| Term | Definition |
|---|---|
| CCNF | Canonical Normalization Function — 8-step deterministic transform from raw input to canonical CER |
| CER | Canonical Event Record — single immutable event format, the system's atomic truth unit |
| entity_key | `SHA256(canonical_entity_signature)` — hard identity, excludes runtime/temporal data |
| collapse_key | Stable human-readable semantic identifier for entity collapse across versions |
| alias_keys | Historical names used for read-time resolution only, never for hashing or identity generation |
| state_delta | Artifact-scoped patch within a CER event — `{artifact_id, before_hash, after_hash, patch}` |
| FULL | Compression strategy storing complete artifact state |
| DELTA | Compression strategy storing only the patch, referencing ancestor via `ancestor_event_id` |
| ALIAS | Compression strategy: identity merge metadata with zero state change |
| SYNTHETIC | Compression strategy: generated event with explicit `derivation_source` |
| Triple-version lock | `(ccnf_version, collapse_engine_version, rehydration_version)` — all three match or snapshot is invalid |
| Golden vector | Fixed (input → expected hash) pair that all implementations must match bitwise |
| Canonical serialization | The single deterministic JSON encoding defined by `SERIALIZATION_CONTRACT.md` |
| Identity epoch | CCNF version boundary; cross-version identity comparison is invalid without explicit migration |
| Semantic collapse | Read-time application of Rules 2 + 3 (collapse_key equivalence, alias resolution) |
