# Evidence and verification limits

## Source inspection

- `mcp-server/internal/guardrails/compliance_probe.go` and `engine_probes.go` define probe and assessment surfaces.
- `mcp-server/internal/guardrails/compliance_report.go` defines `CheckMeasured` and `GenerateMeasuredReport`.
- `mcp-server/internal/mcp/tools_compliance.go` supplies request-path integration and `complianceProbes()`.
- Existing tests include measured report, empty-probe refusal, and unknown requirement cases in `compliance_test.go`, `compliance_watermark_test.go`, and `tools_compliance_test.go`; source search found these symbols but this package did not run them.
- `docs/specs/guardrail-gaps-2026/06-regulatory-compliance-mapping.md` explicitly distinguishes the old simulated collection from measured reports and names unimplemented external evidence sources.

## Not run / not established

No live probe/report run was executed for TPL-47. No compliance framework was selected or activated against a real external system. The existence of measured code and tests is not represented as a live observation, independent audit, or compliance certification.
