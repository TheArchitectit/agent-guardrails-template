#!/usr/bin/env bash
# QA-04: the shipping dependency tree passed while development tools had advisories.
set -u
cd "$(dirname "$0")/../../ide/vscode-extension" || exit 1
result=0
npm audit --audit-level=low || result=1
npm audit --omit=dev --audit-level=low || result=1
exit "$result"
