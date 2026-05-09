#!/bin/bash
# check-doc-coverage.sh: Enforce a minimum Go documentation coverage threshold.
# This is a lightweight docs lint gate backed by go-stats-generator output.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

THRESHOLD="${DOC_COVERAGE_MIN:-80.0}"

analysis_output="$(go-stats-generator analyze . --skip-tests 2>&1)"
overall_line="$(printf '%s\n' "$analysis_output" | grep '^Overall Coverage:' | head -1 || true)"

if [ -z "$overall_line" ]; then
  echo "ERROR: Unable to parse documentation coverage from go-stats-generator output."
  exit 1
fi

current="$(printf '%s\n' "$overall_line" | awk '{print $3}' | tr -d '%')"

echo "Documentation coverage: ${current}%"
echo "Required minimum: ${THRESHOLD}%"

if awk -v cur="$current" -v min="$THRESHOLD" 'BEGIN {exit (cur+0 >= min+0) ? 0 : 1}'; then
  echo "PASS: documentation coverage threshold met"
else
  echo "FAIL: documentation coverage below threshold"
  exit 1
fi
