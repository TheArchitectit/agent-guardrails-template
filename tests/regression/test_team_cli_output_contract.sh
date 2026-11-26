#!/usr/bin/env sh
# Regression test for FAILURE-ID: FAIL-7f20d921
# Bug: status JSON was forwarded to a backend that rejected it; query and
# history JSON serializers were unreachable; and unsupported JSON output could
# proceed through a mutating command's text path.
# Fix: serialize status in the backend, route query/history to their existing
# serializers, and reject unsupported output formats in Cobra's persistent
# pre-run before a command handler can invoke the backend.

set -eu
repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
built_binary="$repo_root/cmd/team-cli/team"
if [ -e "$built_binary" ]; then
  echo "refusing to overwrite existing build output: $built_binary" >&2
  exit 2
fi
tmp_root=$(mktemp -d)
cleanup() {
  rm -f "$built_binary"
  rm -rf "$tmp_root"
}
trap cleanup EXIT HUP INT TERM

make -C "$repo_root/cmd/team-cli" install PREFIX="$tmp_root/prefix"
mkdir -p "$tmp_root/data/.teams"
cp "$repo_root/.teams/fixtures/full-project.json" "$tmp_root/data/.teams/full-project.json"
cp "$tmp_root/data/.teams/full-project.json" "$tmp_root/before.json"
cd "$tmp_root/data"

"$tmp_root/prefix/bin/team" status -p full-project -o json > "$tmp_root/all.json"
python3 -c 'import json,sys; p=json.load(open(sys.argv[1])); assert len(p)==5 and p[0]["phase"].startswith("Phase 1")' "$tmp_root/all.json"
"$tmp_root/prefix/bin/team-cli" status -p full-project --phase "Phase 1: Strategy, Governance & Planning" -o json > "$tmp_root/phase.json"
python3 -c 'import json,sys; p=json.load(open(sys.argv[1])); assert p["total_teams"]==3 and p["phase"].startswith("Phase 1")' "$tmp_root/phase.json"
"$tmp_root/prefix/bin/team" status -p full-project > "$tmp_root/status.txt"
grep -q "Project Status" "$tmp_root/status.txt"

# These existing serializers were present but unreachable in backend dispatch.
"$tmp_root/prefix/bin/team" query -p full-project -o json > "$tmp_root/query.json"
python3 -c 'import json,sys; assert isinstance(json.load(open(sys.argv[1])),list)' "$tmp_root/query.json"
"$tmp_root/prefix/bin/team" history -p full-project -t 1 -o json > "$tmp_root/history.json"
python3 -c 'import json,sys; assert isinstance(json.load(open(sys.argv[1])),list)' "$tmp_root/history.json"

if "$tmp_root/prefix/bin/team" assign -p full-project -t 1 --role "Business Analyst" --person "No mutation" -o json > "$tmp_root/rejected.out" 2>&1; then
  echo "unsupported mutating JSON mode unexpectedly succeeded" >&2
  exit 1
fi
grep -q "JSON output is not supported" "$tmp_root/rejected.out"
if "$tmp_root/prefix/bin/team" assign -p full-project -t 1 --role "Business Analyst" --person "No mutation" -o xml > "$tmp_root/rejected-format.out" 2>&1; then
  echo "unknown output format unexpectedly succeeded" >&2
  exit 1
fi
grep -q "unsupported output format" "$tmp_root/rejected-format.out"
cmp "$tmp_root/before.json" .teams/full-project.json

echo "PASS: installed-prefix output contract and pre-effect rejection"
