# Design

## Context

`get_changed_files()` reads only `git diff --cached` and `git diff`, while the workflow invokes `--all` on a clean checkout. PR and push events provide distinct base revisions, and the workflow fetches full history. Missing or empty CI ranges must not be interpreted as a clean scan.

## Goals / Non-Goals

**Goals:**

- Evaluate committed PR and push patches in CI.
- Preserve existing staged and unstaged local modes.
- Fail closed when the event range cannot be checked.

**Non-Goals:**

- Change registry matching or prevention-rule semantics.
- Replace the local developer workflow with CI-only arguments.

## Decisions

- Add explicit base/head mode for CI. Pull requests use the base SHA and checked-out head SHA; main pushes use event `before` and `github.sha`.
- Require both commits to resolve and the comparison range to be nonempty. Do not silently substitute `HEAD^` when event data or full history is missing.
- When both staged and unstaged local modes are selected, inspect each corresponding patch rather than using the staged diff for every changed file.
- In a temporary Git repository, commit a benign sentinel matching a temporary registry pattern after the base; assert the range reports it. Include clean-range and unresolved-base controls, then cover staged-only, unstaged-only, and combined local matching so the content check uses the same scope as path discovery.

## Risks / Trade-offs

- [A force-push/new branch has an unavailable base] → fail with a clear range error, not a green empty scan.
- [CI change breaks local invocations] → retain existing local argument defaults and regression tests.

## Migration Plan

Update the script and workflow together. Run the committed-range fixtures and verify the workflow event wiring before enabling the new CI command.
