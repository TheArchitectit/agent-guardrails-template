# Health Check Patterns

> Every service needs a health endpoint. Here is how to build one.

**See also:** [Operational Patterns overview](operational-patterns.md) | [INFRASTRUCTURE_STANDARDS.md](infrastructure-standards.md) | [LOGGING_PATTERNS.md](logging-patterns.md)

---

## The /health Endpoint

Every service MUST expose a health check endpoint.

```
HEALTH ENDPOINT REQUIREMENTS:

URL: GET /health or GET /healthz
Response Time: < 200ms
Response Format: JSON

HEALTHY RESPONSE (200 OK):
{
  "status": "healthy",
  "timestamp": "2026-01-21T15:30:00Z",
  "version": "1.2.3",
  "checks": {
    "database": "healthy",
    "cache": "healthy",
    "external_api": "healthy"
  }
}

UNHEALTHY RESPONSE (503 Service Unavailable):
{
  "status": "unhealthy",
  "timestamp": "2026-01-21T15:30:00Z",
  "version": "1.2.3",
  "checks": {
    "database": "unhealthy",
    "cache": "healthy",
    "external_api": "healthy"
  },
  "errors": [
    "Database connection timeout after 5000ms"
  ]
}
```

## Health Check Implementation

```typescript
// health.ts - Health check endpoint

interface HealthCheck {
  name: string;
  check: () => Promise<boolean>;
  timeout: number;
}

const healthChecks: HealthCheck[] = [
  {
    name: 'database',
    check: async () => {
      const result = await db.query('SELECT 1');
      return result !== null;
    },
    timeout: 5000,
  },
  {
    name: 'cache',
    check: async () => {
      await redis.ping();
      return true;
    },
    timeout: 1000,
  },
  {
    name: 'external_api',
    check: async () => {
      const response = await fetch('https://api.stripe.com/health');
      return response.ok;
    },
    timeout: 3000,
  },
];

export async function healthCheck(): Promise<HealthResponse> {
  const results: Record<string, string> = {};
  const errors: string[] = [];
  
  for (const check of healthChecks) {
    try {
      const result = await Promise.race([
        check.check(),
        new Promise((_, reject) => 
          setTimeout(() => reject(new Error('Timeout')), check.timeout)
        ),
      ]);
      results[check.name] = result ? 'healthy' : 'unhealthy';
    } catch (error) {
      results[check.name] = 'unhealthy';
      errors.push(`${check.name}: ${error.message}`);
    }
  }
  
  const isHealthy = Object.values(results).every(r => r === 'healthy');
  
  return {
    status: isHealthy ? 'healthy' : 'unhealthy',
    timestamp: new Date().toISOString(),
    version: process.env.APP_VERSION || 'unknown',
    checks: results,
    ...(errors.length > 0 && { errors }),
  };
}
```

## Liveness vs Readiness

```
TWO TYPES OF HEALTH CHECKS:

LIVENESS (/health/live):
- "Is the process alive?"
- Simple check, always fast
- If fails → Restart the container
- Example: Can the app respond at all?

READINESS (/health/ready):
- "Is the service ready to receive traffic?"
- Checks dependencies (DB, cache, etc.)
- If fails → Remove from load balancer (don't restart)
- Example: Is the database connection established?

Kubernetes uses both:
  livenessProbe:
    httpGet:
      path: /health/live
      port: 3000
    initialDelaySeconds: 10
    periodSeconds: 5
    
  readinessProbe:
    httpGet:
      path: /health/ready
      port: 3000
    initialDelaySeconds: 5
    periodSeconds: 10
```
