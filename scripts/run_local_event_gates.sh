#!/usr/bin/env bash
# Run local equivalents of the event-scoped Secret Validation and Guardrails
# Lint checks. Requires the event range; never guesses one.
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/run_local_event_gates.sh \
    --event push|pull_request \
    --base-sha <full SHA-1> \
    --head-sha <full SHA-1> \
    [--base-ref <branch>] \
    [--pr-head-sha <full SHA-1>] # required for pull_request

The checkout must be clean and at --head-sha. The existing range helper may
deepen a shallow checkout or fetch a PR base ref. A pull request's
origin/<base-ref> must resolve to --base-sha after that. A new-branch push may
use 40 zeroes as --base-sha only when origin/main or origin/master is available.
USAGE
}

event=
base_sha=
head_sha=
base_ref=
pr_head_sha=
while (($#)); do
  case "$1" in
    --event) event="${2:?--event needs a value}"; shift 2 ;;
    --base-sha) base_sha="${2:?--base-sha needs a value}"; shift 2 ;;
    --head-sha) head_sha="${2:?--head-sha needs a value}"; shift 2 ;;
    --base-ref) base_ref="${2:?--base-ref needs a value}"; shift 2 ;;
    --pr-head-sha) pr_head_sha="${2:?--pr-head-sha needs a value}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "error: unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

fail() { echo "local event gates: $*" >&2; exit 2; }
[[ "$event" == push || "$event" == pull_request ]] || fail "--event must be push or pull_request"
[[ "$base_sha" =~ ^([0-9a-f]{40})$ ]] || fail "--base-sha must be a full lowercase SHA-1 (or all zeroes for a new push)"
[[ "$head_sha" =~ ^[0-9a-f]{40}$ ]] || fail "--head-sha must be a full lowercase SHA-1"
if [[ "$event" == pull_request ]]; then
  [[ -n "$base_ref" ]] || fail "--base-ref is required for pull_request"
  [[ "$pr_head_sha" =~ ^[0-9a-f]{40}$ ]] || fail "--pr-head-sha is required for pull_request"
  git check-ref-format "refs/heads/$base_ref" >/dev/null 2>&1 || fail "invalid --base-ref"
else
  [[ -z "$base_ref" ]] || fail "--base-ref is only valid for pull_request"
  [[ -z "$pr_head_sha" ]] || fail "--pr-head-sha is only valid for pull_request"
fi

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || fail "run inside a Git checkout"
cd "$repo_root"
actual_head="$(git rev-parse --verify 'HEAD^{commit}' 2>/dev/null)" || fail "HEAD is not a readable commit"
[[ "$actual_head" == "$head_sha" ]] || fail "checkout HEAD $actual_head does not match --head-sha $head_sha"
if [[ -n "$(git status --porcelain --untracked-files=all)" ]]; then
  fail "working tree is not clean; use a clean checkout at the event head"
fi
if [[ "$base_sha" == 0000000000000000000000000000000000000000 ]]; then
  default_ref=
  for candidate in origin/main origin/master; do
    if git rev-parse --verify --quiet "$candidate^{commit}" >/dev/null; then default_ref="$candidate"; break; fi
  done
  [[ -n "$default_ref" ]] || fail "new-branch push has no origin/main or origin/master merge base"
fi

work="$(mktemp -d "${TMPDIR:-/tmp}/local-event-gates.XXXXXXXX")"
cleanup() { rm -rf -- "$work"; }
trap cleanup EXIT HUP INT TERM

# Create a minimal event payload for the pinned Gitleaks scanner. This invokes
# the workflow's own verified scanner and range logic; no credentials are used.
if [[ "$event" == push ]]; then
  python3 - "$work/event.json" "$base_sha" "$head_sha" <<'PY'
import json, sys
path, before, after = sys.argv[1:]
with open(path, "w", encoding="utf-8") as f:
    json.dump({"before": before, "after": after}, f)
PY
else
  python3 - "$work/event.json" "$base_sha" "$pr_head_sha" <<'PY'
import json, sys
path, base, head = sys.argv[1:]
with open(path, "w", encoding="utf-8") as f:
    json.dump({"pull_request": {"base": {"sha": base}, "head": {"sha": head}}}, f)
PY
fi
export GITHUB_EVENT_NAME="$event" GITHUB_EVENT_PATH="$work/event.json"
export EVENT_NAME="$event" REF_SHA="$head_sha"
if [[ "$event" == push ]]; then
  export BEFORE_SHA="$base_sha"
else
  export BASE_REF="$base_ref"
fi

