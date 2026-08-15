# Retry and Graceful Degradation

> Handle transient failures with backoff, and serve something useful when services go down

**See also:** [Operational Patterns overview](operational-patterns.md) | [Circuit Breakers](operational-circuit-breakers.md)

---

## Exponential Backoff

```typescript
// retry.ts - Exponential backoff with jitter

interface RetryConfig {
  maxAttempts: number;
  baseDelay: number;
  maxDelay: number;
  jitter: boolean;
}

async function withRetry<T>(
  fn: () => Promise<T>,
  config: RetryConfig
): Promise<T> {
  let lastError: Error | null = null;
  
  for (let attempt = 1; attempt <= config.maxAttempts; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error as Error;
      
      if (attempt === config.maxAttempts) break;
      
      // Calculate delay with exponential backoff
      let delay = Math.min(
        config.baseDelay * Math.pow(2, attempt - 1),
        config.maxDelay
      );
      
      // Add jitter to prevent thundering herd
      if (config.jitter) {
        delay = delay * (0.5 + Math.random());
      }
      
      console.log(`Attempt ${attempt} failed, retrying in ${delay}ms`);
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }
  
  throw lastError;
}

// Usage
const data = await withRetry(
  () => fetchFromUnstableAPI(),
  {
    maxAttempts: 3,
    baseDelay: 1000,
    maxDelay: 10000,
    jitter: true,
  }
);
```

## Retry vs Circuit Breaker

```
WHEN TO USE EACH:

RETRY:
- Transient failures (network blip)
- Usually succeeds on second try
- Few failures, quick recovery
- Example: Database connection dropped

CIRCUIT BREAKER:
- Persistent failures (service down)
- Won't succeed even with retries
- Many failures, slow recovery
- Example: Third-party API outage

BEST PRACTICE: Use both together
1. Retry handles transient failures
2. If retries keep failing → Circuit breaker trips
3. Circuit breaker prevents retry storms
```

---

## Fallback Strategies

```typescript
// degradation.ts - Graceful degradation patterns

// PATTERN 1: Default Value Fallback
async function getRecommendations(userId: string): Promise<Product[]> {
  try {
    return await mlService.getPersonalizedRecommendations(userId);
  } catch (error) {
    // Fallback: Return popular products instead
    return await getPopularProducts();
  }
}

// PATTERN 2: Cached Value Fallback
async function getExchangeRate(currency: string): Promise<number> {
  try {
    const rate = await exchangeRateAPI.getRate(currency);
    await cache.set(`rate:${currency}`, rate, 3600); // Cache for 1 hour
    return rate;
  } catch (error) {
    // Fallback: Return cached rate (may be stale)
    const cached = await cache.get(`rate:${currency}`);
    if (cached) {
      console.warn(`Using cached exchange rate for ${currency}`);
      return cached;
    }
    throw error; // No fallback available
  }
}

// PATTERN 3: Feature Toggle Fallback
async function searchProducts(query: string): Promise<SearchResult> {
  if (featureFlags.get('advanced_search')) {
    try {
      return await elasticSearch.search(query);
    } catch (error) {
      console.warn('Elasticsearch failed, falling back to basic search');
    }
  }
  // Fallback: Basic database search
  return await db.products.find({ name: { $regex: query } });
}

// PATTERN 4: Partial Result Fallback
async function getDashboard(userId: string): Promise<Dashboard> {
  const [userResult, ordersResult, recommendationsResult] = await Promise.allSettled([
    getUser(userId),
    getOrders(userId),
    getRecommendations(userId),
  ]);
  
  return {
    user: userResult.status === 'fulfilled' ? userResult.value : null,
    orders: ordersResult.status === 'fulfilled' ? ordersResult.value : [],
    recommendations: recommendationsResult.status === 'fulfilleded' 
      ? recommendationsResult.value 
      : [], // Empty array if recommendations fail
    warnings: [
      userResult.status === 'rejected' && 'User data unavailable',
      ordersResult.status === 'rejected' && 'Order history unavailable',
      recommendationsResult.status === 'rejected' && 'Recommendations unavailable',
    ].filter(Boolean),
  };
}
```
