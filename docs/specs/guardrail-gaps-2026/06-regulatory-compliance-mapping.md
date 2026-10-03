# OpenSpec: Regulatory Compliance Mapping

> **Do not trust scores from this code.** The mapper exists but `CheckRequirement` restates a hand-written database string rather than measuring the system, and evidence collection is simulated. See [STATUS.md](STATUS.md).

**Gap:** Nice-to-have — No explicit EU AI Act or NIST RMF compliance mapping
**Priority:** 🟢 Nice-to-have (Phase 3)
**Depends on:** All previous specs
**Blocks:** None

---

## 1. Problem Statement

The agent-guardrails-template has the **technical primitives** for compliance (logging, audit trails, access controls) but lacks **explicit mapping** to regulatory frameworks — making it difficult for organizations to demonstrate compliance with:

- **EU AI Act** (Regulation 2024/1689) — mandatory for high-risk AI systems in EU
- **NIST AI RMF 1.0** — voluntary US framework for AI risk management
- **ISO/IEC 42001** — AI management system standard
- **OWASP ASVS** — application security verification

Without explicit mapping, users must manually map guardrail features to compliance requirements — error-prone and time-consuming.

---

## 2. Proposed Solution

Add **Compliance Mapping** that explicitly maps guardrail features to regulatory requirements, provides compliance reports, and automates evidence collection.

### 2.1 Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Guardrail      │────▶│ Compliance       │────▶│  Compliance     │
│  Events/Logs    │     │ Mapper           │     │  Reports        │
│                 │     │ (requirement →   │     │  (evidence      │
│                 │     │  feature mapping)│     │   packages)     │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                               │
                        ┌──────▼──────┐
                        │  Requirement │
                        │  Database    │
                        │  (JSON)      │
                        └─────────────┘
```

### 2.2 Compliance Frameworks

#### EU AI Act Mapping

| Requirement | Guardrail Feature | Status | Evidence |
|-------------|-------------------|--------|----------|
| **Art. 9 - Risk Management** | Four Laws + Halt Conditions | ✅ Partial | Audit logs, halt condition records |
| **Art. 10 - Data Governance** | Content Filter (S7 Privacy) | ✅ Partial | Classification logs |
| **Art. 11 - Technical Documentation** | This spec + existing docs | ⚠️ Gap | Needs compliance docs |
| **Art. 12 - Logging for Traceability** | PostgreSQL audit trail | ✅ Full | Audit log queries |
| **Art. 13 - Transparency** | Agent behavior logging | ✅ Partial | Tool call logs |
| **Art. 14 - Human Oversight** | Halt Conditions + human review | ✅ Full | Halt condition logs |
| **Art. 15 - Accuracy/Robustness** | Injection Defense (Spec 01) | ✅ Partial | Injection detection logs |
| **Art. 50 - AI-Generated Content** | Content Filter (Spec 02) | ⚠️ Gap | No content watermarking yet |

#### NIST AI RMF Mapping

| Function | Guardrail Feature | Status | Evidence |
|----------|-------------------|--------|----------|
| **Govern** | Four Laws + policies | ✅ Full | Policy documents, configuration |
| **Map** | Risk assessment tools | ✅ Partial | Guardrail validation logs |
| **Measure** | Audit trail + metrics | ✅ Full | Prometheus metrics, audit queries |
| **Manage** | Halt conditions + sandbox | ✅ Full | Halt logs, sandbox violations |

### 2.3 Compliance Report Generation

```yaml
compliance_reports:
  eu_ai_act:
    enabled: true
    output_format: "json"  # "json" | "pdf" | "markdown"
    evidence_sources:
      - "audit_logs"       # PostgreSQL
      - "guardrail_events" # structured events
      - "config_history"   # configuration snapshots
    sections:
      - "risk_assessment"
      - "technical_documentation"
      - "logging_traceability"
      - "human_oversight"
      - "transparency"
    export_path: "reports/compliance/"
  nist_rmf:
    enabled: true
    output_format: "json"
    sections:
      - "govern"
      - "map"
      - "measure"
      - "manage"
    export_path: "reports/compliance/"
