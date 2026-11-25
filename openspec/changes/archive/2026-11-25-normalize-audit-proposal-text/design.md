# Design

## Context

The local clean-checkout replay used DevGate commit
`8ccb4467c70614c37f4a533428d2aa1e09a80517`, the template's tracked pin.
It found seven disallowed characters in the TPL-45 gap inventory, TPL-46
proposal, and TPL-47 gap inventory.

## Goals / Non-Goals

Preserve the cited section numbers and ranges using `Sec.` and `to`. Keep the
existing project charset policy, baselines, and exception files.

## Decisions

Use the existing pinned scanner for acceptance. Compare each original
citation with its replacement. Run both the scanner on the candidate tree
and the scanner's existing controls.

## Risks / Trade-offs

A broad substitution can change unrelated text. Limit edits to the cited
locations in the three files and review the diff.
