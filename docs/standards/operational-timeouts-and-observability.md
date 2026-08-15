# Timeouts and Observability

> Control how long operations wait, and track what matters

**See also:** [Operational Patterns overview](operational-patterns.md) | [LOGGING_PATTERNS.md](logging-patterns.md)

---

## Request Timeouts

```typescript
// timeout.ts - Timeout wrapper

async function withTimeout<T>(
  promise: Promise<T>,
  timeoutMs: number,
  errorMessage: string = 'Operation timed out'
): Promise<T> {
  const timeoutPromise = new Promise<never>((_, reject) => {
    setTimeout(() => reject(new Error(errorMessage)), timeoutMs);
  });
  
  return Promise.race([promise, timeoutPromise]);
}

// Usage
const data = await withTimeout(
  fetchExternalAPI(),
  5000,
  'External API did not respond in 5 seconds'
);
```

## Timeout Hierarchy

```
TIMEOUT HIERARCHY (inside → outside):

Database Query:    5 seconds
External API:     10 seconds
Request Handler:  30 seconds
Load Balancer:    60 seconds

RULE: Inner timeouts must be shorter than outer timeouts.
If database times out at 5s, handler should catch and respond
before the 30s handler timeout.
```

---

## Metrics to Track

```
MANDATORY METRICS:

1. RED Metrics (per endpoint):
   - Rate: Requests per second
   - Errors: Error rate percentage
   - Duration: Response time (p50, p95, p99)

2. USE Metrics (per resource):
   - Utilization: % of resource used
   - Saturation: Queue depth / backlog
   - Errors: Error count

3. Circuit Breaker Metrics:
   - State (closed/open/half-open)
   - Failure count
   - Success count after half-open

4. Health Check Metrics:
   - Check duration
   - Check result (pass/fail)
   - Last check timestamp
```

## Structured Error Logging

```typescript
// error-logger.ts

function logError(error: Error, context: Record<string, any>) {
  const logEntry = {
    timestamp: new Date().toISOString(),
    level: 'ERROR',
    message: error.message,
    stack: error.stack,
    context: {
      ...context,
      requestId: context.requestId,
      userId: context.userId,
      endpoint: context.endpoint,
    },
    // DO NOT log sensitive data
    // PII must be redacted
  };
  
  console.error(JSON.stringify(logEntry));
}
```
