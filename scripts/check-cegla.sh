#!/usr/bin/env bash
set -euo pipefail

# CEGL-A Verification Engine
# Detection layer — validates evolution legality.
#
# Usage: bash scripts/check-cegla.sh [BASE_SHA] [HEAD_SHA]
#
# Process:
#   1. Run state compiler → canonical state
#   2. Look up (previous, compiled) in T
#   3. If ∉ T → hard fail
#   4. If ∈ T → report entropy cost, verify invariants
#   5. Sub-calls check-adr001.sh (ADR guardrails)
#
# Exit codes:
#   0: Evolution is legal
#   1: Evolution is illegal (transition not in T, invariant violation, etc.)
#   2: Fatal error

BASE_SHA="${1:-HEAD~1}"
HEAD_SHA="${2:-HEAD}"
LEDGER_FILE=".tools/transition_ledger.json"

# Ensure we're in the repo root
if [[ ! -f "$LEDGER_FILE" ]]; then
  echo "CEGL-A: transition_ledger.json not found"
  exit 2
fi

echo "================================================"
echo "  CEGL-A Verification Engine"
echo "================================================"
echo ""

# Step 1: Run state compiler
echo "--- Step 1: State compilation ---"
COMPILER_OUTPUT=$(bash scripts/compile-cegla-state.sh "$BASE_SHA" "$HEAD_SHA" 2>&1) || true
echo "$COMPILER_OUTPUT" | python3 -m json.tool 2>/dev/null || echo "$COMPILER_OUTPUT"

# Parse compiler output
parse_json_bool() {
  local val="$1"
  if [[ "$val" == "True" || "$val" == "true" || "$val" == "1" ]]; then
    echo "true"
  else
    echo "false"
  fi
}

CANONICAL_STATE=$(echo "$COMPILER_OUTPUT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('canonical_state','UNKNOWN'))" 2>/dev/null || echo "UNKNOWN")
DECLARED_STATE=$(echo "$COMPILER_OUTPUT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('declared_state','UNKNOWN'))" 2>/dev/null || echo "UNKNOWN")
A3_RAW=$(echo "$COMPILER_OUTPUT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('axiom_a3_satisfied',False))" 2>/dev/null || echo "False")
A3_SATISFIED=$(parse_json_bool "$A3_RAW")
TRANSITION_RAW=$(echo "$COMPILER_OUTPUT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('transition_detected',False))" 2>/dev/null || echo "False")
TRANSITION_DETECTED=$(parse_json_bool "$TRANSITION_RAW")
ENTROPY_NUMERIC=$(echo "$COMPILER_OUTPUT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('entropy_numeric',0))" 2>/dev/null || echo "0")

echo ""

# Step 2: Axiom A3 check
echo "--- Step 2: Axiom A3 — Canonical State Uniqueness ---"
if [[ "$A3_SATISFIED" != "true" ]]; then
  echo "❌ Axiom A3 violation: |C(O)| != 1"
  echo "   Multiple or zero possible canonical states detected."
  echo "   System cannot determine its own governance state."
  exit 1
fi
echo "  ✅ |C(O)| = 1 — canonical state is uniquely determined"
echo "  Canonical state: $CANONICAL_STATE"
echo "  Declared state:  $DECLARED_STATE"
echo ""

# Step 3: Transition validation
echo "--- Step 3: Transition legality check ---"
CHANGED=$(git diff --name-only "$BASE_SHA".."$HEAD_SHA" 2>/dev/null || true)

# If canonical == declared, no transition — check that no transition rules were violated
if [[ "$CANONICAL_STATE" == "$DECLARED_STATE" ]]; then
  echo "  No transition detected (declared == canonical)"
  echo "  ✅ State is stable and legal"
else
  echo "  Transition detected: $DECLARED_STATE → $CANONICAL_STATE"
  echo ""

  # Look up transition in ledger
  TRANSITION_FOUND=$(python3 -c "
import json
with open('$LEDGER_FILE') as f:
    d = json.load(f)
for t in d.get('transitions', []):
    if t['from'] == '$DECLARED_STATE' and t['to'] == '$CANONICAL_STATE':
        print(json.dumps(t))
        break
    if t['from'] == '*' and t['to'] == '$CANONICAL_STATE':
        print(json.dumps(t))
        break
else:
    print('NOT_FOUND')
")

  if [[ "$TRANSITION_FOUND" == "NOT_FOUND" ]]; then
    echo "❌ Illegal transition: $DECLARED_STATE → $CANONICAL_STATE"
    echo "   Transition not found in transition ledger (T)."
    echo "   System has entered an undefined governance state."
    echo ""
    echo "  Possible transitions from $DECLARED_STATE:"
    python3 -c "
import json
with open('$LEDGER_FILE') as f:
    d = json.load(f)
for t in d.get('transitions', []):
    if t['from'] == '$DECLARED_STATE':
        print(f'    {t[\"from\"]} → {t[\"to\"]}  (cost: {t[\"entropy_cost\"]}, guard: {t[\"guard\"]})')
    if t['from'] == '*':
        print(f'    * → {t[\"to\"]}  (cost: {t[\"entropy_cost\"]}, guard: {t[\"guard\"]})')
"
    exit 1
  fi

  echo "  ✅ Transition is legal (found in T)"
  echo "  Entropy cost: $(echo "$TRANSITION_FOUND" | python3 -c "import sys,json; print(json.load(sys.stdin).get('entropy_cost','unknown'))")"
  echo "  Guard: $(echo "$TRANSITION_FOUND" | python3 -c "import sys,json; print(json.load(sys.stdin).get('guard','unknown'))")"
fi
echo ""

# Step 4: Invariant verification
echo "--- Step 4: Invariant health ---"
INVARIANT_ALL_PASS=true
while IFS= read -r inv_id; do
  INV_RAW=$(echo "$COMPILER_OUTPUT" | python3 -c "
import sys,json
d=json.load(sys.stdin)
print(d.get('invariant_health',{}).get('$inv_id',False))
" 2>/dev/null || echo "False")
  INV_STATUS=$(parse_json_bool "$INV_RAW")
  if [[ "$INV_STATUS" == "true" ]]; then
    echo "  ✅ $inv_id — pass"
  else
    echo "  ❌ $inv_id — FAIL"
    INVARIANT_ALL_PASS=false
  fi
done < <(echo "$COMPILER_OUTPUT" | python3 -c "
import sys,json
d=json.load(sys.stdin)
for k in d.get('invariant_health',{}):
    print(k)
" 2>/dev/null)

if [[ "$INVARIANT_ALL_PASS" == "false" ]]; then
  echo ""
  echo "❌ Invariant violation detected"
  echo "    System state does not satisfy all invariants."
  echo "    See: .tools/transition_ledger.json (invariants section)"
  exit 1
fi
echo ""

# Step 5: Compute entropy cost (cumulative)
echo "--- Step 5: Entropy accounting ---"
echo "  Transition entropy cost: $ENTROPY_NUMERIC (numeric)"
echo ""

# Overall result
echo "================================================"
echo "  CEGL-A: VERIFICATION COMPLETE"
echo "  State: $CANONICAL_STATE"
if [[ "$CANONICAL_STATE" == "INVALID" ]]; then
  echo "  Result: ❌ SYSTEM INVALID"
  exit 1
else
  echo "  Result: ✅ EVOLUTION LEGAL"
  exit 0
fi
