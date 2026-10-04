package web

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/cache"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
)

// APIKeyAuth creates middleware for API key authentication
func APIKeyAuth(cfg *config.Config) echo.MiddlewareFunc {
	registry := buildCredentialRegistry(cfg)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Allow OPTIONS requests (CORS preflight) without authentication
			// This must be checked first, before any path checks
			if c.Request().Method == http.MethodOptions {
				return next(c)
			}

			// Use the actual request URL path, not the route pattern
			// This is critical because c.Path() returns route pattern which may be /*
			requestPath := c.Request().URL.Path

			// Skip health checks, metrics, API docs, and Web UI routes
			path := c.Path()
			if path == "/health/live" || path == "/health/ready" || path == "/metrics" {
				return next(c)
			}
			// Skip API documentation routes
			if path == "/docs" || path == "/openapi.yaml" {
				return next(c)
			}

			// Skip Web UI routes - these are publicly accessible
			// Check both route pattern and actual request path
			if path == "/" || path == "/index.html" || path == "/web/*" || strings.HasPrefix(path, "/static/") {
				return next(c)
			}
			// Also check actual request path for web UI files
			if requestPath == "/" || requestPath == "/index.html" ||
				strings.HasPrefix(requestPath, "/static/") ||
				strings.HasPrefix(requestPath, "/web/") ||
				strings.HasPrefix(requestPath, "/assets/") ||
				strings.HasPrefix(requestPath, "/js/") ||
				strings.HasPrefix(requestPath, "/css/") ||
				requestPath == "/web" {
				return next(c)
			}
			// Static assets are public, but only when genuinely fetched as
			// assets. Both guards are load-bearing: without the method check
			// and the /api/ exclusion, any request whose path merely *ends*
			// in ".js" — such as POST /api/ingest/x.js — would skip
			// authentication entirely.
			if m := c.Request().Method; (m == http.MethodGet || m == http.MethodHead) && !strings.HasPrefix(requestPath, "/api/") {
				if strings.HasSuffix(requestPath, ".js") ||
					strings.HasSuffix(requestPath, ".css") ||
					strings.HasSuffix(requestPath, ".html") ||
					strings.HasSuffix(requestPath, ".svg") ||
					strings.HasSuffix(requestPath, ".png") ||
					strings.HasSuffix(requestPath, ".jpg") ||
					strings.HasSuffix(requestPath, ".ico") ||
					strings.HasSuffix(requestPath, ".woff") ||
					strings.HasSuffix(requestPath, ".woff2") ||
					strings.HasSuffix(requestPath, ".ttf") {
					return next(c)
				}
			}

			// Skip read-only API endpoints for public browsing (GET and OPTIONS requests)
			// OPTIONS is needed for CORS preflight requests
			method := c.Request().Method
			if (method == "GET" || method == "OPTIONS") && (path == "/api/documents" || path == "/api/documents/search" ||
				strings.HasPrefix(path, "/api/documents/") ||
				path == "/api/rules" || strings.HasPrefix(path, "/api/rules/") ||
				path == "/api/stats" || strings.HasPrefix(path, "/api/stats/") ||
				path == "/api/projects" || strings.HasPrefix(path, "/api/projects/") ||
				path == "/api/failures" || strings.HasPrefix(path, "/api/failures/") ||
				path == "/api/ingest/status" ||
				path == "/api/ingest/orphans" ||
				path == "/api/updates/status" ||
				path == "/version") {
				return next(c)
			}

			// POST endpoints /api/ingest, /api/ingest/sync, /api/updates/check
			// now require authentication — removed public access to prevent
			// unauthenticated resource exhaustion via document ingestion.

			// Extract API key from header
			authorizationHeader := c.Request().Header.Get("Authorization")
			if authorizationHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization header")
			}

			// Parse Bearer token
			parts := strings.SplitN(authorizationHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization format, expected 'Bearer <api_key>'")
			}

			apiKey := parts[1]

			// Resolve the caller principal from the explicit credential
			// registry (Spec 15 / gr-xp-01). This is the only source of
			// identity: the truncated log hash is never used as a principal,
			// credential, or tenant.
			if principal, ok := registry.Resolve(apiKey); ok {
				c.Set("api_key_type", "registered")
				c.Set("principal_id", principal.ID)
				c.Set("credential_id", principal.CredentialID)
				c.Set("credential_scopes", principal.Scopes)
				// hashAPIKey remains only a non-authoritative log/rate-limit
				// correlation value; it is never used as identity.
				c.Set("api_key_hash", hashAPIKey(apiKey))

				slog.Debug("API request authenticated",
					"principal_id", principal.ID,
					"credential_id", principal.CredentialID,
					"path", path,
				)
				return next(c)
			}

			// Fall back to the two legacy keys, which keep their current
			// limited surface. When a registry is configured an unregistered
			// legacy credential is confined to the enumerated legacy-safe
			// surface and denied privileged/mutating actions.
			legacyClass := ""
			if subtle.ConstantTimeCompare([]byte(apiKey), []byte(cfg.MCPAPIKey)) == 1 {
				legacyClass = "mcp"
			} else if subtle.ConstantTimeCompare([]byte(apiKey), []byte(cfg.IDEAPIKey)) == 1 {
				legacyClass = "ide"
			}
			if legacyClass == "" {
				slog.Warn("Invalid API key attempt",
					"ip", c.RealIP(),
					"path", path,
				)
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API key")
			}

			if registry.Enabled() && !isLegacySafe(method, requestPath) {
				slog.Warn("Unregistered credential denied privileged action",
					"credential_class", "legacy_"+legacyClass,
					"method", method,
					"path", path,
					"ip", c.RealIP(),
				)
				return echo.NewHTTPError(http.StatusForbidden, "unregistered credential not permitted for this action")
			}

			c.Set("api_key_type", legacyClass)
			c.Set("api_key_hash", hashAPIKey(apiKey))

			slog.Debug("API request authenticated",
				"key_type", legacyClass,
				"path", path,
			)

			return next(c)
		}
	}
}

