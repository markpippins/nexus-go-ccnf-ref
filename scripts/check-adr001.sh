#!/usr/bin/env bash
set -euo pipefail

# ADR-001 IR Identity Governance Guardrail
#
# Usage: bash scripts/check-adr001.sh <BASE_SHA> [HEAD_SHA]
#
# Enforces:
#   Gate 1 - Golden rebaseline: golden_ir.json changes require REBASELINE in branch
#   Gate 2 - Structural change: protected path changes require ADR-001 in branch
#   Gate 3 - Phase lock:        PHASE != 3 + protected change requires ENTER_PHASE_3

BASE_SHA="${1:?missing base sha}"
HEAD_SHA="${2:-HEAD}"

# Resolve true merge base for branch-relative comparison
MERGE_BASE=$(git merge-base "$BASE_SHA" "$HEAD_SHA")

CHANGED=$(git diff --name-only "$MERGE_BASE"...$HEAD_SHA)
COMMITS=$(git log --format=%B "$MERGE_BASE"..$HEAD_SHA)

PROTECTED_PATHS=".tools/adr001_protected_paths.txt"

# Load protected paths file (skip blank lines and comments)
PROTECTED=()
while IFS= read -r line; do
  [[ -z "$line" || "$line" == \#* ]] && continue
  PROTECTED+=("$line")
done < "$PROTECTED_PATHS"

# Check if any changed file matches a protected path
MATCHES_PROTECTED=
for path in "${PROTECTED[@]}"; do
  if echo "$CHANGED" | grep -Fxq "$path"; then
    MATCHES_PROTECTED=1
    break
  fi
done

# If nothing protected changed, exit cleanly
if [[ -z "$MATCHES_PROTECTED" ]]; then
  exit 0
fi

# Gate 1: Golden rebaseline
if echo "$CHANGED" | grep -Fxq "tools/pgv/testdata/golden_ir.json"; then
  if ! echo "$COMMITS" | grep -q "REBASELINE"; then
    echo "❌ ADR-001 violation: golden_ir.json changed without REBASELINE declaration."
    echo "   See: docs/adr/ADR-001-ir-identity-governance.md"
    exit 1
  fi
fi

# Gate 2: Structural change requires ADR-001 declaration
if ! echo "$COMMITS" | grep -q "ADR-001"; then
  echo "❌ ADR-001 violation: IR identity surface modified without governance declaration."
  echo "   Reference ADR-001 in at least one commit message in this branch."
  echo "   See: docs/adr/ADR-001-ir-identity-governance.md"
  exit 1
fi

# Gate 3: Phase lock
if [[ -f pgv.phase ]]; then
  PHASE=$(grep -oP 'PHASE=\K\d+' pgv.phase || echo "0")
  if [[ "$PHASE" -lt 3 ]] && ! echo "$COMMITS" | grep -q "ENTER_PHASE_3"; then
    echo "❌ ADR-001 violation: Phase ${PHASE} does not allow structural changes."
    echo "   Include ENTER_PHASE_3 in commit message to declare epoch transition."
    echo "   See: docs/adr/ADR-001-ir-identity-governance.md"
    exit 1
  fi
else
  echo "❌ ADR-001 violation: pgv.phase file missing — protocol state unknown."
  exit 1
fi
