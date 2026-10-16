package web

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thearchitectit/guardrail-mcp/internal/audit"
	"github.com/thearchitectit/guardrail-mcp/internal/cache"
	"github.com/thearchitectit/guardrail-mcp/internal/clientip"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
	"github.com/thearchitectit/guardrail-mcp/internal/ingest"
	metricsMiddleware "github.com/thearchitectit/guardrail-mcp/internal/metrics"
	loggingMiddleware "github.com/thearchitectit/guardrail-mcp/internal/middleware"
	"github.com/thearchitectit/guardrail-mcp/internal/updates"
)

// Server wraps the Echo server with guardrail dependencies
type Server struct {
	echo          *echo.Echo
	cfg           *config.Config
	db            *database.DB
	cache         *cache.Client
	auditLogger   *audit.Logger
	docStore      *database.DocumentStore
	ruleStore     *database.RuleStore
	projStore     *database.ProjectStore
	failStore     *database.FailureStore
	ingestSvc     *ingest.Service
	updateChecker *updates.Checker
	version       string

	// lifecycle is the explicit process lifecycle state observed by
	// /health/ready (Spec 18 §5.2 / R18-12). It is accessed atomically because
	// shutdown may move it to DRAINING from the signal goroutine while probes
	// are being served.
	lifecycle atomic.Value // LifecycleState

	// readiness runs the ordered readiness checks (Spec 18 §3 / R18-07..R18-13).
	readiness *ReadinessEvaluator
}

// NewServer creates a new web server
func NewServer(cfg *config.Config, db *database.DB, cacheClient *cache.Client, auditLogger *audit.Logger, version string) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	docStore := database.NewDocumentStore(db)

	s := &Server{
		echo:          e,
		cfg:           cfg,
		db:            db,
		cache:         cacheClient,
		auditLogger:   auditLogger,
		docStore:      docStore,
		ruleStore:     database.NewRuleStore(db),
		projStore:     database.NewProjectStore(db),
		failStore:     database.NewFailureStore(db),
		ingestSvc:     ingest.NewService(docStore, database.NewRuleStore(db), []string{"/app/docs"}, "/app/docs"),
		updateChecker: updates.NewChecker(db, version, os.Getenv("GIT_COMMIT")),
		version:       version,
	}

	s.lifecycle.Store(LifecycleStarting)

	// The ordered readiness evaluator (Spec 18 §3). Live probes are bound here
	// so the ordered decision itself stays dependency-free and testable.
	s.readiness = NewReadinessEvaluator(version, ReadinessDeps{
		DBHealth: func(ctx context.Context) error {
			if db == nil {
				return nil
			}
			return db.HealthCheck(ctx)
		},
		CacheHealth: func(ctx context.Context) error {
			if cacheClient == nil {
				return nil
			}
			return cacheClient.HealthCheck(ctx)
		},
		SchemaCompatibility: func(ctx context.Context) error {
			if db == nil {
				return nil
			}
			return database.CheckSchemaCompatibility(ctx, database.NewSQLSchemaLedger(db.DB), database.MinRequiredSchemaVersion)
		},
		RegistryConfigured: config.SourcesConfiguredRegistry(cfg),
		RegistryLoadErr:    registryLoadErr(cfg),
		Policy:             policyState(),
		PolicyRequired:     false,
	})

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// SetLifecycle moves the process lifecycle state (Spec 18 §5.2 / R18-12).
// Readiness observes it: DRAINING and STOPPED are non-ready by construction.
func (s *Server) SetLifecycle(state LifecycleState) {
	s.lifecycle.Store(state)
}

// Lifecycle returns the current lifecycle state.
func (s *Server) Lifecycle() LifecycleState {
	if v, ok := s.lifecycle.Load().(LifecycleState); ok {
		return v
	}
	return LifecycleStarting
}

// EvaluateReadiness runs the ordered readiness checks against the current
// lifecycle state (Spec 18 §3).
func (s *Server) EvaluateReadiness() ReadinessReport {
	return s.readiness.Evaluate(s.Lifecycle())
}

// registryLoadErr reports a configured-but-unloadable credential registry
// without exposing verifier material (Spec 18 §6 / R18-10).
func registryLoadErr(cfg *config.Config) error {
	if !config.SourcesConfiguredRegistry(cfg) {
		return nil
	}
	_, err := buildCredentialRegistry(cfg)
	return err
}

