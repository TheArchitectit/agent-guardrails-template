# Proposal: Record the merged QA-05 source-reachable remediation

## Why

The dated QA-05 baseline reported source-reachable Go advisories, including standard-library findings. Commit `943bcf6` (#12) is merged and includes the tested remediation; record that bounded result as a standalone historical change with evidence that can be reviewed independently.

## What Changes

- Record only the source-reachable remediation merged by #12: Go 1.26.11, aligned workflow toolchain selection and digest-pinned Docker builder, plus dependency updates in the MCP module.
- Preserve the same-toolchain build and advisory-scan results and their limitations in a dedicated evidence artifact.
- Keep the four residual findings explicit as module-only advisories and avoid claiming a CI advisory gate or resolving unrelated QA-05 scope.

## Capabilities

### New Capabilities

- `go-advisory-evidence`: The Go advisory evidence record captures the toolchain, source-reachability comparison, residual module-only findings, and bounded build evidence for a reviewed release.

### Modified Capabilities

None. This change adds no behavior contract to an existing capability.

## Impact

Applies to the MCP Go module, its Docker builder and Go setup steps in the team validation workflows. The source change is already merged at `943bcf6`; this change adds the bounded evidence contract and records its historical acceptance evidence. Continuous Go advisory automation remains separate under TPL-34. The evidence is scoped to Linux/amd64 and the stated vulnerability database snapshot.
