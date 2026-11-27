#!/usr/bin/env bash
# Run the repository's inventory-driven Python and Go checks locally.
# This command never installs tools/packages or starts/stops containers.
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/run_local_ci.sh

Run approved AIGGP charset checks and self-test, then the repository-declared
Python and Go suites/builds with their discovery gates and positive floors.
Python coverage is written to ./coverage.xml. Prerequisites must already be
installed; nothing is installed automatically.

The Docker/Podman context integration test is skipped by default. To opt in,
set MCP_DOCKER_CONTEXT_ENGINE=docker or podman (or the workflow-compatible
MCP_DOCKER_CONTEXT_DAEMON=available). An opted-in engine must already be
installed and its daemon available. This script will only query engine info;
the test itself owns any temporary resources it creates.
EOF
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi
if [[ $# -ne 0 ]]; then
  usage >&2
  exit 2
fi

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || {
  echo "error: run this command from inside a Git checkout" >&2
  exit 2
}
cd "$repo_root"

missing=()
for tool in python3 go jq git; do
  command -v "$tool" >/dev/null 2>&1 || missing+=("$tool")
done
if ((${#missing[@]})); then
  printf 'error: missing required tool(s): %s\n' "${missing[*]}" >&2
  echo "Install prerequisites yourself, then rerun; this command never installs tools." >&2
  exit 2
fi

python3 - <<'PY'
import importlib.util
import sys

required = {"pytest": "pytest", "pytest_cov": "pytest-cov", "yaml": "PyYAML"}
missing = [package for module, package in required.items()
           if importlib.util.find_spec(module) is None]
if missing:
    print("error: missing Python test dependency/dependencies: " + ", ".join(missing), file=sys.stderr)
    print("Install prerequisites yourself; this command never runs pip install.", file=sys.stderr)
    raise SystemExit(2)
PY

python_version="$(python3 --version 2>&1)"
go_version="$(go version 2>&1)"
printf 'Local CI: %s; %s.\n' "$python_version" "$go_version"

# The workflow uses this exact approved checkout and verifies its identity and
# cleanliness before invoking either gate.
aiggp_sha=7d17d6f1baa4357110e57d3a6ffd55fbacdd77f7
if [[ ! -d .devgate/.git ]]; then
  echo "error: approved AIGGP checkout is missing at .devgate" >&2
  echo "Prepare it manually: clone TheArchitectit/AIGGP-Agentic-Framework into .devgate and check out $aiggp_sha." >&2
  exit 2
fi
actual_sha="$(git -C .devgate rev-parse --verify HEAD 2>/dev/null || true)"
[[ "$actual_sha" == "$aiggp_sha" ]] || {
  echo "error: .devgate must be at approved AIGGP commit $aiggp_sha (found ${actual_sha:-unreadable})" >&2
  exit 2
}
origin="$(git -C .devgate remote get-url origin 2>/dev/null || true)"
case "$origin" in
  https://github.com/TheArchitectit/AIGGP-Agentic-Framework|https://github.com/TheArchitectit/AIGGP-Agentic-Framework.git|git@github.com:TheArchitectit/AIGGP-Agentic-Framework.git) ;;
  *) echo "error: unexpected .devgate origin: ${origin:-missing}" >&2; exit 2 ;;
esac
tool_status="$(git -C .devgate status --porcelain --untracked-files=all)"
[[ -z "$tool_status" ]] || {
  echo "error: approved AIGGP checkout is not clean:" >&2
  printf '%s\n' "$tool_status" >&2
  exit 2
}
git ls-files --error-unmatch -- .devgate >/dev/null 2>&1 && {
  echo "error: .devgate must remain an ignored, untracked consumer checkout" >&2
  exit 2
}
git check-ignore -q -- .devgate || {
  echo "error: .devgate is not ignored by this checkout" >&2
  exit 2
}

# Make engine-backed coverage an explicit choice. Never start a daemon or
# create/clean containers here; the integration test manages its fixture.
if [[ -n "${MCP_DOCKER_CONTEXT_ENGINE:-}" ]]; then
  case "${MCP_DOCKER_CONTEXT_ENGINE,,}" in docker|podman) ;; *)
    echo "error: MCP_DOCKER_CONTEXT_ENGINE must be docker or podman" >&2; exit 2 ;;
  esac
  engine="${MCP_DOCKER_CONTEXT_ENGINE,,}"
elif [[ "${MCP_DOCKER_CONTEXT_DAEMON:-}" == available ]]; then
  engine=docker
else
  engine=""
fi
if [[ -n "$engine" ]]; then
  command -v "$engine" >/dev/null 2>&1 || {
    echo "error: explicitly selected engine '$engine' is not installed" >&2; exit 2
  }
  if ! "$engine" info >/dev/null 2>&1; then
    echo "error: explicitly selected engine '$engine' daemon is unavailable; no containers were started" >&2
    exit 2
  fi
  export MCP_DOCKER_CONTEXT_ENGINE="$engine"
  echo "Engine-backed context integration test enabled with $engine."
else
  unset MCP_DOCKER_CONTEXT_ENGINE MCP_DOCKER_CONTEXT_DAEMON || true
  echo "Engine-backed context integration test disabled (default)."
fi

echo "== Approved AIGGP charset gate and self-test =="
python3 .devgate/scripts/check_charset.py --repo .
python3 .devgate/tests/test_check_charset.py

echo "== Python discovery gate and self-test =="
python3 scripts/check_python_tests.py --repo .
python3 scripts/test_check_python_tests.py

echo "== Deterministic Regression Guard gates =="
python3 scripts/check_action_pins.py --repo .
python3 scripts/test_check_action_pins.py
python3 scripts/check_resource_claims.py --repo .
python3 scripts/test_check_resource_claims.py
python3 scripts/check_image_pins.py --repo .
python3 scripts/test_check_image_pins.py
python3 scripts/check_secret_argv.py --repo .
python3 scripts/test_check_secret_argv.py
python3 scripts/check_secret_inventory.py --repo .
python3 scripts/test_check_secret_inventory.py
python3 scripts/check_runner_isolation.py --repo .
python3 scripts/test_check_runner_isolation.py
python3 scripts/check_mandatory_rows.py --repo .
python3 scripts/test_check_mandatory_rows.py
python3 scripts/check_tool_count.py --repo .
python3 scripts/test_check_tool_count.py
python3 scripts/check_policy_pack_lock.py --repo .
python3 scripts/test_check_policy_pack_lock.py

echo "== Every Python inventory suite =="
while IFS=$'\t' read -r path floor command; do
  [[ -n "$path" ]] || continue
  echo "-- $path (floor $floor): $command"
  (cd "$repo_root" && bash -euo pipefail -c "$command")
  count="$(python3 -m pytest "$path" --collect-only -q -o addopts='' 2>/dev/null \
    | grep -oE '[0-9]+ tests? collected' | grep -oE '^[0-9]+' | tail -1 || true)"
  count="${count:-0}"
  echo "$path: collected $count tests (floor: $floor)"
  if ((count < floor)); then
    echo "error: Python suite $path collected $count tests, below floor $floor" >&2
    exit 1
  fi
done < <(python3 - <<'PY'
import json
from pathlib import Path
for row in json.loads(Path(".guardrails/python-test-floor.json").read_text())["suites"]:
    print("\t".join((row["path"], str(row["min_tests"]), row["test_command"])))
PY
)

python3 - <<'PY'
import xml.etree.ElementTree as ET
from pathlib import Path

path = Path("coverage.xml")
if not path.is_file():
    raise SystemExit("error: coverage.xml was not generated by the declared Python suite")
root = ET.parse(path).getroot()
if root.tag != "coverage" or int(root.get("lines-valid", "0")) < 1:
    raise SystemExit("error: coverage.xml must contain a non-empty coverage report")
print(f"Coverage report validated: {path} ({root.get('lines-valid')} lines)")
PY

echo "== Go module discovery gate and self-test =="
python3 scripts/check_go_modules.py --repo .
python3 scripts/test_check_go_modules.py

echo "== Every Go inventory module =="
while IFS=$'\t' read -r path floor test_command build_command; do
  [[ -n "$path" ]] || continue
  echo "-- $path (floor $floor)"
  # Race builds and mutation fixtures can take longer on a local machine than
  # on a CI runner. Keep the assertions unchanged and allow 30 minutes per test.
  case "$test_command" in
    "go test "*) test_command="go test -timeout=30m ${test_command#go test }" ;;
  esac
  echo "test: $test_command"
  (cd "$path" && CGO_ENABLED=1 bash -euo pipefail -c "$test_command")
  echo "build: $build_command"
  (cd "$path" && CGO_ENABLED=1 bash -euo pipefail -c "$build_command")
  listing="$(cd "$path" && CGO_ENABLED=1 go test -list '^Test' ./...)"
  count="$(printf '%s\n' "$listing" | grep -c '^Test' || true)"
  echo "$path: discovered $count tests (floor: $floor)"
  if ((count < floor)); then
    echo "error: Go module $path discovered $count tests, below floor $floor" >&2
    exit 1
  fi
done < <(jq -r '.modules[] | [.path, (.min_tests|tostring), .test_command, .build_command] | @tsv' .guardrails/test-floor.json)

echo "Local CI checks passed. Coverage report: $repo_root/coverage.xml"
