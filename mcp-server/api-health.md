# Health Endpoints

> See [API.md](api.md) for base URLs, authentication, and shared conventions.

No authentication required.

## GET /health/live

Liveness probe - checks if the process is running.

**Response**
```json
{
  "status": "alive",
  "version": "1.0.0",
  "timestamp": "2026-02-07T10:00:00Z"
}
```

## GET /health/ready

Readiness probe - checks database and Redis connectivity.

**Response (200)**
```json
{
  "status": "ready",
  "version": "1.0.0",
  "timestamp": "2026-02-07T10:00:00Z"
}
```

**Response (503)**
```json
{
  "status": "not ready",
  "timestamp": "2026-02-07T10:00:00Z"
}
```

## GET /metrics

Prometheus metrics endpoint.

**Response**
```
# HELP guardrail_validations_total Total number of validations performed
# TYPE guardrail_validations_total counter
guardrail_validations_total{tool="bash",result="allowed"} 42
```

## GET /version

Server version information.

**Response**
```json
{
  "version": "1.0.0",
  "service": "guardrail-mcp",
  "timestamp": "2026-02-07T10:00:00Z"
}
```
