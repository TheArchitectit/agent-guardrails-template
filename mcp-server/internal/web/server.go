package web

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thearchitectit/guardrail-mcp/internal/audit"
	"github.com/thearchitectit/guardrail-mcp/internal/cache"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
)

// Server wraps the Echo server with guardrail dependencies
type Server struct {
	echo        *echo.Echo
	cfg         *config.Config
	db          *database.DB
	cache       *cache.Client
	auditLogger *audit.Logger
	docStore    *database.DocumentStore
	ruleStore   *database.RuleStore
	projStore   *database.ProjectStore
	failStore   *database.FailureStore
	version     string
}

// NewServer creates a new web server
func NewServer(cfg *config.Config, db *database.DB, cache *cache.Client, auditLogger *audit.Logger, version string) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	s := &Server{
		echo:        e,
		cfg:         cfg,
		db:          db,
		cache:       cache,
		auditLogger: auditLogger,
		docStore:    database.NewDocumentStore(db),
		ruleStore:   database.NewRuleStore(db),
		projStore:   database.NewProjectStore(db),
		failStore:   database.NewFailureStore(db),
		version:     version,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware configures Echo middleware
func (s *Server) setupMiddleware() {
	// Request ID generation
	s.echo.Use(middleware.RequestID())

	// Recovery from panics
	s.echo.Use(middleware.Recover())

	// Security headers
	s.echo.Use(securityHeadersMiddleware())

	// API Key Authentication (required for all routes except health/metrics)
	s.echo.Use(APIKeyAuth(s.cfg))

	// Rate Limiting
	limiter := s.cache.NewDistributedLimiter()
	s.echo.Use(RateLimitMiddleware(limiter, s.cfg))

	// CORS - restrict in production
	corsOrigins := []string{"http://localhost:*", "https://localhost:*"}
	if s.cfg.DBSSLMode == "require" {
		// In production, be more restrictive
		corsOrigins = []string{"http://localhost:8081", "https://localhost:8081"}
	}
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: corsOrigins,
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// Request timeout
	s.echo.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: s.cfg.RequestTimeout,
	}))

	// Body limit
	s.echo.Use(middleware.BodyLimit("10M"))
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

	// Document routes
	api.GET("/documents", s.listDocuments)
	api.GET("/documents/:id", s.getDocument)
	api.PUT("/documents/:id", s.updateDocument)
	api.GET("/documents/search", s.searchDocuments)

	// Rule routes
	api.GET("/rules", s.listRules)
	api.GET("/rules/:id", s.getRule)
	api.POST("/rules", s.createRule)
	api.PUT("/rules/:id", s.updateRule)
	api.DELETE("/rules/:id", s.deleteRule)
	api.POST("/rules/:id/toggle", s.toggleRule)

	// Project routes
	api.GET("/projects", s.listProjects)
	api.GET("/projects/:slug", s.getProject)
	api.POST("/projects", s.createProject)
	api.PUT("/projects/:slug", s.updateProject)
	api.DELETE("/projects/:slug", s.deleteProject)

	// Failure registry routes
	api.GET("/failures", s.listFailures)
	api.GET("/failures/:id", s.getFailure)
	api.POST("/failures", s.createFailure)
	api.PUT("/failures/:id", s.updateFailure)

	// System routes
	api.GET("/stats", s.getStats)
	api.POST("/ingest", s.triggerIngest)

	// IDE API endpoints
	ide := s.echo.Group("/ide")
	ide.GET("/health", s.ideHealth)
	ide.POST("/validate/file", s.validateFile)
	ide.POST("/validate/selection", s.validateSelection)
	ide.GET("/rules", s.getIDERules)
	ide.GET("/quick-reference", s.getQuickReference)

	// Static files (Web UI)
	if s.cfg.WebEnabled {
		s.echo.Static("/", "static")
		s.echo.File("/", "static/index.html")
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
			c.Response().Header().Set("X-XSS-Protection", "1; mode=block")
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			c.Response().Header().Set("Permissions-Policy", "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()")

			return next(c)
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
	ctx, cancel := context.WithTimeout(c.Request().Context(), s.cfg.HealthCheckTimeout)
	defer cancel()

	// Check database
	if err := s.db.HealthCheck(ctx); err != nil {
		slog.Error("Readiness check failed - database", "error", err)
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status":    "not ready",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			// Don't expose which component failed for security
		})
	}

	// Check cache
	if err := s.cache.HealthCheck(ctx); err != nil {
		slog.Error("Readiness check failed - cache", "error", err)
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status":    "not ready",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			// Don't expose which component failed for security
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "ready",
		"version":   s.version,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
