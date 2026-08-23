# OpenSpec: Semantic Content Filtering

**Gap:** Critical — No native semantic content moderation
**Priority:** 🔴 Critical (Phase 1)
**Depends on:** None
**Blocks:** 04-multi-agent-safety-policies

---

## 1. Problem Statement

The agent-guardrails-template validates **structural patterns** (command syntax, file paths, git operations) but has no mechanism to evaluate the **semantic content** of text — what the agent reads, generates, or transmits. This means:

- The agent could generate or relay harmful content without detection
- No classification of dangerous/hateful/illegal content in tool outputs
- No content policy enforcement beyond command-level patterns
- No equivalent to NeMo's input/output moderation or Llama Guard's S1-S13 taxonomy

---

## 2. Proposed Solution

Add a **Semantic Content Filter** that classifies text against configurable safety policies before it reaches the agent or leaves the agent's outputs.

### 2.1 Architecture

```
┌──────────────┐     ┌──────────────────┐     ┌──────────────┐
│  Text In/Out  │────▶│ Content Filter    │────▶│ Agent / User │
│              │     │ (multi-classifier)│     │              │
└──────────────┘     └──────────────────┘     └──────────────┘
                           │
                    ┌──────▼──────┐
                    │  Policy     │
                    │  Engine     │
                    │  (YAML)     │
                    └─────────────┘
```

### 2.2 Safety Taxonomy (extends Llama Guard S1-S13)

| ID | Category | Action | Example Violation |
|----|----------|--------|-------------------|
| S1 | Violent Crimes | Block | Instructions for violence |
| S2 | Non-Violent Crimes | Block | Fraud, theft guidance |
| S3 | Sex Crimes | Block | Sexual exploitation |
| S4 | Child Exploitation | Block | Any CSE material |
| S5 | Defamation | Warn | False statements about individuals |
| S6 | Specialized Advice | Warn | Unqualified medical/legal advice |
| S7 | Privacy | Block | PII exposure, doxxing |
| S8 | Intellectual Property | Warn | Copyright infringement guidance |
| S9 | Indiscriminate Weapons | Block | Weapons of mass destruction |
| S10 | Hate | Block | Hate speech, discrimination |
| S11 | Self-Harm | Block | Self-harm encouragement |
| S12 | Sexual Content | Block/Config | Explicit sexual content |
| S13 | Elections | Warn | Electoral misinformation |
| S14 | Code Abuse | Block | Malicious code generation |
| S15 | Data Exfiltration | Block | Attempts to extract training data |

**Coding-specific additions (S14-S15)** are tailored to the agent-guardrails use case.

---

## 3. Technical Requirements

### 3.1 New MCP Tools

```go
// guardrail_classify_content — classifies text against safety taxonomy
// Input:  { text: string, direction: enum("input","output"), context?: string }
// Output: { safe: bool, categories: []CategoryResult, overall_risk: float }
guardrail_classify_content(text, direction, context?) → ClassificationResult

// guardrail_check_policy — checks if text violates a specific policy
// Input:  { text: string, policy_id: string }
// Output: { compliant: bool, violations: []Violation }
guardrail_check_policy(text, policy_id) → PolicyResult
```

### 3.2 Configuration

```yaml
# guardrails.yaml — content filtering section
content_filter:
  enabled: true
  backend: "llama-guard"  # "llama-guard" | "nemo" | "openai-moderation" | "custom"
  backend_config:
    llama_guard:
      model: "llama-guard-3"
      ollama_url: "http://localhost:11434"
    nemo:
      config_path: "config/nemo/actions.yml"
    openai_moderation:
      api_key_env: "OPENAI_API_KEY"
  taxonomy:
    enabled_categories: ["S1","S2","S3","S4","S7","S9","S10","S11","S14","S15"]
    custom_categories: []  # user-defined categories
  thresholds:
    default: 0.7
    per_category: {}  # overrides per category
  policies:
    - id: "coding-safety"
      description: "Coding-specific content policy"
      rules:
        - category: "S14"
          action: "block"
          description: "Block malicious code generation"
        - category: "S15"
          action: "block"
          description: "Block data exfiltration attempts"
  fail_policy: "block"  # classifier unavailable → block
```

