# go-advisory-automation Specification

Status: CLOSED (EXERCISED 1)

## Purpose

Defines recurring Go advisory scan behavior and how the workflow distinguishes reachable findings from dependency-only records.

## Requirements

### Requirement: Go source-reachable findings are checked in CI
The MCP Go module SHALL be scanned on pull requests and main pushes with govulncheck using the toolchain declared by `mcp-server/go.mod` and a pinned scanner version; source-call-path findings and incomplete or failed scans SHALL fail the job.

#### Scenario: source-reachable advisory appears
- **GIVEN** structured scan output contains a vulnerability with a source call path
- **WHEN** the CI advisory gate classifies the result
- **THEN** the job fails and reports the affected package and advisory identifier

#### Scenario: dependency-only advisory appears
- **GIVEN** output contains only imported-package or module-only findings without a source call path
- **WHEN** the CI advisory gate classifies the result
- **THEN** it reports those findings in a separate reachability category and does not label them source-reachable

#### Scenario: scanner cannot complete
- **GIVEN** the scanner, database, toolchain, output parser, or package analysis fails or analyzes no packages
- **WHEN** the workflow evaluates the scan result
- **THEN** the job fails rather than claiming a clean advisory scan
