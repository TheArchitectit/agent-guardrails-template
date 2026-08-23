# OpenSpec: Regulatory Compliance Mapping

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
