#!/usr/bin/env bash
set -euo pipefail

# CEGL-A State Compiler
# Pure deterministic function C: O → S
#
# Determines canonical governance state from observations.
# No side effects, no network calls, no environment dependence.
#
# Usage: bash scripts/compile-cegla-state.sh [BASE_SHA] [HEAD_SHA]
#   BASE_SHA defaults to HEAD~1
#   HEAD_SHA defaults to HEAD
#
# Output: JSON to stdout:
#   {
#     "declared_state": "<string>",
#     "canonical_state": "<string>",
#     "transition_detected": <bool>,
#     "invariant_health": { "<invariant_id>": <bool>, ... },
#     "entropy_cost": <int|"unknown">,
#     "axiom_a3_satisfied": <bool>
#   }
#
# Exit codes:
#   0: State determined (possibly INVALID)
#   1: A3 violation (|C(O)| != 1 — multiple possible states)
#   2: Fatal error (missing files, malformed state)

BASE_SHA="${1:-HEAD~1}"
HEAD_SHA="${2:-HEAD}"

STATE_FILE="pgv.phase"
LEDGER_FILE=".tools/transition_ledger.json"
STATE_MACHINE_FILE=".tools/pgv.state_machine.json"
PROTECTED_PATHS_FILE=".tools/adr001_protected_paths.txt"

# --- Input validation ---
if [[ ! -f "$STATE_FILE" ]]; then
  echo '{"error":"pgv.phase not found","declared_state":"UNKNOWN","canonical_state":"INVALID","transition_detected":false,"invariant_health":{"I1":false},"entropy_cost":"unknown","axiom_a3_satisfied":false}'
  exit 2
fi

if [[ ! -f "$LEDGER_FILE" ]]; then
  echo '{"error":"transition_ledger.json not found","declared_state":"UNKNOWN","canonical_state":"INVALID","transition_detected":false,"invariant_health":{"I1":true,"I_LEDGER":false},"entropy_cost":"unknown","axiom_a3_satisfied":false}'
  exit 2
fi

# --- Observation layer ---
# O1: Declared phase
DECLARED_PHASE=$(grep -oP 'PHASE=\K\d+' "$STATE_FILE" || echo "0")

# O2: Did pgv.phase change in this commit range?
PHASE_CHANGED=$(git diff --name-only "$BASE_SHA".."$HEAD_SHA" 2>/dev/null | grep -Fxq "$STATE_FILE" && echo "1" || echo "0")

