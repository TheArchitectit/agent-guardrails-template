#!/usr/bin/env bash
# changed_commits.sh — the commit set for the CURRENT event.
#
# Companion to scripts/changed_files.sh, and it exists for the same Spec 19
# R19-14 reason: the call sites it replaces used
#
#     git log origin/main..HEAD ... || git log -5 ...
#
# which has two defects. `origin/main` is the wrong range for a workflow that
# also triggers on `develop`; and the fallback silently inspects the last five
# commits — an arbitrary window that can both miss commits in the change and
# include commits outside it, while the job still reports green.
#
# Usage: changed_commits.sh [--format FORMAT]
#   --format defaults to "%h %s".
#
# Exit: 0 with commits on stdout; non-zero when the range cannot be computed.
set -euo pipefail

format="%h %s"
while [ $# -gt 0 ]; do
  case "$1" in
    --format) format="${2:?--format needs a value}"; shift 2 ;;
    *) echo "changed_commits: unknown argument '$1'" >&2; exit 2 ;;
  esac
done

event="${EVENT_NAME:-${GITHUB_EVENT_NAME:-}}"
base_ref="${BASE_REF:-${GITHUB_BASE_REF:-}}"

die() {
  echo "changed_commits: $*" >&2
  echo "changed_commits: refusing to report a commit set it could not compute (Spec 19 R19-14)" >&2
  exit 1
}

case "$event" in
  pull_request|pull_request_target)
    [ -n "$base_ref" ] || die "pull_request event without a base ref"
    git rev-parse --verify --quiet "origin/${base_ref}" >/dev/null \
      || git fetch --quiet origin "${base_ref}" >/dev/null 2>&1 \
      || die "base branch '${base_ref}' is not available"
    git log --no-merges --pretty=format:"${format}" "origin/${base_ref}..HEAD" \
      || die "log against origin/${base_ref} failed"
    ;;

  push)
    before="${BEFORE_SHA:-${GITHUB_EVENT_BEFORE:-}}"
    if [ -n "$before" ] && [ "$before" != "0000000000000000000000000000000000000000" ] \
       && git cat-file -e "${before}^{commit}" 2>/dev/null; then
      git log --no-merges --pretty=format:"${format}" "${before}..HEAD" \
        || die "log ${before}..HEAD failed"
    else
      for candidate in origin/main origin/master; do
        if git rev-parse --verify --quiet "$candidate" >/dev/null; then
          base=$(git merge-base "$candidate" HEAD) || die "no merge base with $candidate"
          git log --no-merges --pretty=format:"${format}" "${base}..HEAD" \
            || die "log from merge base failed"
          exit 0
        fi
      done
      git log --no-merges -n 1 --pretty=format:"${format}" HEAD \
        || die "cannot determine a commit range for this push"
    fi
    ;;

  workflow_dispatch|workflow_call|"")
    if [ -n "${BASE_SHA:-}" ]; then
      git log --no-merges --pretty=format:"${format}" "${BASE_SHA}..HEAD" \
        || die "log ${BASE_SHA}..HEAD failed"
    else
      git log --no-merges -n 1 --pretty=format:"${format}" HEAD \
        || die "cannot determine a commit range for a manual run (set BASE_SHA)"
    fi
    ;;

  *)
    die "unsupported event '${event}'"
    ;;
esac