// policyState reports the policy engine posture. The engine is driven by an
// Ollama-backed content filter (see cmd/server); when OLLAMA_URL is unset the
// engine is not configured. Configured/enabled/exercised are distinct facts
// (Spec 18 §6 / R18-11).
func policyState() PolicyState {
	if os.Getenv("OLLAMA_URL") == "" {
		return PolicyState{Configured: false, Enabled: false, Exercised: false}
	}
	return PolicyState{Configured: true, Enabled: true, Exercised: false, Version: os.Getenv("OLLAMA_MODEL")}
}

// Echo exposes the underlying Echo instance for external route registration.
func (s *Server) Echo() *echo.Echo {
	return s.echo
}

// setupMiddleware configures Echo middleware
func (s *Server) setupMiddleware() {
	// Client-IP trust boundary (Spec 12 §3.4). Forwarded headers are honoured
	// only from a configured TRUSTED_PROXIES peer; otherwise the socket peer is
	// used and a forwarded header is recorded as a protocol anomaly. The
	// stance is validated at config load, so a parse error here falls back to
	// the safe default (trust no proxy) and is logged.
	stance, err := clientip.NewStance(s.cfg.TrustedProxies)
	if err != nil {
		slog.Error("invalid TRUSTED_PROXIES; trusting no proxy", "error", err)
		stance, _ = clientip.NewStance(nil)
	}
	s.echo.IPExtractor = stance.Extractor()

	// Request ID generation
	s.echo.Use(middleware.RequestID())

	// Correlation ID propagation
	s.echo.Use(correlationIDMiddleware())

	// Recovery from panics with metrics
	s.echo.Use(panicRecoveryMiddleware())

	// Prometheus metrics middleware
	s.echo.Use(metricsMiddleware.PrometheusMiddleware())

	// Request logging
	s.echo.Use(loggingMiddleware.RequestLogger())

	// Security headers
	s.echo.Use(securityHeadersMiddleware())

	// Forwarded-header protocol anomaly (Spec 12 §3.4.4): an untrusted peer
	// that asserts a forwarded header is logged, not rejected.
	s.echo.Use(forwardedHeaderAnomalyMiddleware(stance))

	// CORS - MUST run before auth middleware to handle OPTIONS preflight requests
	// CORS preflight (OPTIONS) requests should return 200 before auth check.
	//
	// Spec 12 §3.3.4: a preflight is answered for a DECLARED route only. This
	// guard runs before CORS and refuses an OPTIONS request whose matched route
	// is a catch-all (the Web UI file/SPA fallback), so OPTIONS cannot declare
	// an arbitrary path. Spec 12 §3.3.2: production wildcard/empty origins are
	// a startup failure, never a substitution — EffectiveCORSOrigins fails
	// closed here as well as in config.Validate.
	s.echo.Use(declaredPreflightOnly())

	corsOrigins, substituted, err := s.cfg.EffectiveCORSOrigins()
	if err != nil {
		// Fail closed: an invalid CORS posture must not become a widened one.
		// echo replaces an empty AllowOrigins with the wildcard default, so a
		// sentinel that matches nothing is used instead of an empty slice.
		slog.Error("invalid CORS configuration; refusing to serve cross-origin requests", "error", err)
		corsOrigins = []string{"https://invalid.invalid"}
	}
	if substituted {
		slog.Warn("CORS_ALLOWED_ORIGINS unset or wildcard; using restrictive development defaults", "origins", corsOrigins)
	}
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: corsOrigins,
		AllowMethods: s.cfg.CORSAllowedMethods,
		AllowHeaders: s.cfg.CORSAllowedHeaders,
		MaxAge:       s.cfg.CORSMaxAge,
	}))

	// API Key Authentication (required for all routes except health/metrics)
	s.echo.Use(APIKeyAuth(s.cfg, s.auditLogger))

	// Rate Limiting
	limiter := s.cache.NewDistributedLimiter()
	s.echo.Use(RateLimitMiddleware(limiter, s.cfg))

	// Request timeout (skip vision endpoints — they have long-running inference)
	s.echo.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: s.cfg.RequestTimeout,
		Skipper: func(c echo.Context) bool {
			return strings.HasPrefix(c.Request().URL.Path, "/v1/vision")
		},
	}))

	// Body limit
	s.echo.Use(middleware.BodyLimit("10M"))

	// Cache control - prevent caching of API responses
	s.echo.Use(cacheControlMiddleware())
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	// Health endpoints (no auth required)
	s.echo.GET("/health/live", s.healthLive)
	s.echo.GET("/health/ready", s.healthReady)

	// Version endpoint (no auth required)
	s.echo.GET("/version", s.versionInfo)

	// Metrics endpoint
	s.echo.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// API routes with authentication
	api := s.echo.Group("/api")

	// Document routes - order matters: specific routes before parameterized routes
	api.GET("/documents", s.listDocuments)
	api.GET("/documents/search", s.searchDocuments)
	api.GET("/documents/:id", s.getDocument)
	api.PUT("/documents/:id", s.updateDocument)

	// Rule routes
	api.GET("/rules", s.listRules)
	api.GET("/rules/:id", s.getRule)
	api.POST("/rules", s.createRule)
	api.PUT("/rules/:id", s.updateRule)
	api.DELETE("/rules/:id", s.deleteRule)
	api.PATCH("/rules/:id", s.patchRule)

	// Rule sync routes
	api.POST("/rules/sync", s.syncRules)
	api.GET("/rules/sync/status", s.getRuleSyncStatus)
	api.POST("/rules/sync/upload", s.triggerRuleSyncFromUpload)

	// Project routes
	api.GET("/projects", s.listProjects)
	api.GET("/projects/:id", s.getProject)
	api.POST("/projects", s.createProject)
	api.PUT("/projects/:id", s.updateProject)
	api.DELETE("/projects/:id", s.deleteProject)

	// Failure registry routes
	api.GET("/failures", s.listFailures)
	api.GET("/failures/:id", s.getFailure)
	api.POST("/failures", s.createFailure)
	api.PUT("/failures/:id", s.updateFailure)

	// System routes
	api.GET("/stats", s.getStats)
	api.POST("/ingest", s.triggerIngest)

	// Update routes
	api.GET("/updates/status", s.getUpdateStatus)
	api.POST("/updates/check", s.checkForUpdates)

	// Ingest routes
	api.POST("/ingest/upload", s.uploadFiles)
	api.POST("/ingest/sync", s.syncRepo)
	api.GET("/ingest/status", s.getIngestStatus)
	api.GET("/ingest/orphans", s.listOrphans)
	api.DELETE("/ingest/orphans/:id", s.deleteOrphan)

	// Policy enforcement (CI/CD)
	api.POST("/v1/policy/check", s.policyCheck)

	// API documentation (no auth required)
	s.echo.GET("/docs", s.apiDocs)
	s.echo.GET("/openapi.yaml", s.openAPISpec)

	// IDE API endpoints
	ide := s.echo.Group("/ide")
	ide.GET("/health", s.ideHealth)
	ide.POST("/validate/file", s.validateFile)
	ide.POST("/validate/selection", s.validateSelection)
	ide.GET("/rules", s.getIDERules)
	ide.GET("/quick-reference", s.getQuickReference)

	// Static files (Web UI) - ORDER MATTERS: specific routes first
	if s.cfg.WebEnabled {
		// SPA fallback: serve index.html for non-file routes (client-side routing)
		// This handles /web, /web/dashboard, /web/rules etc but NOT /web/js/app.js
		// MUST be registered BEFORE the Static("/", ...) route
		s.echo.GET("/web/*", func(c echo.Context) error {
			// Check if this is a request for an actual file
			requestPath := c.Request().URL.Path

			// If the path has a file extension, try to serve it directly first
			if strings.Contains(requestPath, ".") {
				filePath := "/app/web" + strings.TrimPrefix(requestPath, "/web")
				if _, err := os.Stat(filePath); err == nil {
					return c.File(filePath)
				}
			}

			// For routes without file extension (client-side routes), serve index.html
			return c.File("/app/web/index.html")
		})

		// Static file serving - register AFTER specific routes
		// This serves files at root paths like /index.html, /js/app.js, etc.
		s.echo.Static("/web", "/app/web")
		s.echo.Static("/", "/app/web")
	}
}

