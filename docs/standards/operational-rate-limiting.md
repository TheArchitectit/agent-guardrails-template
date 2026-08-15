# Rate Limiting

> Control request flow with token buckets and proper headers

**See also:** [Operational Patterns overview](operational-patterns.md)

---

## Token Bucket Implementation

```typescript
// rate-limiter.ts

class TokenBucket {
  private tokens: number;
  private lastRefill: number;
  
  constructor(
    private capacity: number,
    private refillRate: number, // tokens per second
  ) {
    this.tokens = capacity;
    this.lastRefill = Date.now();
  }
  
  tryConsume(tokens: number = 1): boolean {
    this.refill();
    
    if (this.tokens >= tokens) {
      this.tokens -= tokens;
      return true;
    }
    
    return false;
  }
  
  private refill() {
    const now = Date.now();
    const elapsed = (now - this.lastRefill) / 1000;
    const tokensToAdd = elapsed * this.refillRate;
    
    this.tokens = Math.min(this.capacity, this.tokens + tokensToAdd);
    this.lastRefill = now;
  }
}

// Usage in middleware
const rateLimiters = new Map<string, TokenBucket>();

function rateLimitMiddleware(req, res, next) {
  const key = req.ip; // or req.user.id for authenticated users
  
  if (!rateLimiters.has(key)) {
    rateLimiters.set(key, new TokenBucket(100, 10)); // 100 burst, 10/sec steady
  }
  
  const bucket = rateLimiters.get(key)!;
  
  if (bucket.tryConsume()) {
    next();
  } else {
    res.status(429).json({
      error: 'Too Many Requests',
      retryAfter: 10,
    });
  }
}
```

## Rate Limit Headers

```typescript
// Always return rate limit information in headers

function setRateLimitHeaders(res, bucket) {
  res.setHeader('X-RateLimit-Limit', bucket.capacity);
  res.setHeader('X-RateLimit-Remaining', Math.floor(bucket.tokens));
  res.setHeader('X-RateLimit-Reset', Date.now() + 60000);
}
```