```

---

## 3. Technical Requirements

### 3.1 New MCP Tools

```go
// guardrail_generate_compliance_report — generates a compliance report for a framework
// Input:  { framework: enum("eu_ai_act","nist_rmf","iso_42001"), date_range?: TimeRange, format?: string }
// Output: { report_path: string, sections: []ComplianceSection, gaps: []Gap }
guardrail_generate_compliance_report(framework, date_range?, format?) → ComplianceReport

// guardrail_check_compliance — checks current system against a specific requirement
// Input:  { framework: string, requirement_id: string }
// Output: { compliant: bool, evidence: []Evidence, gaps: []Gap, recommendations: []string }
guardrail_check_compliance(framework, requirement_id) → ComplianceCheck

// guardrail_collect_evidence — collects evidence for a specific requirement
// Input:  { framework: string, requirement_id: string, date_range?: TimeRange }
// Output: { evidence: []Evidence, completeness: float }
guardrail_collect_evidence(framework, requirement_id, date_range?) → EvidenceCollection
```

### 3.2 Requirement Database

```json
{
  "eu_ai_act": {
    "art_12": {
      "title": "Logging for traceability",
      "description": "High-risk AI systems must be designed to enable automatic recording of events (logs) over the lifetime of the system.",
      "required_for": ["high-risk"],
      "guardrail_features": ["audit_trail", "tool_call_logging", "guardrail_decisions"],
      "evidence_queries": [
        "SELECT * FROM audit_logs WHERE event_type = 'guardrail_decision' AND created_at > ?",
        "SELECT * FROM guardrail_events WHERE event = 'injection_detected'"
      ],
      "compliance_status": "full",
      "gaps": [],
      "recommendations": []
    }
  }
}
```

### 3.3 Evidence Collection

Evidence is automatically collected from:
- **PostgreSQL audit logs** — all guardrail decisions
- **Structured events** — injection detections, content classifications
- **Configuration snapshots** — guardrail policies at point in time
- **Metrics** — Prometheus counters for guardrail decisions

### 3.4 Compliance Dashboard (Optional)

```yaml
compliance_dashboard:
  enabled: false  # opt-in
  port: 8080
  metrics:
    - "compliance_score_by_framework"
    - "guardrail_decisions_by_type"
    - "gap_resolution_progress"
    - "evidence_completeness"
```

---

## 4. Implementation Notes

### 4.1 Compliance Score Calculation

```go
func calculateComplianceScore(framework string, evidence []Evidence) float64 {
    total := len(requirements[framework])
    covered := 0
    for _, req := range requirements[framework] {
        if isCovered(req, evidence) {
            covered++
        }
    }
    return float64(covered) / float64(total) * 100
}
```

### 4.2 Report Format

Compliance reports include:
1. **Executive Summary** — overall compliance score, critical gaps
2. **Requirement-by-Requirement** — status, evidence, gaps, recommendations
3. **Evidence Package** — queryable audit logs and structured events
4. **Gap Analysis** — what's missing and how to address it
5. **Action Items** — prioritized list of compliance improvements

### 4.3 Gap Resolution Tracking

```yaml
compliance_gaps:
  - id: "gap_eu_001"
    framework: "eu_ai_act"
    requirement: "art_50"
    description: "No AI-generated content watermarking"
    severity: "medium"
    status: "open"
    assigned_to: ""
    due_date: ""
    resolution_plan: "Add content fingerprinting to Spec 02"
