# Proposal: Verify compliance reporting against live probes

## Why

The compliance subsystem has measured-probe/report code, but the audit found no end-to-end live probe/report evidence proving that report rows originate from observed controls rather than declarations or simulated collection. The legacy compliance spec documents both simulated collection and currently missing service sources; a JSON/Markdown report by itself is not evidence that its inputs were live.

## What Changes

- Require an authorized isolated runtime probe run with exact probe sources, runtime identity, timestamps and observed results.
- Verify the report distinguishes observed, unknown/unmeasured and declared states, and refuses vacuous probe sets.
- Bind the report to the actual server/config revision and verify a report artifact can be read without exposing secrets.

## Scope mapping

- Primary source: `docs/specs/guardrail-gaps-2026/06-regulatory-compliance-mapping.md`, sections 3–5 and reconciled implementation status.
- Related code: `mcp-server/internal/guardrails/compliance_probe.go`, `engine_probes.go`, `compliance_report.go`, and `mcp-server/internal/mcp/tools_compliance.go`.
- Distinct from RUN-06 / `release-gate-conformance-reports`, which covers version-bound injection/semantic conformance reports and promotion gating. TPL-47 covers actual compliance control probes and measured-report provenance; it does not add classifier release thresholds.

## Capabilities

### New Capabilities
- `compliance-probe-report-evidence`: substantiate measured compliance reports with live probe observations and explicit unknowns.

## Impact

Compliance report/probe integration and a redacted runtime artifact. A report cannot be described as compliance certification or legal advice.