# Verify that the exact workflow change helpers can derive this event's set.
# `changed_files.sh` owns any full-history fetch, and fails if that cannot
# recover the complete range. Validate the pinned commit object after it runs;
# never accept the helper's documented fallback when the requested base is gone.
scripts/changed_files.sh > "$work/changed-paths.txt"
COMMIT_SHAS="$(scripts/changed_commits.sh --format '%H')" || fail "could not derive event commit range"
COMMIT_SUMMARIES="$(scripts/changed_commits.sh --format '%h %s')" || fail "could not derive event commit summaries"
if [[ "$base_sha" != 0000000000000000000000000000000000000000 ]]; then
  resolved_base="$(git rev-parse --verify "$base_sha^{commit}" 2>/dev/null)" || fail "--base-sha is unavailable after the event-range helper attempted to deepen/fetch history"
  [[ "$resolved_base" == "$base_sha" ]] || fail "--base-sha did not resolve exactly"
fi
if [[ "$event" == pull_request ]]; then
  remote_base="$(git rev-parse --verify "refs/remotes/origin/$base_ref^{commit}" 2>/dev/null)" || fail "origin/$base_ref remains unavailable after the event-range helper"
  [[ "$remote_base" == "$base_sha" ]] || fail "origin/$base_ref ($remote_base) does not match pinned --base-sha $base_sha"
  resolved_pr_head="$(git rev-parse --verify "$pr_head_sha^{commit}" 2>/dev/null)" || fail "--pr-head-sha is unavailable after the event-range helper"
  [[ "$resolved_pr_head" == "$pr_head_sha" ]] || fail "--pr-head-sha did not resolve exactly"
  git merge-base --is-ancestor "$pr_head_sha" "$head_sha" || [[ "$pr_head_sha" == "$head_sha" ]] || fail "PR source SHA is not contained in the checked-out head"
fi
printf '%s\n' "$COMMIT_SHAS" > "$work/changed-commits.txt"
echo "Event range verified: $event $base_sha -> $head_sha"

echo "== Changed-file scope summary =="
CHANGED_FILES="$(cat "$work/changed-paths.txt")"
DOC_CHANGES=$(echo "$CHANGED_FILES" | grep -E '\.md$|docs/' | wc -l || echo "0")
CODE_CHANGES=$(echo "$CHANGED_FILES" | grep -E '\.(py|js|ts|go|rs|java)$' | wc -l || echo "0")
CONFIG_CHANGES=$(echo "$CHANGED_FILES" | grep -E '\.(json|yaml|yml|toml)$' | wc -l || echo "0")
WORKFLOW_CHANGES=$(echo "$CHANGED_FILES" | grep -E '\.github/workflows' | wc -l || echo "0")
echo "Documentation: $DOC_CHANGES; Code: $CODE_CHANGES; Config: $CONFIG_CHANGES; Workflows: $WORKFLOW_CHANGES"
TOTAL_FILES=$(echo "$CHANGED_FILES" | wc -l)
if [[ "$TOTAL_FILES" -gt 20 ]]; then
  echo "WARNING: Large change scope ($TOTAL_FILES files); consider splitting the change."
fi

# Secret Validation: this is the same pinned scanner and the same whole-checkout
# local scans as the workflow. Gitleaks requires Linux x86_64 and public network
# access to download and digest-verify the pinned binary.
echo "== Verified Gitleaks history scan =="
bash .github/scripts/gitleaks-verified.sh

echo "== .env file check =="
echo "Checking for accidentally committed .env files..."
ENV_FILES="$(find . -name '.env*' ! -name '.env.example' ! -name '.env.template' -type f 2>/dev/null || true)"
if [[ -n "$ENV_FILES" ]]; then
  printf 'ERROR: Found .env files that should not be committed:\n%s\n' "$ENV_FILES" >&2
  exit 1
fi
echo "No .env files found (good!)"

echo "== Credential file check =="
ALLOWLIST=".guardrails/secret-scan-allowlist.txt"
mapfile -t ALLOWED < <([ -f "$ALLOWLIST" ] && grep -vE '^[[:space:]]*(#|$)' "$ALLOWLIST")
PATTERNS=("*.pem" "*.key" "*credentials*.json" "*secrets*.json" "*service-account*.json")
FAILING=""
for pattern in "${PATTERNS[@]}"; do
  while IFS= read -r file; do
    [[ -n "$file" ]] || continue
    rel="${file#./}"
    allowed=0
    for item in "${ALLOWED[@]:-}"; do
      if [[ "$rel" == "$item" ]]; then
        allowed=1
        echo "Allowlisted synthetic fixture (not a failure): $rel"
        break
      fi
    done
    [[ "$allowed" == 1 ]] || FAILING+="$rel"$'\n'
  done < <(find . -name "$pattern" -type f 2>/dev/null || true)
