# Design

## Context

The step counts conventional `fix`, `bugfix`, and `hotfix` subjects and writes
that numeric count to `GITHUB_OUTPUT`. It also writes all subjects using shell
command substitution, which embeds newlines in a record format that expects
one `name=value` line unless a delimiter protocol is used. No workflow step
consumes the resulting `commits` output.

## Goals / Non-Goals

**Goals:**

- Preserve the bug-fix count and all existing gating behavior.
- Ensure each emitted output is one valid line and only consumed values are
  emitted.
- Cover mixed ordinary/fix subjects in a deterministic local regression.

**Non-Goals:**

- Change the Git range, subject matching rule, or regression-test evaluation.
- Publish or interpolate commit subjects into workflow outputs.

## Decisions

- Move the small count operation into a shell helper that accepts a
  subject-list file. It writes only `bug_fix_count=<decimal>` to
  `GITHUB_OUTPUT` and prints only the decimal count for the existing workflow
  diagnostic.
- Keep Git log acquisition in the workflow and pass its existing subject list
  to the helper.
- Test ordinary and fix subjects together, verify the numeric count on stdout,
  and validate that every GitHub output line has the allowed key and decimal
  value.

## Risks / Trade-offs

- A helper could drift from the workflow call site; the regression checks both
  the helper's output and that the workflow invokes it.

## Migration Plan

No migration is required. Replace the inline count/output block with the
helper, then run the focused regression and workflow syntax/strict OpenSpec
validation.