# O3: Changed protected paths
CHANGED=$(git diff --name-only "$BASE_SHA".."$HEAD_SHA" 2>/dev/null || true)
MATCHES_PROTECTED=0
HAS_NON_TOOLS_PROTECTED=0
if [[ -f "$PROTECTED_PATHS_FILE" ]]; then
  while IFS= read -r line; do
    [[ -z "$line" || "$line" == \#* ]] && continue
    if echo "$CHANGED" | grep -q "^${line}"; then
      MATCHES_PROTECTED=1
      if [[ "$line" != .tools/* ]]; then
        HAS_NON_TOOLS_PROTECTED=1
      fi
      break
    fi
  done < "$PROTECTED_PATHS_FILE"
fi

# O4: Commit message keywords
COMMITS=$(git log --format=%B "$BASE_SHA".."$HEAD_SHA" 2>/dev/null || true)
HAS_REBASELINE=0
HAS_ENTER_PHASE_3=0
HAS_GOVERNANCE_REF=0
if echo "$COMMITS" | grep -q "REBASELINE"; then HAS_REBASELINE=1; fi
if echo "$COMMITS" | grep -q "ENTER_PHASE_3"; then HAS_ENTER_PHASE_3=1; fi
if echo "$COMMITS" | grep -qE "CEGL|ADR-00[1-9]|ADR-0[0-9][0-9]"; then HAS_GOVERNANCE_REF=1; fi

# O5: Identity test health (quick compilation check only)
IDENTITY_SOURCE_EXISTS=1
if [[ ! -d "runtime/identity" ]]; then IDENTITY_SOURCE_EXISTS=0; fi

# --- Rules Engine ---
# Encode each rule as a potential canonical_state.
# If >1 rule matches → A3 violation.
# If 0 rules match → INVALID.

CANDIDATES=()

if [[ "$DECLARED_PHASE" == "2" ]]; then
  if [[ "$HAS_NON_TOOLS_PROTECTED" -eq 0 && "$PHASE_CHANGED" -eq 0 ]]; then
    CANDIDATES+=("PHASE_2_FROZEN")
  fi
  if [[ "$HAS_NON_TOOLS_PROTECTED" -eq 1 && "$HAS_REBASELINE" -eq 1 ]]; then
    CANDIDATES+=("REBASELINE_PENDING")
  fi
  if [[ "$HAS_NON_TOOLS_PROTECTED" -eq 1 && "$HAS_REBASELINE" -eq 0 ]]; then
    CANDIDATES+=("INVALID")
  fi
  if [[ "$HAS_ENTER_PHASE_3" -eq 1 ]]; then
    CANDIDATES+=("PHASE_3_DUAL")
  fi
fi

if [[ "$DECLARED_PHASE" == "3" ]]; then
  CANDIDATES+=("PHASE_3_DUAL")
fi

if [[ "$DECLARED_PHASE" == "4" ]]; then
  CANDIDATES+=("PHASE_4_SWITCH")
fi

# Catch-all: no matching rule
if [[ ${#CANDIDATES[@]} -eq 0 ]]; then
  CANDIDATES+=("INVALID")
fi

# --- Axiom A3: |C(O)| = 1 ---
UNIQUE_CANDIDATES=()
for c in "${CANDIDATES[@]}"; do
  found=0
  for u in "${UNIQUE_CANDIDATES[@]}"; do
    if [[ "$u" == "$c" ]]; then found=1; break; fi
  done
  if [[ $found -eq 0 ]]; then UNIQUE_CANDIDATES+=("$c"); fi
done

A3_SATISFIED=false
CANONICAL_STATE="INVALID"
TRANSITION_DETECTED=false
ENTROPY_COST="unknown"

if [[ ${#UNIQUE_CANDIDATES[@]} -eq 1 ]]; then
  A3_SATISFIED=true
  CANONICAL_STATE="${UNIQUE_CANDIDATES[0]}"
elif [[ ${#UNIQUE_CANDIDATES[@]} -gt 1 ]]; then
  # Multiple possible states — A3 violation
  # CANONICAL_STATE stays INVALID
  A3_SATISFIED=false
fi

# --- Declared state from phase number (derived from state machine) ---
declare -A PHASE_MAP
eval "$(python3 -c "
import json
with open('$STATE_MACHINE_FILE') as f:
    d = json.load(f)
for name, state in d.get('states', {}).items():
    pn = state.get('phase_number')
    if pn is not None:
        print(f'PHASE_MAP[{pn}]=\"{name}\"')
" 2>/dev/null)"
DECLARED_STATE="${PHASE_MAP[$DECLARED_PHASE]:-UNKNOWN}"

# --- Transition detection ---
if [[ "$CANONICAL_STATE" != "INVALID" && "$DECLARED_STATE" != "UNKNOWN" ]]; then
  if [[ "$DECLARED_STATE" != "$CANONICAL_STATE" ]]; then
    TRANSITION_DETECTED=true
    # Extract transition cost from state machine
    ENTROPY_COST=$(python3 -c "
import json
with open('$STATE_MACHINE_FILE') as f:
    d = json.load(f)
for t in d.get('transitions', []):
    if (t['from'] == '$DECLARED_STATE' and t['to'] == '$CANONICAL_STATE') or \
       (t['from'] == '*' and t['to'] == '$CANONICAL_STATE'):
        print(t['entropy_cost'])
        break
else:
    print('unknown')
" 2>/dev/null || echo "unknown")
  fi
fi

# Scale cost to numeric
declare -A COST_SCALE
COST_SCALE=$(python3 -c "
import json
with open('$STATE_MACHINE_FILE') as f:
    d = json.load(f)
s = d.get('entropy_scale', {})
print(f'low={s.get(\"low\", 1)} medium={s.get(\"medium\", 5)} high={s.get(\"high\", 10)}')
" 2>/dev/null || echo "low=1 medium=5 high=10")
eval "$COST_SCALE"
ENTROPY_NUMERIC="${COST_SCALE[$ENTROPY_COST]:-0}"

# --- Invariant health ---
I1=$([[ -f "$STATE_FILE" ]] && echo "true" || echo "false")
I2=$([[ "$A3_SATISFIED" == "true" ]] && echo "true" || echo "false")
I4=$([[ "$PHASE_CHANGED" -eq 1 && "$HAS_GOVERNANCE_REF" -eq 0 ]] && echo "false" || echo "true")
I5=$([[ "$TRANSITION_DETECTED" == "true" && "$HAS_GOVERNANCE_REF" -eq 0 ]] && echo "false" || echo "true")

# --- JSON output ---
cat <<OUT
{
  "declared_state": "$DECLARED_STATE",
  "canonical_state": "$CANONICAL_STATE",
  "transition_detected": $TRANSITION_DETECTED,
  "invariant_health": {
    "I1": $I1,
    "I2": $I2,
    "I4": $I4,
    "I5": $I5
  },
  "entropy_cost": "$ENTROPY_COST",
  "entropy_numeric": $ENTROPY_NUMERIC,
  "axiom_a3_satisfied": $A3_SATISFIED,
  "observations": {
    "declared_phase": $DECLARED_PHASE,
    "phase_changed": $PHASE_CHANGED,
    "protected_paths_modified": $MATCHES_PROTECTED,
    "non_tools_protected_modified": $HAS_NON_TOOLS_PROTECTED,
    "has_rebaseline": $HAS_REBASELINE,
    "has_enter_phase_3": $HAS_ENTER_PHASE_3,
    "has_governance_ref": $HAS_GOVERNANCE_REF
  }
}
OUT

# Exit code
if [[ "$A3_SATISFIED" == "false" && ${#UNIQUE_CANDIDATES[@]} -gt 1 ]]; then
  exit 1
fi
exit 0
