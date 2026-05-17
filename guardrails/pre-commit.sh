#!/usr/bin/env bash
# Guardrails pre-commit hook
# Scans staged files for secrets and validates scope compliance
set -euo pipefail

RED='\033[0;31m'
YELLOW='\033[0;33m'
GREEN='\033[0;32m'
NC='\033[0m'

VIOLATIONS=0

# Secret patterns (same as output-validator/validator.ts)
SECRET_PATTERNS=(
  'AKIA[0-9A-Z]{16}'
  'gh[pousr]_[A-Za-z0-9_]{36,}'
  'glpat-[A-Za-z0-9\-]{20}'
  'xox[baprs]-[0-9]{10,13}-[0-9]{10,13}-[a-zA-Z0-9]{24,34}'
  'sk_live_[A-Za-z0-9]{24,}'
  '-----BEGIN (.*)PRIVATE KEY-----'
  '(postgres|mysql|mongodb|redis)://[^\s"'"'"']+'
)

echo -e "${GREEN}[guardrails]${NC} Scanning staged files..."

# Get staged files
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM 2>/dev/null || true)

if [ -z "$STAGED_FILES" ]; then
  echo -e "${GREEN}[guardrails]${NC} No staged files to scan"
  exit 0
fi

for FILE in $STAGED_FILES; do
  # Skip binary files and deleted files
  if [ ! -f "$FILE" ]; then continue; fi

  # Check for secrets
  for PATTERN in "${SECRET_PATTERNS[@]}"; do
    if grep -qE "$PATTERN" "$FILE" 2>/dev/null; then
      MATCH=$(grep -oE "$PATTERN" "$FILE" 2>/dev/null | head -1)
      echo -e "${RED}[guardrails] SECRET DETECTED${NC} in ${FILE}: ${MATCH}"
      VIOLATIONS=$((VIOLATIONS + 1))
    fi
  done

  # Check for scope compliance if .pi-guardrails.json exists
  if [ -f ".pi-guardrails.json" ]; then
    SCOPE=$(node -e "
      try {
        const c = JSON.parse(require('fs').readFileSync('.pi-guardrails.json', 'utf8'));
        const scope = c.defaultScope || [];
        console.log(scope.join('|'));
      } catch { console.log(''); }
    " 2>/dev/null || echo "")

    if [ -n "$SCOPE" ] && ! echo "$FILE" | grep -qE "^($SCOPE)"; then
      echo -e "${YELLOW}[guardrails] OUT OF SCOPE${NC} ${FILE} is outside authorized scope"
      VIOLATIONS=$((VIOLATIONS + 1))
    fi
  fi
done

if [ "$VIOLATIONS" -gt 0 ]; then
  echo -e "${RED}[guardrails] Blocking commit: ${VIOLATIONS} violation(s) found${NC}"
  echo -e "${YELLOW}[guardrails] To bypass: git commit --no-verify${NC}"
  exit 1
fi

echo -e "${GREEN}[guardrails]${NC} All checks passed"
exit 0