### 3.3 Policy Engine

Policies are YAML-defined rules that map categories to actions:

```yaml
policies:
  - id: "enterprise-safe"
    description: "Enterprise-safe content policy"
    rules:
      - category: "S10"
        action: "block"
        threshold: 0.5  # lower threshold = stricter
        reason: "Enterprise hate speech policy"
      - category: "S6"
        action: "warn"
        reason: "Unqualified advice warning"
    overrides:
      - category: "S12"
        action: "allow"
        description: "Allow medical content in healthcare contexts"
        context_pattern: "medical/.*"
```

### 3.4 Classification Result Format

```json
{
  "safe": false,
  "overall_risk": 0.87,
  "categories": [
    {
      "id": "S10",
      "name": "Hate",
      "score": 0.92,
      "action": "block",
      "reason": "Content contains discriminatory language targeting protected groups"
    },
    {
      "id": "S7",
      "name": "Privacy",
      "score": 0.65,
      "action": "warn",
      "reason": "Potential PII detected (name + address pattern)"
    }
  ],
  "backend": "llama-guard-3",
  "latency_ms": 42
}
```

---

## 4. Implementation Notes

### 4.1 Classifier Backends

| Backend | Pros | Cons | Latency |
|---------|------|------|---------|
| Llama Guard 3 | Free, local, 13 categories | Requires GPU/quantized model | ~40ms |
| NeMo Guardrails | Rich policy language, Colang | Python dependency, heavier | ~100ms |
| OpenAI Moderation | Zero setup, reliable | API cost, data leaves server | ~200ms |
| Custom Model | Tailored to use case | Training required | Varies |

### 4.2 Go MCP Server Changes

- Add `content_filter.go` package under `internal/guardrails/`
- Classifier backends behind `ContentClassifier` interface
- Policy engine evaluates classification results against configured policies
- Results cached for 60s (same text = same classification)

### 4.3 Streaming Support

For real-time agent output filtering:
- Agent generates text in chunks
- Each chunk is classified as it arrives
- If a chunk violates policy, generation is stopped immediately
- Requires streaming integration with the agent framework

---

## 5. Testing Criteria

### 5.1 Unit Tests
- [ ] Llama Guard correctly classifies S1-S15 categories
- [ ] Policy engine correctly applies threshold overrides
- [ ] Cache hit/miss behavior is correct
- [ ] Backend failover works (Llama Guard → OpenAI Moderation)

### 5.2 Integration Tests
- [ ] Full pipeline: text → classifier → policy → decision
- [ ] Streaming mode stops generation on policy violation
- [ ] Audit log entries are complete and structured
- [ ] Configuration hot-reload for policies

### 5.3 Accuracy Tests
- [ ] False positive rate <5% on legitimate coding content
- [ ] False negative rate <1% on known harmful content
- [ ] Per-category precision/recall within 10% of Llama Guard benchmarks
- [ ] No regression on 500+ test cases

---

## 6. Dependencies

- **Internal:** Existing MCP server, PostgreSQL audit trail
- **External:** Llama Guard model, Ollama runtime (optional)
- **Related Specs:** 04-multi-agent-safety-policies builds on this

---

## 7. References

- [Llama Guard 3 Taxonomy](https://developer.meta.com/ai/docs/model-cards-and-prompt-formats/llama-guard-3/) — S1-S13 categories
- [NVIDIA NeMo Guardrails](https://github.com/NVIDIA/NeMo-Guardrails) — input/output moderation
- [OpenAI Moderation API](https://platform.openai.com/docs/guides/moderation) — content classification
- [OWASP LLM Top 10](https://owasp.org/www-project-top-10-for-large-language-model-applications/) — content risks
