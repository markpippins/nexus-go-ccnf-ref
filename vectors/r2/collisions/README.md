# R2 Collision Atlas

This directory contains the equivalence class structure of the CCNF specification,
discovered by structured fuzzing.

## Files

| File | Status | Description |
|---|---|---|
| `v0.1.0-fuzz.json` | committed | Primary collision report — deterministic, seeded, reproducible |
| `classified.json` | committed | Classification layer: expected vs unexpected collisions |

## Rules

- Collision atlas is a **version-scoped spec artifact**, not a test log.
- Regenerate with `make r2-collisions` before committing CCNF changes.
- Unexpected collisions are **spec ambiguities**, not bugs — they require spec clarification or version epoch bump.
- Do not interpret collision-free runs as "system health" — they document boundary conditions.

## Classification

- **expected**: collisions defined by the serialization contract (key reordering, null/omission equivalence, whitespace normalization)
- **spec_ambiguity**: collisions not defined by the contract — requires spec clarification
- **divergence**: non-determinism or implementation bug (should never appear here)
