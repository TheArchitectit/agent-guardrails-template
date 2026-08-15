# Troubleshooting: API Key Authentication

> Diagnosing and fixing authentication errors

See the [Troubleshooting Index](troubleshooting.md) for other topics.

---

## API Key Authentication Failed

**Symptoms:**
```
Error: AUTH-001: Authentication required
Error: AUTH-002: Invalid API key
```

**Causes:**
- Missing Authorization header
- Expired or revoked API key
- Incorrect API key format

**Solutions:**

1. **Verify header format:**
   ```bash
   curl -H "Authorization: Bearer YOUR_API_KEY" ...
   ```

2. **Generate new API key:**
   ```bash
   python scripts/generate_api_key.py
   ```

3. **Check key permissions:**
   ```bash
   python scripts/verify_api_key.py YOUR_API_KEY
   ```
