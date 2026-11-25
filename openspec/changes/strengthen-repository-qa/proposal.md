# Proposal: Strengthen repository QA coverage

## Why

The pinned review found repository QA and packaging consistency issues that
are not represented by the successful hosted workflows alone. Strict OpenSpec
validation covers only the eight indexed specs and rejects seven unindexed
specs on scenario structure. The GitLab configuration installs from a root
`requirements.txt` that is absent in this checkout. GitLab and Jenkins declare
Go 1.21 while `mcp-server/go.mod` declares Go 1.25.5; the version mismatch may
depend on automatic toolchain download and is a reproducibility risk, not
proof of a guaranteed failure. The team CLI installer installs a Go binary,
while its Python backend defaults to a checkout-relative script path, so a
standalone install requires `TEAM_MANAGER_PATH` or a compatible layout.

These findings are `QA-01`, `CI-01`, `CI-02`, and `DIST-01` in
`openspec/audits/2026-11-25-qa-and-gap-review.md`. The historical archive
retirement reference is also absent locally; tracked under `INV-01` as a
traceability issue. IDE request wiring, installer paths, and Pi package
metadata have additional reproducible gaps in the audit.
Several numbered design specifications also retain historical status claims
that need owner-accepted reconciliation against current code and named tests
(`LEGACY-01`).
The dated govulncheck scan also reports source-reachable findings across four
modules and the Go standard library. Its output does not provide standardized
severity data (`QA-05`).

## Change

Require QA validation to account for the complete OpenSpec inventory and
validate proposed packages strictly. Align CI dependency and Go toolchain
inputs with repository reality, and make the team CLI's backend path explicit
for installed use.

## Impact

This is a proposal for QA, CI, and distribution evidence. It does not modify
those workflows or claim that the identified jobs failed at the pinned
revision.
