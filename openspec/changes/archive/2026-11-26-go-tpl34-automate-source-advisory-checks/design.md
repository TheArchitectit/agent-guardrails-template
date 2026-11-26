# Design

## Context

QA-05 records Go 1.26.11 and zero source-reachable findings at a dated database snapshot while retaining four module-only findings. The Makefile has a local `govulncheck` target, but no workflow invocation was found. Recurring scans must preserve reachability distinctions and fail closed on scan errors.

## Goals / Non-Goals

**Goals:**

- Detect future source-call-path advisories in the MCP Go module on PRs and main pushes.
- Preserve separate reporting for imported-only and module-only records.
- Avoid claiming a clean scan after tool, database, parser, or package-analysis failures.

**Non-Goals:**

- Reclassify QA-05's dated scan or imply the dependency graph has no advisories.
- Fail source reachability policy on module presence alone.

## Decisions

- Read the Go version from `mcp-server/go.mod`; pin the govulncheck tool version in the workflow and use a maintained vulnerability database.
- Consume structured `govulncheck -json` output. Source-call-path findings and analyzer/database/toolchain failures fail CI; imported-only and module-only findings are reported separately under the approved policy.
- Require the govulncheck JSON config to declare source mode and symbol scan level, plus a nonempty SBOM `roots` list. Progress messages are informational and do not count as package-analysis evidence.
- Classify each finding from its first trace frame (`function` means source-call-path; `package` without `function` means imported-package; neither means module-only). Upload only advisory IDs, package names, and reachability classes; omit function traces and source locations.
- Require every finding ID to resolve to an OSV record in the completed stream. In pinned v1.10.0 source mode, raw OSV records are emitted before findings; unreferenced OSV records remain valid because records can apply to a module without matching its selected version.
- Run the classifier fixtures directly in this workflow. The repository's general Python discovery scope intentionally excludes `mcp-server/` due existing scanner findings there, so the new tests must not rely on that inventory to execute them.
- Unit-test the result classifier with sanitized fixtures representing source-call-path, imported-only, module-only, malformed output, database error, and zero analyzed packages. Require evidence that at least one package was analyzed before an all-clear result.

## Risks / Trade-offs

- [Database outage appears as a clean result] → treat incomplete scans and parser/tool failures as job failures.
- [Module-only records are conflated with reachable paths] → retain separate typed categories and test them with sanitized output fixtures.
- [Pinned tool version ages] → make the pin explicit and review updates as dependency changes.

## Migration Plan

Add the classifier, its fixture tests, and the PR/main workflow. Run the documented command from `mcp-server/` with `GOTOOLCHAIN=local` before enabling the job as a required check.
