# MCP Server Observability Review

> Comprehensive review of observability and monitoring capabilities.

**Date:** 2026-02-08
**Scope:** MCP Server at `/mcp-server`

---

## Executive Summary

| Aspect | Status | Notes |
|--------|--------|-------|
| Logging | Good | Structured JSON logging with slog |
| Health Checks | Good | Live/ready probes implemented |
| Metrics | Partial | Prometheus endpoint exists, missing custom metrics |
| Request Tracing | Missing | No correlation ID propagation |
| Performance Metrics | Missing | No latency/error tracking |
| Alert Conditions | Missing | No defined SLOs or alert rules |

---

## 1. Logging Levels Review

### Current State
- Using `log/slog` with JSON handler
- Log level configurable via `LOG_LEVEL` env var
- Default level: `info`

### Assessment: GOOD
- Debug logs for auth success/failure
- Info logs for server startup/shutdown
- Error logs for failures

### Gaps
- Missing request/response logging
- Missing performance timing logs

---

## 2. Structured Logging Review

### Current State
- JSON structured logging enabled
- Audit logger with event types

### Assessment: GOOD
```go
slog.Info("Starting Guardrail MCP Server",
    "version", version,
    "build_time", buildTime,
)
```

### Gaps
- Missing standardized request context fields
- Missing operation timing fields

---

## 3. Metrics Review

### Current State
- Prometheus `/metrics` endpoint exposed
- Basic Go runtime metrics only

### Assessment: INCOMPLETE

### Missing Metrics

**RED Metrics (Per Endpoint):**
- `http_requests_total` (counter with status code labels)
- `http_request_duration_seconds` (histogram)
- `http_request_errors_total` (counter)

**Business Metrics:**
- `guardrail_validations_total` (by tool, result)
- `guardrail_sessions_active` (gauge)
- `guardrail_sessions_created_total` (counter)
- `guardrail_audit_events_total` (by type)

**Circuit Breaker Metrics:**
- `circuit_breaker_state` (gauge by name)
- `circuit_breaker_failures_total` (counter)

**Health Metrics:**
- `health_check_duration_seconds` (histogram)
- `health_check_failures_total` (counter by check)

---

## 4. Health Checks Review

### Current State
- `/health/live` - Liveness probe (simple)
- `/health/ready` - Readiness probe (checks DB, cache)
- CLI health check via `--health-check` flag

### Assessment: GOOD
```yaml
# Kubernetes probes configured
livenessProbe:
  exec:
    command: ["/server", "--health-check"]
readinessProbe:
  httpGet:
    path: /health/ready
    port: web
```

### Gaps
- Missing detailed health check response with component status
- No circuit breaker state in health response

---

## 5. Request Tracing Review

### Current State
- Echo provides `RequestID` middleware
- Audit logger captures request IDs

### Assessment: PARTIAL

### Gaps
- Correlation ID not propagated to logs consistently
- No request context in slog output
- No trace span support

---

## 6. Performance Metrics Review

### Current State
- Request timeout configured (30s)
- Rate limiting implemented

### Assessment: MISSING

### Missing Metrics
- Request latency percentiles (p50, p95, p99)
- Database query duration
- Cache hit/miss rates
- Session duration tracking

---

## 7. Alert Conditions

### Recommended Alerts

**Critical Alerts:**
- Error rate > 5% for 5 minutes
- P99 latency > 2s for 5 minutes
- Health check failing for 2 minutes
- Circuit breaker open

**Warning Alerts:**
- Error rate > 1% for 5 minutes
- P95 latency > 1s for 5 minutes
- Rate limiting triggered > 100/min
- Audit log buffer full

---

## 8. Dashboard Metrics

### Recommended Panels

**Overview Dashboard:**
- Request rate (req/s)
- Error rate (%)
- P50/P95/P99 latency
- Active sessions

**Health Dashboard:**
- Component health status
- Circuit breaker states
- Database connection pool
- Cache hit rate

**Business Dashboard:**
- Validations by tool type
- Audit events by severity
- Session creation rate
- Guardrail violations

---

## Implementation Plan

1. Add Prometheus metrics collectors
2. Add request logging middleware
3. Enhance health check responses
4. Add correlation ID propagation
5. Create alerting rules documentation

---

**Authored by:** TheArchitectit
**Last Updated:** 2026-02-08