done
if [[ -n "$FAILING" ]]; then
  printf 'ERROR: Found credential files that are not allowlisted synthetic fixtures:\n%s\n' "$FAILING" >&2
  exit 1
fi
echo "No non-allowlisted credential files found (good!)"

echo "== Hardcoded secret pattern check =="
ALLOW_ARGS=()
if [[ -f "$ALLOWLIST" ]]; then
  while IFS= read -r entry; do
    [[ -n "$entry" ]] || continue
    ALLOW_ARGS+=(--exclude="$(basename "$entry")")
  done < <(grep -vE '^[[:space:]]*(#|$)' "$ALLOWLIST")
fi
SECRET_PATTERNS=(
  "password\\s*=\\s*['\"]"
  "api_key\\s*=\\s*['\"]"
  "secret\\s*=\\s*['\"]"
  "token\\s*=\\s*['\"]"
  "aws_access_key_id\\s*=\\s*['\"]"
  "private_key\\s*=\\s*['\"]"
)
ISSUES_FOUND=0
for pattern in "${SECRET_PATTERNS[@]}"; do
  # Preserve the workflow's grep filters and allowlist behavior, but retain only
  # file names locally so matching secret values are not echoed to the terminal.
  MATCHED_LINES="$(grep -riE "$pattern" --include='*.py' --include='*.js' --include='*.ts' --include='*.yaml' --include='*.yml' --include='*.json' --exclude-dir=.devgate --exclude-dir=.git "${ALLOW_ARGS[@]}" . 2>/dev/null | grep -v 'node_modules' | grep -v '.example' | grep -v 'test_fixtures' || true)"
  MATCHES="$(printf '%s\n' "$MATCHED_LINES" | cut -d: -f1 | sed '/^$/d')"
  if [[ -n "$MATCHES" ]]; then
    echo "FAIL: Hardcoded secret pattern found in these files:"
    printf '%s\n' "$MATCHES"
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
  fi
done
if ((ISSUES_FOUND > 0)); then
  echo "ERROR: Found $ISSUES_FOUND hardcoded secret pattern(s)." >&2
  exit 1
fi
echo "No obvious hardcoded secrets found (good!)"

# Guardrails Lint: the event-scoped path checks receive the exact paths emitted
# by the existing helper. No absent-range fallback is accepted by this wrapper.
echo "== Changed-path and forbidden-file checks =="
python3 scripts/check_forbidden_files.py --paths-file "$work/changed-paths.txt"
python3 scripts/test_check_forbidden_files.py

echo "== Commit message format (workflow advisory) =="
INVALID_COMMITS=""
VALID_TYPES='feat|fix|docs|style|refactor|perf|test|chore|security|build|ci'
while IFS= read -r line; do
  [[ -n "$line" ]] || continue
  hash="${line%% *}"
  msg="${line#* }"
  if ! grep -qE "^($VALID_TYPES)(\\(.+\\))?: .+" <<< "$msg"; then
    if ! grep -qE '^Merge ' <<< "$msg"; then INVALID_COMMITS+="$hash: $msg"$'\n'; fi
  fi
done <<< "$COMMIT_SUMMARIES"
if [[ -n "$INVALID_COMMITS" ]]; then
  echo "WARNING: Commit messages outside the workflow's conventional format:"
  printf '%s\n' "$INVALID_COMMITS"
  echo "Workflow behavior is advisory only; this finding does not fail the gate."
else
  echo "All commit messages follow conventional format."
fi

echo "== AI attribution check =="
INSPECTED=0
MISSING=0
while IFS= read -r commit_sha; do
  [[ -n "$commit_sha" ]] || continue
  [[ "$commit_sha" =~ ^[0-9a-f]{40}$ ]] || fail "invalid commit record; expected full lowercase SHA-1"
  message="$(git show -s --format=%B "$commit_sha")" || fail "could not read complete message for $commit_sha"
  INSPECTED=$((INSPECTED + 1))
  if grep -qiE 'Claude|GPT|Gemini|AI|LLM|Copilot|agent' <<< "$message" && ! grep -qiF 'Authored by TheArchitectit' <<< "$message"; then
    echo "NOTE: Commit $commit_sha may need AI attribution."
    MISSING=$((MISSING + 1))
  fi
done < "$work/changed-commits.txt"
echo "AI attribution inspection: $INSPECTED commit(s) checked; $MISSING may need attribution."
if ((MISSING > 0)); then
  echo "If AI assisted with these commits, add: Authored by TheArchitectit"
else
  echo "Attribution check complete."
fi

echo "Local Secret Validation and Guardrails Lint checks passed for the supplied event range."
echo "GitHub-only artifacts, compliance summary, and event comment were not produced locally."
