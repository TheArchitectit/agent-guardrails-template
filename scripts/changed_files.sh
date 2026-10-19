#!/usr/bin/env bash
# changed_files.sh — the changed-path set for the CURRENT event.
#
# Spec 19 R19-14: "A filter error, missing diff, shallow checkout, or
# unclassified path is NOT_RUN/FAIL, not an automatic skip."
#
# The call sites this replaces used one hardcoded line:
#
#     git diff --name-only origin/main...HEAD 2>/dev/null \
#       || git diff --name-only HEAD~1
#
# Two ways that is wrong, both measured on the real tree:
#
#   * `origin/main` is the wrong base for a workflow that also triggers on
#     `develop`, so a develop change was compared against the wrong branch.
#   * The `||` fallback is a NARROWING fallback. When the real diff fails (shallow
#     checkout, missing base ref, filter error) it silently scans the tip commit
#     only and the job still passes — the run measures one commit and reports on
#     the whole change.
#
# This script derives the base from the event and FAILS (non-zero) when it cannot
# compute an honest set. Refusing is the requirement; degrading quietly is the
# defect.
#
# Events:
#   pull_request  -> three-dot diff against the PR base branch
#   push          -> two-dot diff against the parent of the range's first commit
#                    (github.event.before when it is a real commit), so every
#                    commit in the pushed range is included
#   anything else -> requires BASE_SHA or falls back to a single parent; a new
#                    branch with no parent is refused rather than guessed
#
# Exit: 0 with paths on stdout (possibly empty); non-zero when the set cannot be
# computed — never a silent empty success.
set -euo pipefail

event="${EVENT_NAME:-${GITHUB_EVENT_NAME:-}}"
base_ref="${BASE_REF:-${GITHUB_BASE_REF:-}}"

die() {
  echo "changed_files: $*" >&2
  echo "changed_files: refusing to report a change set it could not compute (Spec 19 R19-14)" >&2
  exit 1
}

ensure_full_history() {
  # A shallow clone cannot produce a trustworthy diff. Deepen before deciding,
  # and refuse if the history is still truncated for the range we need.
  if [ "$(git rev-parse --is-shallow-repository 2>/dev/null || echo false)" = "true" ]; then
    echo "changed_files: shallow clone detected; deepening" >&2
    git fetch --unshallow --quiet origin >/dev/null 2>&1 || die "cannot unshallow the checkout"
  fi
  if [ "$(git rev-parse --is-shallow-repository 2>/dev/null || echo false)" = "true" ]; then
    die "history is still shallow; a partial diff would under-report the change set"
  fi
}

case "$event" in
  pull_request|pull_request_target)
    [ -n "$base_ref" ] || die "pull_request event without a base ref"
    git rev-parse --verify --quiet "origin/${base_ref}" >/dev/null \
      || git fetch --quiet origin "${base_ref}" >/dev/null 2>&1 \
      || die "base branch '${base_ref}' is not available"
    ensure_full_history
    # Three dots: the merge base, i.e. what this PR actually adds.
    git diff --name-only "origin/${base_ref}...HEAD" || die "diff against origin/${base_ref} failed"
    ;;

  push)
    before="${BEFORE_SHA:-${GITHUB_EVENT_BEFORE:-}}"
    ensure_full_history
    if [ -n "$before" ] && [ "$before" != "0000000000000000000000000000000000000000" ] \
       && git cat-file -e "${before}^{commit}" 2>/dev/null; then
      git diff --name-only "${before}" "${REF_SHA:-HEAD}" || die "diff ${before}..HEAD failed"
    else
      # First push of a branch (or a force push whose old tip is gone). Compare
      # against the merge base with the default branch when one exists; otherwise
      # refuse, because "everything" and "nothing" are both guesses.
      for candidate in origin/main origin/master; do
        if git rev-parse --verify --quiet "$candidate" >/dev/null; then
          base=$(git merge-base "$candidate" HEAD) || die "no merge base with $candidate"
          git diff --name-only "$base" HEAD || die "diff from merge base failed"
          exit 0
        fi
      done
      parents=$(git rev-list --parents -n 1 HEAD | wc -w)
      if [ "$parents" -ge 2 ]; then
        git diff --name-only HEAD^ HEAD || die "diff HEAD^..HEAD failed"
      else
        die "cannot determine a base for this push (no before-SHA, no default branch, no parent)"
      fi
    fi
    ;;

  workflow_dispatch|workflow_call|"")
    if [ -n "${BASE_SHA:-}" ]; then
      ensure_full_history
      git diff --name-only "${BASE_SHA}" HEAD || die "diff ${BASE_SHA}..HEAD failed"
    else
      ensure_full_history
      if git rev-parse --verify --quiet HEAD^ >/dev/null; then
        git diff --name-only HEAD^ HEAD || die "diff HEAD^..HEAD failed"
      else
        die "no base available for a manually dispatched run (set BASE_SHA)"
      fi
    fi
    ;;

  *)
    die "unsupported event '${event}'"
    ;;
esac
