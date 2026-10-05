#!/usr/bin/env sh
# Full-history secret scan for the R19-06 gate.
#
# Runs Gitleaks over the entire git history (every commit, not just the working
# tree) so a secret that was committed and later removed is still detected.
#
# Exit codes:
#   0    scan ran and found no leaks
#   1    scan ran and found leaks (a real finding blocks)
#   127  gitleaks is not installed (NOT_RUN — never a silent pass)
#
# Extra arguments are forwarded to gitleaks (e.g. --redact, -f json).
set -eu

if ! command -v gitleaks >/dev/null 2>&1; then
    echo "gitleaks-history: NOT_RUN (gitleaks not installed on PATH)" >&2
    exit 127
fi

# gitleaks >= 8.19 uses the 'git' subcommand; older releases use 'detect'.
if gitleaks git --help >/dev/null 2>&1; then
    exec gitleaks git --log-opts=--all --no-banner "$@"
fi
exec gitleaks detect --source . --log-opts=--all --no-banner "$@"