// Start starts the server
func (s *Server) Start(addr string) error {
	return s.echo.Start(addr)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

// correlationIDMiddleware extracts or generates correlation ID for request tracing
func correlationIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()

			// Check for existing correlation ID from upstream
			correlationID := req.Header.Get("X-Correlation-ID")
			if correlationID == "" {
				// Generate new correlation ID using request ID
				correlationID = res.Header().Get(echo.HeaderXRequestID)
			}

			// Set correlation ID in response header for downstream tracing
			res.Header().Set("X-Correlation-ID", correlationID)

			// Store in context for use in handlers and logging
			c.Set("correlation_id", correlationID)

			// Add to request context for propagation to downstream services
			ctx := context.WithValue(req.Context(), "correlation_id", correlationID)
			c.SetRequest(req.WithContext(ctx))

			return next(c)
		}
	}
}

// panicRecoveryMiddleware recovers from panics and records metrics
func panicRecoveryMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = echo.NewHTTPError(http.StatusInternalServerError, r)
					}

					// Record panic metric
					metricsMiddleware.RecordPanic(c.Path())

					// Log panic with stack trace
					slog.Error("Panic recovered",
						"error", err,
						"path", c.Path(),
						"method", c.Request().Method,
						"correlation_id", c.Get("correlation_id"),
						"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
						"stack", string(debug.Stack()),
					)

					// Return 500 error
					c.Error(err)
				}
			}()
			return next(c)
		}
	}
}

