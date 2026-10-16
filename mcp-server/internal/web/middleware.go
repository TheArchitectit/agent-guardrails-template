package web

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thearchitectit/guardrail-mcp/internal/audit"
	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/cache"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/metrics"
)

// APIKeyAuth creates middleware for API key authentication and authorization.
// After a credential resolves to a server-controlled principal, the Spec 11
// section 4.4 intersection (authenticated AND scope_allows AND role_allows AND
// resource_allows) is applied before the request reaches its handler.
// Identity/role/tenant are never taken from request content.
func APIKeyAuth(cfg *config.Config, auditLogger *audit.Logger) echo.MiddlewareFunc {
	registry, regErr := buildCredentialRegistry(cfg)
	// Spec 11 §4.5.2: parse the legacy-acceptance removal deadline once at
	// construction; the deadline is compared per request so a passed deadline
	// takes effect without a restart. A malformed value fails closed below.
	legacyDeadline, legacyConfigured, legacyDeadlineErr := cfg.LegacyAcceptanceDeadline()
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Allow OPTIONS requests (CORS preflight) without authentication.
			// Preflight is not authorization for the corresponding effect.
			if c.Request().Method == http.MethodOptions {
				return next(c)
			}

			// Use the actual request URL path, not the route pattern.
			requestPath := c.Request().URL.Path
			method := c.Request().Method

			// Spec 12 §3.1: the declared exposure registry is the single source of
			// truth for this middleware and for RateLimitMiddleware, so the two
			// can no longer drift. A route that is not declared public is
			// authenticated (deny by default).
			path := c.Path()
			if ExposureFor(method, path, requestPath) == ExposurePublic {
				return next(c)
			}

			// A registry that is configured but failed to load is a hard
			// fail-closed condition: deny ALL protected traffic. Never fall
			// back to unrestricted legacy access on a broken registry.
			if regErr != nil {
				slog.Error("credential registry configured but failed to load; denying all protected traffic",
					"error", regErr,
					"path", path,
					"ip", c.RealIP(),
				)
				return echo.NewHTTPError(http.StatusServiceUnavailable, "credential registry unavailable")
			}

			authorizationHeader := c.Request().Header.Get("Authorization")
			if authorizationHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization header")
			}

			parts := strings.SplitN(authorizationHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization format, expected 'Bearer <api_key>'")
			}

			apiKey := parts[1]

			// Resolve the caller principal from the explicit credential
			// registry. Identity never comes from the truncated log hash.
			var caller auth.Caller
			var action auth.ActionRequest
			isLegacy := false

			if principal, ok := registry.Resolve(apiKey); ok {
				caller = principal.Caller()
				c.Set("api_key_type", "registered")
				c.Set("principal_id", caller.PrincipalID)
				c.Set("credential_id", caller.CredentialID)
				c.Set("credential_scopes", caller.Scopes)
				c.Set("principal_role", caller.Role)
				c.Set("principal_resources", caller.Resources)
				c.Set("caller", caller)
				// hashAPIKey is a non-authoritative log/rate-limit correlation
				// value only; it is never used as identity.
				c.Set("api_key_hash", hashAPIKey(apiKey))
				action = auth.ClassifyREST(method, requestPath)
			} else {
				// Legacy keys. When a registry is configured an unregistered
				// legacy credential is confined to the explicit method/path
				// allowlist (Spec 16 section 4) and the same privilege model.
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

				legacyCaller, ok := auth.LegacyPrincipal(legacyClass)
				if !ok {
					return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API key")
				}

				// Spec 11 §4.5.2 / acceptance criterion 8: the legacy-acceptance
				// removal switch. Once the configured deadline has passed, a legacy
				// key is refused on this surface regardless of registry state. A
				// malformed deadline is a hard failure, never a silent pass.
				if legacyDeadlineErr != nil {
					slog.Error("legacy-acceptance deadline is malformed; refusing legacy credential",
						"error", legacyDeadlineErr,
						"path", path,
					)
					return echo.NewHTTPError(http.StatusServiceUnavailable, "legacy acceptance policy unavailable")
				}
				if legacyConfigured && time.Now().After(legacyDeadline) {
					slog.Warn("Legacy credential refused: acceptance window closed",
						"credential_class", "legacy_"+legacyClass,
						"path", path,
						"ip", c.RealIP(),
					)
					restAuditDecision(c, auditLogger, legacyCaller, auth.ActionRequest{
						Name: method + " " + requestPath, Kind: "unknown",
					}, auth.Decision{Allow: false, Code: auth.ReasonLegacyNotAllowed, Reason: "legacy acceptance window closed"}, false)
					return echo.NewHTTPError(http.StatusForbidden, "legacy credential acceptance window closed")
				}

				caller = legacyCaller
				isLegacy = true

				// Spec 11 §4.5.2: accepted legacy use is counted so the migration
				// window is measured (a zero rate gates removal), never assumed.
				metrics.RecordLegacyCredentialUse("rest", legacyClass)

				if registry.Enabled() {
					// Explicit method/path allowlist. No safe-method wildcard
					// and no /ide/ prefix wildcard.
					la, ok := auth.LegacyRESTAction(legacyClass, method, requestPath)
					if !ok {
						slog.Warn("Legacy credential denied: not on approved method/path table",
							"credential_class", "legacy_"+legacyClass,
							"method", method,
							"path", path,
							"ip", c.RealIP(),
						)
						restAuditDecision(c, auditLogger, caller, auth.ActionRequest{
							Name: method + " " + requestPath, Kind: "unknown",
						}, auth.Decision{Allow: false, Code: auth.ReasonLegacyNotAllowed, Reason: "legacy method/path not on approved table"}, false)
						return echo.NewHTTPError(http.StatusForbidden, "unregistered credential not permitted for this action")
					}
					action = la
				} else {
					// Migration mode (no registry configured): legacy keys keep
					// their historical surface. Containment applies once a
					// registry is enabled.
					action = auth.ClassifyREST(method, requestPath)
				}

				c.Set("api_key_type", legacyClass)
				c.Set("principal_id", caller.PrincipalID)
				c.Set("credential_id", caller.CredentialID)
				c.Set("principal_role", caller.Role)
				c.Set("caller", caller)
				c.Set("api_key_hash", hashAPIKey(apiKey))
			}

			// Spec 11 section 4.4 intersection. Missing/unknown inputs deny.
			// Registered credentials and legacy keys under a configured
			// registry both pass through this gate. Migration mode (no
			// registry) preserves the legacy surface without the intersection.
			enforce := !isLegacy || registry.Enabled()
			if enforce {
				decision := auth.Decide(caller, action)
				if !decision.Allow {
					slog.Warn("Authorization denied",
						"principal_id", caller.PrincipalID,
						"credential_id", caller.CredentialID,
						"action", action.Name,
						"kind", string(action.Kind),
						"resource", action.Resource,
						"reason", decision.Code,
						"path", path,
					)
					restAuditDecision(c, auditLogger, caller, action, decision, false)
					return echo.NewHTTPError(auth.HTTPStatusForDecision(decision), "permission denied")
				}

				// Security-sensitive mutations require a durable audit record
				// before the effect. Fail closed if it cannot be written.
				required := action.Kind == auth.ActionAdmin
				if err := restAuditDecision(c, auditLogger, caller, action, decision, required); err != nil {
					slog.Error("required audit failed; denying security-sensitive mutation",
						"error", err,
						"principal_id", caller.PrincipalID,
						"action", action.Name,
					)
					return echo.NewHTTPError(http.StatusServiceUnavailable, "authorization audit unavailable")
				}
			} else {
				_ = restAuditDecision(c, auditLogger, caller, action,
					auth.Decision{Allow: true, Code: auth.ReasonAllowed, Reason: "legacy migration mode"}, false)
			}

			slog.Debug("API request authorized",
				"principal_id", caller.PrincipalID,
				"credential_id", caller.CredentialID,
				"path", path,
				"action", action.Name,
				"kind", string(action.Kind),
			)

			return next(c)
		}
	}
}

