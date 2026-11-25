#!/usr/bin/env bash
set -Eeuo pipefail

readonly VERSION=8.24.3
readonly ARCHIVE_SHA256=9991e0b2903da4c8f6122b5c3186448b927a5da4deef1fe45271c3793f4ee29c
readonly RELEASE_URL="https://github.com/gitleaks/gitleaks/releases/download/v${VERSION}/gitleaks_${VERSION}_linux_x64.tar.gz"

summary_result() {
  if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
    printf 'Gitleaks verification/scan: %s\n' "$1" >> "$GITHUB_STEP_SUMMARY"
  fi
}
fail() { summary_result failed; echo "gitleaks-verified: $1" >&2; exit 1; }
command -v git >/dev/null || fail "git is required"
[[ "$(uname -s)" == Linux && "$(uname -m)" == x86_64 ]] || fail "only Linux x86_64 is supported"

resolve_commit() {
  local input=$1 resolved
  [[ "$input" =~ ^[0-9a-fA-F]{40}$ ]] || return 1
  resolved=$(git rev-parse --verify "${input}^{commit}" 2>/dev/null) || return 1
  [[ "$resolved" =~ ^[0-9a-f]{40}$ ]] || return 1
  printf '%s' "$resolved"
}
is_zero() { [[ "$1" =~ ^0{40}$ ]]; }

range=
if [[ $# == 2 ]]; then
  [[ "$(git rev-parse --is-shallow-repository)" == false ]] || fail "shallow checkout cannot prove scan history"
  base=$(resolve_commit "$1") || fail "local base must resolve to a full commit"
  head=$(resolve_commit "$2") || fail "local head must resolve to a full commit"
  git merge-base "$base" "$head" >/dev/null 2>&1 || fail "local commits have no merge base"
  merge_base=$(git merge-base "$base" "$head") || fail "local merge base unavailable"
  range="${merge_base}..${head}"
elif [[ $# == 0 && -n "${GITHUB_EVENT_NAME:-}" ]]; then
  [[ "$(git rev-parse --is-shallow-repository)" == false ]] || fail "shallow checkout cannot prove scan history"
  event_file=${GITHUB_EVENT_PATH:?GitHub event file is required}
  [[ -f "$event_file" ]] || fail "GitHub event file is unavailable"
  event_data=$(python3 - "$event_file" "${GITHUB_EVENT_NAME}" <<'PY'
import json, sys
try:
    event = json.load(open(sys.argv[1], encoding="utf-8"))
    if sys.argv[2] == "pull_request":
        values = [event["pull_request"]["base"]["sha"], event["pull_request"]["head"]["sha"]]
    elif sys.argv[2] == "push":
        values = [event["before"], event["after"]]
    else:
        raise ValueError("unsupported event")
    if any(not isinstance(value, str) for value in values):
        raise ValueError("invalid commit fields")
    print("\n".join(values))
except Exception:
    sys.exit(1)
PY
  ) || fail "GitHub event could not be parsed"
  readarray -t event_commits <<< "$event_data"
  [[ ${#event_commits[@]} == 2 ]] || fail "event commit boundaries are invalid"
  base_value=${event_commits[0]}; head_value=${event_commits[1]}
  head=$(resolve_commit "$head_value") || fail "event head does not resolve to a full commit"
  if [[ "$GITHUB_EVENT_NAME" == pull_request ]]; then
    base=$(resolve_commit "$base_value") || fail "pull-request base does not resolve to a full commit"
    merge_base=$(git merge-base "$base" "$head") || fail "pull-request merge base unavailable"
    range="${merge_base}..${head}"
  else
    [[ "$base_value" =~ ^[0-9a-fA-F]{40}$ ]] || fail "push before boundary is malformed"
  fi
  if [[ "$GITHUB_EVENT_NAME" == push ]]; then
    if is_zero "$base_value"; then
      range="$head"
    elif base=$(resolve_commit "$base_value"); then
      if git merge-base --is-ancestor "$base" "$head"; then
        range="${base}..${head}"
      else
        range="$head"
      fi
    else
      range="$head"
    fi
  fi
else
  fail "provide a pinned BASE HEAD pair locally or run inside a supported GitHub event"
fi

if [[ "$range" == *..* && "$range" == "${range%%..*}..${range##*..}" ]]; then
  left=${range%%..*}; right=${range##*..}
  [[ -n "$left" && -n "$right" ]] || fail "empty scan range"
  if [[ "$left" == "$right" ]]; then range="-1 $right"; fi
fi

command -v curl >/dev/null || fail "curl is required"
command -v sha256sum >/dev/null || fail "sha256sum is required"
command -v tar >/dev/null || fail "tar is required"
tmp_parent=${RUNNER_TEMP:-${TMPDIR:-/tmp}}
mkdir -p -- "$tmp_parent"
tmp_parent=$(cd -- "$tmp_parent" && pwd -P)
work=$(mktemp -d "$tmp_parent/gitleaks-verified.XXXXXXXX")
cleanup() { rm -rf -- "$work"; }
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

archive="$work/gitleaks.tar.gz"
curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' \
  --tlsv1.2 --connect-timeout 15 --max-time 120 --output "$archive" "$RELEASE_URL" || fail "release download failed"
actual=$(sha256sum "$archive" | awk '{print $1}')
[[ "$actual" == "$ARCHIVE_SHA256" ]] || fail "release archive digest mismatch"
tar -xzf "$archive" -C "$work" gitleaks || fail "release extraction failed"
scanner="$work/gitleaks"
[[ -f "$scanner" && -x "$scanner" ]] || fail "verified executable missing"
reported=$("$scanner" version 2>&1) || fail "version check failed"
[[ "$reported" == "$VERSION" || "$reported" == "v$VERSION" ]] || fail "unexpected scanner version"

set +e
"$scanner" git --no-banner --redact=100 --exit-code 2 --log-opts="$range" >"$work/scan.out" 2>&1
status=$?
set -e
if [[ "$status" == 0 ]]; then
  summary_result "clean (verified Gitleaks ${VERSION})"
  echo "gitleaks-verified: clean"
elif [[ "$status" == 2 ]]; then
  summary_result "findings detected (verified Gitleaks ${VERSION})"
  echo "gitleaks-verified: findings detected" >&2
  exit 2
else
  fail "scanner failed (exit ${status})"
fi
