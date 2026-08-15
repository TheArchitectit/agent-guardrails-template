# Operational Patterns

> Self-healing systems. Build resilience into your applications.

**Related:** [INFRASTRUCTURE_STANDARDS.md](infrastructure-standards.md) | [LOGGING_PATTERNS.md](logging-patterns.md)

Assume everything will fail. Plan for it. These patterns ensure your systems recover from failures automatically, provide meaningful health information, and degrade under stress.

## Topics

- [Health Checks](operational-health-checks.md) — The /health endpoint, liveness vs readiness probes
- [Circuit Breakers](operational-circuit-breakers.md) — Prevent cascading failures with fail-fast circuits
- [Retry and Graceful Degradation](operational-retry-and-degradation.md) — Exponential backoff, fallback strategies, and partial results
- [Rate Limiting](operational-rate-limiting.md) — Token bucket implementation and rate limit headers
- [Timeouts and Observability](operational-timeouts-and-observability.md) — Request timeouts, timeout hierarchies, and metrics

## Quick Reference

```
+------------------------------------------------------------------+
|              OPERATIONAL PATTERNS QUICK REFERENCE                 |
+------------------------------------------------------------------+
| HEALTH CHECKS:                                                    |
|   /health/live  - Is process alive? (simple)                     |
|   /health/ready - Is service ready for traffic? (full check)     |
+------------------------------------------------------------------+
| CIRCUIT BREAKER:                                                  |
|   CLOSED → Normal operation                                       |
|   OPEN → Fail fast (don't call failing service)                  |
|   HALF-OPEN → Test if service recovered                          |
+------------------------------------------------------------------+
| RETRY:                                                            |
|   Use for: Transient failures                                    |
|   Pattern: Exponential backoff with jitter                       |
|   Limit: Usually 3 attempts max                                  |
+------------------------------------------------------------------+
| GRACEFUL DEGRADATION:                                             |
|   1. Try primary service                                         |
|   2. If fail → Use fallback (cache, default, partial data)       |
|   3. Always serve something useful                               |
+------------------------------------------------------------------+
| TIMEOUTS:                                                         |
|   Database: 5s | External API: 10s | Handler: 30s | LB: 60s      |
|   Inner < Outer (always)                                         |
+------------------------------------------------------------------+
```
