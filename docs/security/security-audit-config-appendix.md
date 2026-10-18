# Security Audit — Secure Configuration Template

> Appendix A from the configuration security audit: a reviewed secure-configuration template.

**Part of:** [Security Audit: Configuration Files and Environment Handling](./security-audit-config.md)

---

## Appendix A: Secure Configuration Template

```bash
# Production-ready .env file template
PRODUCTION_MODE=true

# Server
MCP_PORT=8080
WEB_PORT=8081
LOG_LEVEL=warn

# Database (SSL required)
DB_HOST=postgres.example.com
DB_PORT=5432
DB_NAME=guardrails
DB_USER=guardrail
DB_PASSWORD=<GENERATE_STRONG_PASSWORD>
DB_SSLMODE=verify-full

# Redis (TLS required)
REDIS_HOST=redis.example.com
REDIS_PORT=6379
REDIS_PASSWORD=<GENERATE_STRONG_PASSWORD>
REDIS_USE_TLS=true

# TLS (Required for production)
TLS_ENABLED=true
TLS_CERT_PATH=/etc/ssl/certs/server.crt
TLS_KEY_PATH=/etc/ssl/private/server.key
TLS_CA_PATH=/etc/ssl/certs/ca.crt
TLS_MIN_VERSION=1.3

# Secrets (Generate with: openssl rand -hex 32)
MCP_API_KEY=<64_CHAR_HEX_STRING>
IDE_API_KEY=<64_CHAR_HEX_STRING>
JWT_SECRET=<128_CHAR_HEX_STRING>

# CORS (Explicit origins only)
CORS_ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com

# Security
PPROF_ENABLED=false
RATE_LIMIT_MCP=1000
RATE_LIMIT_IDE=500
```