```

---

## 5. Testing Criteria

### 5.1 Unit Tests
- [ ] Compliance score calculation is correct
- [ ] Requirement mapping covers all EU AI Act articles
- [ ] Evidence collection queries return correct results
- [ ] Report generation produces valid JSON/Markdown

### 5.2 Integration Tests
- [ ] Full pipeline: guardrail events → compliance mapper → report
- [ ] Gap detection identifies missing features correctly
- [ ] Evidence collection gathers from all configured sources
- [ ] Report export works for all formats

### 5.3 Validation Tests
- [ ] Compliance scores match manual audit
- [ ] Evidence packages satisfy auditor requirements
- [ ] Gap recommendations are actionable and specific

---

## 6. Dependencies

- **Internal:** All previous specs (01-05), PostgreSQL audit trail
- **External:** None
- **Data:** EU AI Act text, NIST AI RMF 1.0, ISO 42001

---

## 7. References

- [EU AI Act](https://digital-strategy.ec.europa.eu/en/policies/regulatory-framework-ai) — Regulation 2024/1689
- [NIST AI RMF 1.0](https://www.nist.gov/itl/ai-risk-management-framework) — AI risk management framework
- [ISO/IEC 42001](https://www.iso.org/standard/81230.html) — AI management system standard
- [OWASP ASVS](https://owasp.org/www-project-application-security-verification-standard/) — application security verification
- [NIST AI RMF Generative AI Profile](https://www.nist.gov/itl/ai-rmf) — generative AI-specific guidance

---

## 8. Implementation Status (reconciled 2026-10-03)

Evidence: `docs/specs/guardrail-gaps-2026/STATUS.md`.

| § | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 3.1 | `guardrail_generate_compliance_report` | **Not implemented** | name absent from every `.go` file |
| 3.1 | `guardrail_check_compliance` | **Not implemented** | same |
| 3.1 | `guardrail_collect_evidence` | **Not implemented** | same |
| 3.2 | Requirement DB | Implemented as **data** | `internal/guardrails/compliance_requirements.json` covers `eu_ai_act` art. 9, 10, 12, 13, 14, 15, 50; `nist_rmf` govern/map/measure/manage; `iso_42001` |
| 3.2 | **Art. 11** | **Missing** | absent from the JSON although this spec lists it as a gap — so §5.1's own acceptance criterion fails today |
| 4.1 | `ComplianceMapper` / `CalculateComplianceScore` | Built, **not wired** | `compliance.go:64,228`; constructed only in `compliance_test.go` |
| 4.2 | Report generation | Built, not wired | `ComplianceReporter` / `GenerateReport` / `ExportReport` (`compliance.go:105,113,163`) |
| 4.x | Evidence collection | **Simulated** | `CollectEvidence` hardcodes `Value: "Query result for [%s]: 12 events found"` and returns `completeness = 1.0` for any non-empty query (`compliance.go:213`) |
| 3.4 | `compliance_gaps` tracking, `compliance_dashboard` | **Not implemented** | no code |
| — | Config | Never constructed | `ComplianceConfig` / `DefaultComplianceConfig` (`compliance_config.go:7,17`) have no production call site |

**Do not quote a score from this code.** `CheckRequirement` returns `true`
purely because the hand-written `compliance_status` string in the JSON equals
`"full"` (`compliance.go:90-93`), with the in-code comment *"In a real
implementation, this would check if the features are actually enabled."* A
score produced this way restates the database; it does not measure the system.
Presenting it as compliance evidence to an auditor would be inaccurate — this
is the single most consequential gap in the six specs.

**Deviations:** `GenerateReport` emits one undifferentiated
`"General Compliance"` section (`compliance.go:153-157`) rather than the five
sections of §4.2; formats are json/markdown (`compliance_config.go:51-52`)
while the spec asked for json/pdf/markdown.

**Blocking decisions:**

1. **What does a requirement actually check?** §5.x criteria ("scores match
   manual audit", "evidence satisfies auditor requirements") have no oracle,
   threshold or pass condition, so they cannot pass or fail as written. Each
   needs a defined measurement before any score is produced.
2. Add art. 11 to the DB or record explicitly why it is excluded.
3. Real evidence collection needs named sources (PostgreSQL tables, Prometheus
   queries) — the current simulator returns a fixed string.