// restAuditDecision writes a structured allow/deny decision record. It never
// logs raw keys and never uses hashAPIKey as identity.
func restAuditDecision(c echo.Context, logger *audit.Logger, caller auth.Caller, action auth.ActionRequest, d auth.Decision, required bool) error {
	if logger == nil {
		if required {
			return errAuditUnavailable
		}
		return nil
	}
	decision := auth.DecisionAllow
	if !d.Allow {
		decision = auth.DecisionDeny
	}
	return logger.LogDecision(c.Request().Context(), audit.DecisionRecord{
		PrincipalID:   caller.PrincipalID,
		CredentialID:  caller.CredentialID,
		Action:        action.Name,
		Resource:      action.Resource,
		Decision:      decision,
		Reason:        d.Code,
		PolicyVersion: auth.PolicyVersion,
		Kind:          string(action.Kind),
		Surface:       "rest",
	}, required)
}

// errAuditUnavailable is returned when a required durable audit record cannot
// be produced.
var errAuditUnavailable = echo.NewHTTPError(http.StatusServiceUnavailable, "authorization audit unavailable")

// buildCredentialRegistry loads the credential-to-principal registry from
// configuration. An absent registry is normal and preserves the legacy two-key
// behaviour. A registry that is configured but unloadable returns an error so
// callers can fail closed; it is never treated as absent.
func buildCredentialRegistry(cfg *config.Config) (*auth.Registry, error) {
	if cfg == nil {
		return nil, nil
	}
	registry, err := auth.LoadFromSourcesEx(cfg.CredentialRegistryJSON, cfg.CredentialRegistryFile, cfg.CredentialVerifierKey, cfg.CredentialVerifierKeyFile)
	if err != nil {
		slog.Error("Failed to load credential registry; protected traffic will be denied", "error", err)
		return nil, err
	}
	return registry, nil
}

// RateLimitMiddleware creates middleware for rate limiting
func RateLimitMiddleware(limiter *cache.DistributedRateLimiter, cfg *config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestPath := c.Request().URL.Path
			path := c.Path()

			// Spec 12 §3.1: the same declared exposure registry APIKeyAuth uses.
			// A route that is not declared public is authenticated, so the rate
			// limiter can no longer keep a private skip list of its own.
			if ExposureFor(c.Request().Method, path, requestPath) == ExposurePublic {
				return next(c)
			}

			var limit int
			keyType := c.Get("api_key_type")

			if strings.HasPrefix(path, "/ide") {
				limit = cfg.RateLimitIDE
			} else {
				limit = cfg.RateLimitMCP
			}

			keyHash, ok := c.Get("api_key_hash").(string)
			if !ok {
				keyHash = c.RealIP()
			}

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
	var h [32]byte
	h = sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:8])
}
