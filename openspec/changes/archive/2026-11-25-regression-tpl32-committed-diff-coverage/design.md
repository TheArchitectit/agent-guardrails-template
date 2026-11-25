# Design

## Context

`get_changed_files()` reads only `git diff --cached` and `git diff`, while the workflow invokes `--all` on a clean checkout. PR and push events provide distinct base revisions, and the workflow fetches full history. Missing or empty CI ranges must not be interpreted as a clean scan.

## Goals / Non-Goals

**Goals:**

- Evaluate committed PR and push patches in CI with event-specific ancestry semantics.
- Preserve existing staged and unstaged local modes.
- Fail closed when the event range cannot be checked.

**Non-Goals:**

- Change registry matching or prevention-rule semantics.
- Replace the local developer workflow with CI-only arguments.

## Decisions

- Add explicit base/head mode for CI. The workflow passes the pull request's exact `base.sha` and `head.sha`, and opts into PR merge-base mode. Pushes pass event `before` and `github.sha` without that mode.
- Resolve revision inputs to canonical commit IDs with option termination. Reject shallow history. PR mode requires exactly one merge base and scans that commit to the event head, excluding changes that exist only on a base branch advanced after the PR fork. Push mode requires `before` to be an ancestor of head. Both modes reject missing commits and empty file ranges; neither substitutes `HEAD^`.
- For local `--all`, evaluate staged and unstaged patches independently before combining findings. A forbidden-context match in one patch must not suppress a matching pattern in the other patch.
- Use literal pathspecs when reading per-file committed patches so a filename cannot be interpreted as Git pathspec syntax.
- In temporary Git repositories, test a benign sentinel in a committed PR patch, a clean nonempty range, divergent PR history with base-only sentinel content, missing and shallow histories, push divergence, and an empty range. Cover staged-only, unstaged-only, and mixed local patches where the unstaged forbidden context cannot mask the staged match.

## Risks / Trade-offs

- [A force-push/new branch has unavailable or ambiguous ancestry] → fail with a clear range error, not a green empty scan.
- [CI change breaks local invocations] → retain existing local argument defaults and regression tests.

## Migration Plan

Update the script and workflow together. Run the committed-range fixtures and verify the workflow event wiring before enabling the new CI command.
