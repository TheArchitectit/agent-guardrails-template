# Standards Documentation Index

> Navigation hub for documentation standards and patterns.

---

## Overview

This directory contains documentation standards that ensure consistency, maintainability, and efficiency across all project documentation.

---

## Quick Reference Table

| Document | Purpose | Key Rules |
|----------|---------|-----------|
| [test-production-separation.md](test-production-separation.md) | Test/production isolation | MANDATORY separation requirements |
| [modular-documentation.md](modular-documentation.md) | 500-line max rule | No doc over 500 lines |
| [logging-patterns.md](logging-patterns.md) | Array-based logging | Standard log format |
| [logging-integration.md](logging-integration.md) | External logging hooks | Hook interface spec |
| [api-specifications.md](api-specifications.md) | OpenAPI + OpenSpec | When to use each |
| [cross-cutting-2026.md](cross-cutting-2026.md) | 2026 universal standards | SBOM, SLSA, AI code gen, OWASP |

---

## Document Summaries

### test-production-separation.md
Establishes mandatory standards for separating test and production environments. All testing code, data, services, and infrastructure must be completely isolated from production.

**Key sections:**
- The Three Laws of Test/Production Separation
- Environment separation requirements (databases, services, users)
- Code creation sequence (production first, then test)
- Test code labeling requirements
- Uncertainty handling protocol
- Examples, patterns, and anti-patterns
- Blocking violations checklist

### modular-documentation.md
Defines the 500-line maximum rule for all documentation files and provides strategies for splitting large documents.

**Key sections:**
- The 500-Line Rule (why and how)
- Document structure standards
- Breaking up large documents
- Directory organization
- Compliance checklist

### logging-patterns.md
Establishes array-based structured logging patterns for agent operations.

**Key sections:**
- Array-based log entry structure
- Log levels (DEBUG, INFO, WARN, ERROR)
- Standard log categories
- Log array management
- Output formats

### logging-integration.md
Defines hooks and interfaces for integrating with external logging systems.

**Key sections:**
- Standard hook interface
- Webhook integration patterns
- File-based integration
- Queue-based integration
- Error handling

### api-specifications.md
Guidance on choosing between OpenAPI and OpenSpec for API documentation.

**Key sections:**
- OpenAPI overview and use cases
- OpenSpec overview and use cases
- When to use each format
- Hybrid approach guidance
- Template files

### cross-cutting-2026.md
Cross-cutting security and quality standards for ALL language profiles, reflecting 2026 best practices.

**Key sections:**
- Supply Chain Security (SBOM/SLSA, dependency verification)
- Secret Scanning (gitleaks, rotation guard)
- AI Code Generation Awareness (hallucinated deps, mandatory review)
- License Compliance (compatibility checks)
- Container Security (CVE scanning, multi-stage builds)
- OWASP Top 10 (2025/2026 checks)
- Mobile/Game (performance budgets, privacy, app store)

---

## Integration with Guardrails

These standards support the [agent-guardrails.md](../getting-started/agent-guardrails.md) requirements for:

- **Test/production separation** → test-production-separation.md
- **Audit requirements** → logging-patterns.md
- **External integration** → logging-integration.md
- **Documentation quality** → modular-documentation.md
- **API documentation** → api-specifications.md
- **2026 universal standards** → cross-cutting-2026.md

---

## Related Documents

- [agent-guardrails.md](../getting-started/agent-guardrails.md) - Mandatory safety protocols
- [Workflows Index](../workflows/INDEX.md) - Operational workflows
