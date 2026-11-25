#!/usr/bin/env bash
set -euo pipefail

subjects_file=${1:?usage: count-bug-fix-commits.sh SUBJECTS_FILE}
github_output=${GITHUB_OUTPUT:?GITHUB_OUTPUT must be set by GitHub Actions}

bug_fix_count=$(grep -cE '^(fix|bugfix|hotfix)(\(.+\))?:' "$subjects_file" || true)
printf 'bug_fix_count=%s\n' "$bug_fix_count" >> "$github_output"
printf '%s\n' "$bug_fix_count"
