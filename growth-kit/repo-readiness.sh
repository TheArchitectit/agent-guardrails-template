#!/usr/bin/env bash
# Offline readiness check for a repo directory. Exit 1 if any required item is missing.
d="${1:-.}"; fail=0
chk(){ if eval "$2"; then echo "PASS  $1"; else echo "FAIL  $1"; fail=1; fi; }
chk "README.md present"            "[ -s '$d/README.md' ]"
chk "LICENSE present"              "ls '$d'/LICENSE* >/dev/null 2>&1"
chk "SECURITY.md present"          "[ -s '$d/SECURITY.md' ] || [ -s '$d/.github/SECURITY.md' ]"
chk "CONTRIBUTING present"         "[ -s '$d/CONTRIBUTING.md' ] || [ -s '$d/.github/CONTRIBUTING.md' ]"
chk "README has install section"   "grep -qi 'install' '$d/README.md' 2>/dev/null"
chk "README links a demo/image"    "grep -Eqi '\\.(gif|png|mp4|webm)|asciinema|youtu' '$d/README.md' 2>/dev/null"
chk "No private IPs in README"     "! grep -Eq '(^|[^0-9])(10\\.[0-9]+|192\\.168|172\\.(1[6-9]|2[0-9]|3[01]))\\.[0-9]+\\.[0-9]+' '$d/README.md' 2>/dev/null"
exit $fail
