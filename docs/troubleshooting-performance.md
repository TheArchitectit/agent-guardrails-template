# Troubleshooting: Performance Issues

> Diagnosing and resolving slow operations, high memory usage, and rate limiting

See the [Troubleshooting Index](troubleshooting.md) for other topics.

---

## Slow Team Operations

**Symptoms:**
- Team list takes > 5 seconds
- Batch assignments timeout
- Phase gate checks are slow

**Diagnosis:**
```bash
# Check response times
time curl -s -X POST http://localhost:8094/mcp \
  -d '...team_list...'

# Monitor server resources
htop
iostat -x 1
```

**Solutions:**

1. **Enable caching:**
   ```json
   {
     "cache": {
       "enabled": true,
       "ttl": 30
     }
   }
   ```

2. **Optimize batch operations:**
   ```bash
   # Use parallel processing
   python scripts/batch_execute.py --parallel 8
   ```

3. **Increase timeouts:**
   ```bash
   curl --max-time 30 ...
   ```

---

## High Memory Usage

**Symptoms:**
- MCP server using > 500MB RAM
- System swapping
- Out of memory errors

**Diagnosis:**
```bash
# Check memory usage
ps aux | grep mcp_server
free -h

# Monitor over time
watch -n 1 'ps -o pid,rss,cmd -p $(pgrep -f mcp_server)'
```

**Solutions:**

1. **Limit cache size:**
   ```json
   {
     "cache": {
       "max_size": 100,
       "ttl": 30
     }
   }
   ```

2. **Restart server periodically:**
   ```bash
   # Add to cron
   0 */6 * * * systemctl restart mcp-server
   ```

3. **Profile memory usage:**
   ```bash
   python -m memory_profiler mcp_server.py
   ```

---

## Rate Limiting

**Symptoms:**
```
Error: RATE-001: Rate limit exceeded
Retry after 60 seconds
```

**Solutions:**

1. **Implement backoff:**
   ```python
   import time
   import random

   def with_backoff(func, max_retries=5):
       for i in range(max_retries):
           try:
               return func()
           except RateLimitError:
               wait = (2 ** i) + random.random()
               time.sleep(wait)
       raise MaxRetriesExceeded()
   ```

2. **Use batch endpoints:**
   ```bash
   # Instead of multiple single calls
   python scripts/batch_execute.py --file operations.json
   ```

3. **Request limit increase:**
   Contact support to increase rate limits for your use case.
