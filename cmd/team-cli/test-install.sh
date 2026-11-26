#!/usr/bin/env bash
set -euo pipefail

CLI_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$CLI_DIR/../.." && pwd)"
BUILT_BINARY="$CLI_DIR/team"
if [[ -e "$BUILT_BINARY" ]]; then
  echo "refusing to overwrite existing build output: $BUILT_BINARY" >&2
  exit 2
fi

TMP_ROOT="$(mktemp -d)"
cleanup() {
  rm -f "$BUILT_BINARY"
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT

if make -C "$CLI_DIR" -n uninstall PREFIX= >/dev/null 2>&1; then
  echo "Makefile must reject an empty PREFIX before uninstall." >&2
  exit 1
fi
for blank_prefix in '' '   '; do
  if PREFIX="$blank_prefix" bash "$CLI_DIR/install.sh" >"$TMP_ROOT/blank-prefix.out" 2>&1; then
    echo "install.sh accepted an empty or whitespace PREFIX" >&2
    exit 1
  fi
done
if make -C "$CLI_DIR" install PREFIX="$TMP_ROOT/missing-make-prefix" TEAM_MANAGER_SOURCE="$TMP_ROOT/empty-backend" >"$TMP_ROOT/missing-make-backend.out" 2>&1; then
  echo "Makefile accepted an incomplete backend payload" >&2
  exit 1
fi
[[ ! -e "$TMP_ROOT/missing-make-prefix/bin/team" ]]

MAKE_PREFIX="$TMP_ROOT/make-prefix"
SHELL_PREFIX="$TMP_ROOT/shell-prefix"
OUTSIDE="$TMP_ROOT/outside-checkout"
mkdir -p "$OUTSIDE/.teams"
cp "$REPO_ROOT/.teams/fixtures/full-project.json" "$OUTSIDE/.teams/full-project.json"

run_installed_status() {
  local binary="$1"
  local output="$2"
  (
    cd "$OUTSIDE"
    "$binary" list -p full-project
  ) > "$output"
  grep -q "Business & Product Strategy" "$output"
}

assert_payload() {
  local prefix="$1"
  local scripts="$prefix/share/agent-guardrails-template/team-cli/scripts"
  for file in team_manager.py batch_operations.py encryption.py migrations/__init__.py migrations/v1_to_v2.py; do
    [[ -f "$scripts/$file" ]] || { echo "missing installed payload: $scripts/$file" >&2; exit 1; }
  done
}

echo "Testing Makefile installation..."
make -C "$CLI_DIR" install PREFIX="$MAKE_PREFIX"
assert_payload "$MAKE_PREFIX"
[[ -x "$MAKE_PREFIX/bin/team" && -L "$MAKE_PREFIX/bin/team-cli" ]]
run_installed_status "$MAKE_PREFIX/bin/team" "$TMP_ROOT/make-status.json"
RELATIVE_PREFIX=".test-relative-prefix"
make -C "$CLI_DIR" install PREFIX="$RELATIVE_PREFIX"
[[ -L "$CLI_DIR/$RELATIVE_PREFIX/bin/team-cli" ]]
[[ "$(readlink "$CLI_DIR/$RELATIVE_PREFIX/bin/team-cli")" == team ]]
"$CLI_DIR/$RELATIVE_PREFIX/bin/team-cli" --version >/dev/null
make -C "$CLI_DIR" uninstall PREFIX="$RELATIVE_PREFIX"
rm -rf "$CLI_DIR/$RELATIVE_PREFIX"
cp -R "$MAKE_PREFIX/share/agent-guardrails-template/team-cli/scripts" "$TMP_ROOT/expected-scripts"

echo "Testing TEAM_MANAGER_PATH precedence and uninstall ownership..."
OVERRIDE="$TMP_ROOT/user-override.py"
OVERRIDE_MARKER="$TMP_ROOT/override-called"
cat > "$OVERRIDE" <<'PY'
import os
from pathlib import Path

Path(os.environ["TEAM_OVERRIDE_MARKER"]).write_text("called", encoding="utf-8")
print("user override backend invoked")
PY
(
  cd "$OUTSIDE"
  TEAM_MANAGER_PATH="$OVERRIDE" TEAM_OVERRIDE_MARKER="$OVERRIDE_MARKER" \
    "$MAKE_PREFIX/bin/team" list -p full-project
) > "$TMP_ROOT/override-status.txt"
grep -q "user override backend invoked" "$TMP_ROOT/override-status.txt"
[[ -f "$OVERRIDE_MARKER" ]]

mkdir -p "$MAKE_PREFIX/share/agent-guardrails-template"
printf 'user-owned\n' > "$MAKE_PREFIX/share/agent-guardrails-template/user-data.txt"
mkdir -p "$MAKE_PREFIX/share/agent-guardrails-template/team-cli/scripts/migrations"
printf 'user-owned\n' > "$MAKE_PREFIX/share/agent-guardrails-template/team-cli/scripts/migrations/local.py"
make -C "$CLI_DIR" uninstall PREFIX="$MAKE_PREFIX"
[[ ! -e "$MAKE_PREFIX/bin/team" ]]
[[ ! -e "$MAKE_PREFIX/bin/team-cli" ]]
[[ -f "$MAKE_PREFIX/share/agent-guardrails-template/team-cli/scripts/migrations/local.py" ]]
[[ -f "$MAKE_PREFIX/share/agent-guardrails-template/user-data.txt" ]]
[[ -f "$OVERRIDE" && -f "$OVERRIDE_MARKER" ]]

echo "Testing symlink-safe uninstall..."
LINK_PREFIX="$TMP_ROOT/link-prefix"
LINK_TARGET="$TMP_ROOT/link-target"
mkdir -p "$LINK_PREFIX/bin" "$LINK_PREFIX/share/agent-guardrails-template/team-cli/scripts" "$LINK_TARGET"
cp "$CLI_DIR/team" "$LINK_PREFIX/bin/team"
printf 'sentinel\n' > "$LINK_TARGET/keep.txt"
ln -s "$LINK_TARGET" "$LINK_PREFIX/share/agent-guardrails-template/team-cli/scripts/migrations"
if make -C "$CLI_DIR" uninstall PREFIX="$LINK_PREFIX" >"$TMP_ROOT/symlink-uninstall.out" 2>&1; then
  echo "uninstall accepted a substituted migrations symlink" >&2
  exit 1
fi
[[ -f "$LINK_PREFIX/bin/team" && -f "$LINK_TARGET/keep.txt" ]]
rm "$LINK_PREFIX/share/agent-guardrails-template/team-cli/scripts/migrations"

echo "Testing shell installer backend preflight..."
FAKE_ROOT="$TMP_ROOT/fake-repo"
mkdir -p "$FAKE_ROOT/cmd/team-cli" "$FAKE_ROOT/scripts" "$TMP_ROOT/fake-cwd"
cp "$CLI_DIR/install.sh" "$FAKE_ROOT/cmd/team-cli/install.sh"
cp "$CLI_DIR/team" "$TMP_ROOT/fake-cwd/team"
if (cd "$TMP_ROOT/fake-cwd" && PREFIX="$TMP_ROOT/missing-prefix" bash "$FAKE_ROOT/cmd/team-cli/install.sh") >"$TMP_ROOT/missing-backend.out" 2>&1; then
  echo "install.sh accepted a standalone binary without backend payload" >&2
  exit 1
fi
[[ ! -e "$TMP_ROOT/missing-prefix/bin/team" ]]

echo "Testing install.sh payload parity..."
make -C "$CLI_DIR" build
(
  cd "$OUTSIDE"
  PREFIX="$SHELL_PREFIX" bash "$CLI_DIR/install.sh"
)
assert_payload "$SHELL_PREFIX"
[[ -x "$SHELL_PREFIX/bin/team" && -L "$SHELL_PREFIX/bin/team-cli" ]]
for file in team_manager.py batch_operations.py encryption.py migrations/__init__.py migrations/v1_to_v2.py; do
  cmp "$TMP_ROOT/expected-scripts/$file" \
    "$SHELL_PREFIX/share/agent-guardrails-template/team-cli/scripts/$file"
done
run_installed_status "$SHELL_PREFIX/bin/team" "$TMP_ROOT/shell-status.json"

mkdir -p "$SHELL_PREFIX/share/agent-guardrails-template"
printf 'user-owned\n' > "$SHELL_PREFIX/share/agent-guardrails-template/user-data.txt"
mkdir -p "$SHELL_PREFIX/share/agent-guardrails-template/team-cli/scripts/migrations"
printf 'user-owned\n' > "$SHELL_PREFIX/share/agent-guardrails-template/team-cli/scripts/migrations/local.py"
make -C "$CLI_DIR" uninstall PREFIX="$SHELL_PREFIX"
[[ -f "$SHELL_PREFIX/share/agent-guardrails-template/team-cli/scripts/migrations/local.py" ]]
[[ -f "$SHELL_PREFIX/share/agent-guardrails-template/user-data.txt" ]]
[[ -f "$OVERRIDE" && -f "$OVERRIDE_MARKER" ]]

echo "PASS: Makefile and shell installs resolve the packaged backend outside the checkout."
