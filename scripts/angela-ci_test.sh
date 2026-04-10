#!/usr/bin/env bash
# angela-ci_test.sh — minimal sanity test for the severity-counting grep
# patterns used by angela-ci.sh. Runs the same patterns against a fixture
# that mirrors the real `lore angela draft --all` output format and fails
# if the counts don't match the expected values.
#
# Run: ./scripts/angela-ci_test.sh
#
# This test exists because the original grep pattern (`^  warning`) silently
# counted 0 warnings once draft --all started printing inline suggestions
# indented with 9 spaces, which made `--fail-on warning` always pass.

set -euo pipefail

FAIL=0

assert_eq() {
  local name="$1" expected="$2" actual="$3"
  if [[ "$expected" != "$actual" ]]; then
    echo "FAIL: $name — expected $expected, got $actual" >&2
    FAIL=1
  else
    echo "PASS: $name"
  fi
}

# --- Fixture: real draft --all output with inline warnings (9-space indent) ---
FIXTURE_DRAFT_ALL=$(cat <<'EOF'
lore angela draft --all — 3 documents

  B    review   decision-auth-2026.md    3 suggestions (2 warnings)
         warning  structure      Section "## What" is missing
         warning  completeness   "Why" section is too brief
         info     coherence      Related: feature-api-2026.md
  A    ok       good-doc.md
  D    review   bad.md                     1 suggestions
         warning  structure      Body too short

2/3 documents need attention. 4 total suggestions.
EOF
)

# --- Fixture: single-file draft output (2-space indent) ---
FIXTURE_DRAFT_SINGLE=$(cat <<'EOF'
lore angela draft — decision-auth.md
  Reviewed by: Sialou + Doumbia

  error    structure       Missing "Impact" section
  warning  tone            Avoid "just"
  info     coherence       Related: feature-api.md

3 suggestions
EOF
)

# --- Fixture: review output (contradiction + gap) ---
FIXTURE_REVIEW=$(cat <<'EOF'
lore angela review — 10 documents

  contradiction  decision-a.md vs decision-b.md
  gap            No doc covering auth flow
  obsolete       decision-legacy.md references deleted API

3 findings
EOF
)

# --- Run the same grep patterns as angela-ci.sh ---
count_errors() {
  echo "$1" | grep -cE "^[[:space:]]+(error|contradiction)[[:space:]]" || true
}
count_warnings() {
  echo "$1" | grep -cE "^[[:space:]]+(warning|gap|obsolete|style)[[:space:]]" || true
}

# --- Assertions ---
assert_eq "draft --all errors"     "0" "$(count_errors "$FIXTURE_DRAFT_ALL")"
assert_eq "draft --all warnings"   "3" "$(count_warnings "$FIXTURE_DRAFT_ALL")"

assert_eq "draft single errors"    "1" "$(count_errors "$FIXTURE_DRAFT_SINGLE")"
assert_eq "draft single warnings"  "1" "$(count_warnings "$FIXTURE_DRAFT_SINGLE")"

assert_eq "review errors"          "1" "$(count_errors "$FIXTURE_REVIEW")"
assert_eq "review warnings"        "2" "$(count_warnings "$FIXTURE_REVIEW")"

# --- Regression guard: empty output must produce 0 (not crash) ---
assert_eq "empty errors"           "0" "$(count_errors "")"
assert_eq "empty warnings"         "0" "$(count_warnings "")"

if [[ "$FAIL" -ne 0 ]]; then
  echo ""
  echo "One or more assertions failed." >&2
  exit 1
fi

echo ""
echo "All angela-ci severity counting tests passed."