// isLegacySafe reports whether a request is on the explicitly enumerated
// legacy-safe surface the two legacy keys may still drive once a credential
// registry is configured. Read-only requests and the endpoints the legacy keys
// were designed for stay permitted; every other mutating/privileged action
// requires a credential registered with a principal.
func isLegacySafe(method, path string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	switch path {
	case "/api/ingest", "/api/ingest/sync", "/api/updates/check":
		return true
	}
	return strings.HasPrefix(path, "/ide/")
}

// buildCredentialRegistry loads the credential-to-principal registry from
// configuration. An absent registry is normal and preserves the legacy two-key
// behaviour. A registry that is configured but unloadable is logged and treated
// as absent rather than failing open on a privileged surface.
func buildCredentialRegistry(cfg *config.Config) *auth.Registry {
	if cfg == nil {
		return nil
	}
	registry, err := auth.LoadFromSources(cfg.CredentialRegistryJSON, cfg.CredentialRegistryFile, cfg.CredentialVerifierKey)
	if err != nil {
		slog.Error("Failed to load credential registry; legacy key behaviour retained", "error", err)
		return nil
	}
	return registry
}

// RateLimitMiddleware creates middleware for rate limiting
func RateLimitMiddleware(limiter *cache.DistributedRateLimiter, cfg *config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Use the actual request URL path
			requestPath := c.Request().URL.Path

			// Skip health checks, API docs, and Web UI routes
			path := c.Path()
			if path == "/health/live" || path == "/health/ready" || path == "/metrics" {
				return next(c)
			}
			if path == "/docs" || path == "/openapi.yaml" {
				return next(c)
			}

			// Skip Web UI routes - these are publicly accessible
			if path == "/" || path == "/index.html" || path == "/web/*" || strings.HasPrefix(path, "/static/") {
				return next(c)
			}
			// Also check actual request path for web UI files
			if requestPath == "/" || requestPath == "/index.html" ||
				strings.HasPrefix(requestPath, "/static/") ||
				strings.HasPrefix(requestPath, "/web/") ||
				strings.HasPrefix(requestPath, "/assets/") ||
				strings.HasPrefix(requestPath, "/js/") ||
				strings.HasPrefix(requestPath, "/css/") ||
				requestPath == "/web" {
				return next(c)
			}
			// Static assets are public, but only when genuinely fetched as
			// assets. Same load-bearing guards as APIKeyAuth: a request whose
			// path merely ends in ".js" must not escape rate limiting.
			if m := c.Request().Method; (m == http.MethodGet || m == http.MethodHead) && !strings.HasPrefix(requestPath, "/api/") {
				if strings.HasSuffix(requestPath, ".js") ||
					strings.HasSuffix(requestPath, ".css") ||
					strings.HasSuffix(requestPath, ".html") ||
					strings.HasSuffix(requestPath, ".svg") ||
					strings.HasSuffix(requestPath, ".png") ||
					strings.HasSuffix(requestPath, ".jpg") ||
					strings.HasSuffix(requestPath, ".ico") ||
					strings.HasSuffix(requestPath, ".woff") ||
					strings.HasSuffix(requestPath, ".woff2") ||
					strings.HasSuffix(requestPath, ".ttf") {
					return next(c)
				}
			}

			// Skip read-only API endpoints for public browsing (GET and OPTIONS requests)
			// OPTIONS is needed for CORS preflight requests
			method := c.Request().Method
			if (method == "GET" || method == "OPTIONS") && (path == "/api/documents" || path == "/api/documents/search" ||
				strings.HasPrefix(path, "/api/documents/") ||
				path == "/api/rules" || strings.HasPrefix(path, "/api/rules/") ||
				path == "/api/stats" || strings.HasPrefix(path, "/api/stats/") ||
				path == "/api/projects" || strings.HasPrefix(path, "/api/projects/") ||
				path == "/api/failures" || strings.HasPrefix(path, "/api/failures/") ||
				path == "/api/ingest/status" ||
				path == "/api/ingest/orphans" ||
				path == "/api/updates/status" ||
				path == "/version") {
				return next(c)
			}

			// Write endpoints are deliberately NOT skipped here. /api/ingest
			// and /api/updates/check are the most expensive handlers in the
			// server; exempting them let any caller issue them without limit,
			// which is the resource-exhaustion vector closed in APIKeyAuth.

			// Determine rate limit based on endpoint and key type
			var limit int
			keyType := c.Get("api_key_type")

			if strings.HasPrefix(path, "/ide") {
				limit = cfg.RateLimitIDE
			} else {
				limit = cfg.RateLimitMCP
			}

			// Use API key hash as rate limit key
			keyHash, ok := c.Get("api_key_hash").(string)
			if !ok {
				keyHash = c.RealIP()
			}

			// Check rate limit
			if !limiter.Allow(c.Request().Context(), keyHash, limit) {
				slog.Warn("Rate limit exceeded",
					"key_type", keyType,
					"key_hash", keyHash,
					"path", path,
					"limit", limit,
				)
				return echo.NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded")
			}

			return next(c)
		}
	}
}

// hashAPIKey creates a hash of the API key for logging and rate-limit
// bucketing. It is a non-authoritative correlation value: identity is resolved
// from the credential registry, never from this truncated digest.
func hashAPIKey(key string) string {
	// Use stack-allocated array for hashing
	var h [32]byte
	h = sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:8])
}
