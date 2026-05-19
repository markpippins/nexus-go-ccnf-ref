// Package identity provides StableID lineage tracking for IR nodes.
//
// This subsystem is the temporal root of truth for identity continuity.
// It owns exactly three responsibilities:
//   1. OriginSeed assignment (UUIDv7, write-once)
//   2. StableID assembly
//   3. Historical replay validation
//
// Identity MUST NOT depend on rehydration, execution, or graph traversal.
// See: docs/adr/ADR-002-stableid-identity-normalization.md
package identity
