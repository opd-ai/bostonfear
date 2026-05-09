#!/bin/bash
# check-common-deps.sh: Enforce that serverengine/common/* packages do not import
# from game-specific modules (arkhamhorror, eldersign, eldritchhorror, finalhour).
# This ensures the common package remains game-family-agnostic.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

VIOLATIONS=0

# List of game-specific modules that should not be imported by common packages
GAME_MODULES=(
  "arkhamhorror"
  "eldersign"
  "eldritchhorror"
  "finalhour"
)

# Check each common package
for common_pkg in serverengine/common/*/; do
  if [ ! -d "$common_pkg" ]; then
    continue
  fi
  
  common_pkg_name=$(basename "$common_pkg")
  
  for game_module in "${GAME_MODULES[@]}"; do
    # Search for imports of game-specific modules in this common package
    if grep -r "import.*$game_module" "$common_pkg" --include="*.go" 2>/dev/null | grep -v "^Binary"; then
      echo "ERROR: $common_pkg_name imports game-specific module '$game_module'"
      VIOLATIONS=$((VIOLATIONS + 1))
    fi
  done
done

if [ $VIOLATIONS -gt 0 ]; then
  echo ""
  echo "FAILED: Found $VIOLATIONS dependency violation(s)"
  echo "serverengine/common packages must not import from game-specific modules."
  exit 1
else
  echo "✓ All common packages are free of game-specific imports."
  exit 0
fi
