# Circuit Breaker Pattern

> Prevent cascading failures when external services go down

**See also:** [Operational Patterns overview](operational-patterns.md) | [Retry and Graceful Degradation](operational-retry-and-degradation.md)

---

## Why Circuit Breakers?

```
PROBLEM: Cascading Failures

When an external service (Stripe, OpenAI, etc.) goes down:
1. Your app keeps trying to call it
2. Each call times out (30 seconds)
3. Requests pile up
4. Your app runs out of resources
5. Your app crashes
6. Users blame you, not Stripe

SOLUTION: Circuit Breaker

When external service fails:
1. First few failures → Try normally
2. Too many failures → "Trip" the circuit
3. Circuit is OPEN → Fail immediately (don't wait)
4. After cooldown → Try again
5. If success → "Close" the circuit
6. Back to normal
```

## Circuit Breaker States

```
┌─────────────────────────────────────────────────────────────┐
│                   CIRCUIT BREAKER STATES                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  CLOSED (Normal Operation)                                   │
│    │                                                         │
│    │ Failure threshold exceeded                              │
│    ▼                                                         │
│  OPEN (Fail Fast)                                            │
│    │ - All calls immediately fail                           │
│    │ - No actual API calls made                             │
│    │ - Return fallback or error                             │
│    │                                                         │
│    │ Cooldown period elapsed                                │
│    ▼                                                         │
│  HALF-OPEN (Testing)                                         │
│    │ - Allow ONE request through                            │
│    │                                                         │
│    ├── Success → Back to CLOSED                             │
│    └── Failure → Back to OPEN                               │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Circuit Breaker Implementation

```typescript
// circuit-breaker.ts

interface CircuitBreakerConfig {
  failureThreshold: number;    // Failures before opening
  successThreshold: number;    // Successes to close from half-open
  timeout: number;             // Cooldown in ms
}

class CircuitBreaker {
  private state: 'CLOSED' | 'OPEN' | 'HALF_OPEN' = 'CLOSED';
  private failures = 0;
  private successes = 0;
  private lastFailureTime: number | null = null;
  
  constructor(private config: CircuitBreakerConfig) {}
  
  async execute<T>(fn: () => Promise<T>, fallback?: T): Promise<T> {
    // Check if circuit should transition from OPEN to HALF_OPEN
    if (this.state === 'OPEN') {
      const timeSinceFailure = Date.now() - (this.lastFailureTime || 0);
      if (timeSinceFailure >= this.config.timeout) {
        this.state = 'HALF_OPEN';
        this.successes = 0;
      } else {
        // Circuit is OPEN - fail fast
        if (fallback !== undefined) return fallback;
        throw new Error('Circuit breaker is OPEN');
      }
    }
    
    try {
      const result = await fn();
      this.onSuccess();
      return result;
    } catch (error) {
      this.onFailure();
      if (fallback !== undefined) return fallback;
      throw error;
    }
  }
  
  private onSuccess() {
    this.failures = 0;
    if (this.state === 'HALF_OPEN') {
      this.successes++;
      if (this.successes >= this.config.successThreshold) {
        this.state = 'CLOSED';
      }
    }
  }
  
  private onFailure() {
    this.failures++;
    this.lastFailureTime = Date.now();
    if (this.failures >= this.config.failureThreshold) {
      this.state = 'OPEN';
    }
  }
  
  getState() {
    return this.state;
  }
}

// Usage
const stripeCircuit = new CircuitBreaker({
  failureThreshold: 5,
  successThreshold: 2,
  timeout: 30000, // 30 seconds
});

async function processPayment(amount: number) {
  return stripeCircuit.execute(
    () => stripe.charges.create({ amount }),
    { success: false, error: 'Payment service temporarily unavailable' }
  );
}
```