// forwardedHeaderAnomalyMiddleware records a protocol anomaly when a request
// from an untrusted peer asserts a forwarded header (Spec 12 §3.4.4). It does
// not reject: misconfiguration stays visible without breaking traffic, and the
// effective client IP is already the socket peer via the IPExtractor.
func forwardedHeaderAnomalyMiddleware(stance clientip.Stance) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if _, untrustedForwarded := stance.ClientIP(c.Request()); untrustedForwarded {
				slog.Warn("forwarded header from untrusted peer ignored",
					"peer", c.Request().RemoteAddr,
					"path", c.Request().URL.Path,
				)
			}
			return next(c)
		}
	}
}

// securityHeadersMiddleware adds security headers
func securityHeadersMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Content Security Policy
			csp := "default-src 'self'; " +
				"script-src 'self'; " +
				"style-src 'self' 'unsafe-inline'; " +
				"img-src 'self' data:; " +
				"font-src 'self'; " +
				"connect-src 'self'; " +
				"frame-ancestors 'none'; " +
				"base-uri 'self'; " +
				"form-action 'self'"

			c.Response().Header().Set("Content-Security-Policy", csp)
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			// X-XSS-Protection is intentionally NOT set. The legacy XSS auditor
			// it controls is removed in every current browser, and the header
			// is on MDN's deprecated list. Its CSP role (`block`) is superseded
			// by the Content-Security-Policy above. Spec 12-3.6 / audit gap 6.
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			c.Response().Header().Set("Permissions-Policy", "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()")

			return next(c)
		}
	}
}

// cacheControlMiddleware adds cache-control headers to prevent caching of API responses
func cacheControlMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Execute handler first
			err := next(c)

			// Then set cache headers on the response
			path := c.Request().URL.Path
			if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ide/") {
				c.Response().Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
				c.Response().Header().Set("Pragma", "no-cache")
				c.Response().Header().Set("Expires", "0")
			}

			return err
		}
	}
}

// versionInfo returns server version information
func (s *Server) versionInfo(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"version":   s.version,
		"service":   "guardrail-mcp",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Health handlers
func (s *Server) healthLive(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "alive",
		"version":   s.version,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) healthReady(c echo.Context) error {
	// Ordered readiness evaluation (Spec 18 §3 / R18-07..R18-13). The report
	// carries bounded reason codes only — never credentials or raw dependency
	// errors. A non-ready state returns 503 and gates normal protected traffic.
	report := s.readiness.Evaluate(s.Lifecycle())

	code := http.StatusOK
	if !report.Ready {
		code = http.StatusServiceUnavailable
	}
	return c.JSON(code, report)
}
