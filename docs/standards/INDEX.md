# Standards Documentation Index

> Navigation hub for documentation standards and patterns.

---

## Overview

This directory contains documentation standards that ensure consistency, maintainability, and efficiency across all project documentation.

---

## Quick Reference Table

| Document | Purpose | Key Rules |
|----------|---------|-----------|
| [MODULAR_DOCUMENTATION.md](./MODULAR_DOCUMENTATION.md) | 500-line max rule | No doc over 500 lines |
| [LOGGING_PATTERNS.md](./LOGGING_PATTERNS.md) | Array-based logging | Standard log format |
| [LOGGING_INTEGRATION.md](./LOGGING_INTEGRATION.md) | External logging hooks | Hook interface spec |
| [API_SPECIFICATIONS.md](./API_SPECIFICATIONS.md) | OpenAPI + OpenSpec | When to use each |

---

## Document Summaries

### MODULAR_DOCUMENTATION.md
Defines the 500-line maximum rule for all documentation files and provides strategies for splitting large documents.

**Key sections:**
- The 500-Line Rule (why and how)
- Document structure standards
- Breaking up large documents
- Directory organization
- Compliance checklist

### LOGGING_PATTERNS.md
Establishes array-based structured logging patterns for agent operations.

**Key sections:**
- Array-based log entry structure
- Log levels (DEBUG, INFO, WARN, ERROR)
- Standard log categories
- Log array management
- Output formats

### LOGGING_INTEGRATION.md
Defines hooks and interfaces for integrating with external logging systems.

**Key sections:**
- Standard hook interface
- Webhook integration patterns
- File-based integration
- Queue-based integration
- Error handling

### API_SPECIFICATIONS.md
Guidance on choosing between OpenAPI and OpenSpec for API documentation.

**Key sections:**
- OpenAPI overview and use cases
- OpenSpec overview and use cases
- When to use each format
- Hybrid approach guidance
- Template files

---

## Integration with Guardrails

These standards support the [AGENT_GUARDRAILS.md](../AGENT_GUARDRAILS.md) requirements for:

- **Audit requirements** → LOGGING_PATTERNS.md
- **External integration** → LOGGING_INTEGRATION.md
- **Documentation quality** → MODULAR_DOCUMENTATION.md
- **API documentation** → API_SPECIFICATIONS.md

---

## Related Documents

- [AGENT_GUARDRAILS.md](../AGENT_GUARDRAILS.md) - Mandatory safety protocols
- [../workflows/INDEX.md](../workflows/INDEX.md) - Operational workflows
- [../sprints/INDEX.md](../sprints/INDEX.md) - Sprint task framework

---

**Last Updated:** 2026-01-14
**Document Count:** 4
