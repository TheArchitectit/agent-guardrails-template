package metrics

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Namespace for all guardrail metrics
const namespace = "guardrail"

// HTTP metrics
var (
	// HTTPRequestsTotal tracks total HTTP requests
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration tracks HTTP request latency
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request latency in seconds",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestSize tracks HTTP request size
	HTTPRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "request_size_bytes",
			Help:      "HTTP request size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	// HTTPResponseSize tracks HTTP response size
	HTTPResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path", "status"},
	)
)

// MCP tool metrics
var (
	// MCPValidationsTotal tracks MCP tool validations
	MCPValidationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "mcp",
			Name:      "validations_total",
			Help:      "Total number of MCP validation requests",
		},
		[]string{"tool", "result"},
	)

	// MCPValidationDuration tracks validation latency
	MCPValidationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "mcp",
			Name:      "validation_duration_seconds",
			Help:      "MCP validation latency in seconds",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		[]string{"tool"},
	)

	// MCPSessionsActive tracks active sessions
	MCPSessionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "mcp",
			Name:      "sessions_active",
			Help:      "Number of active MCP sessions",
		},
	)

	// MCPSessionsCreatedTotal tracks total sessions created
	MCPSessionsCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "mcp",
			Name:      "sessions_created_total",
			Help:      "Total number of MCP sessions created",
		},
	)

	// MCPSessionsExpiredTotal tracks expired sessions
	MCPSessionsExpiredTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "mcp",
			Name:      "sessions_expired_total",
			Help:      "Total number of MCP sessions expired",
		},
	)
)

// Audit metrics
var (
	// AuditEventsTotal tracks audit events
	AuditEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "audit",
			Name:      "events_total",
			Help:      "Total number of audit events",
		},
		[]string{"type", "severity"},
	)

	// AuditEventsDropped tracks dropped audit events
	AuditEventsDropped = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "audit",
			Name:      "events_dropped_total",
			Help:      "Total number of audit events dropped due to full buffer",
		},
	)
)

// Circuit breaker metrics
var (
	// CircuitBreakerState tracks circuit breaker state
	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "circuitbreaker",
			Name:      "state",
			Help:      "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"name"},
	)

	// CircuitBreakerFailures tracks circuit breaker failures
	CircuitBreakerFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "circuitbreaker",
			Name:      "failures_total",
			Help:      "Total number of circuit breaker failures",
		},
		[]string{"name"},
	)

	// CircuitBreakerSuccesses tracks circuit breaker successes
	CircuitBreakerSuccesses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "circuitbreaker",
			Name:      "successes_total",
			Help:      "Total number of circuit breaker successes",
		},
		[]string{"name"},
	)
)

// Health metrics
var (
	// HealthCheckDuration tracks health check latency
	HealthCheckDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "health",
			Name:      "check_duration_seconds",
			Help:      "Health check latency in seconds",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		[]string{"check"},
	)

	// HealthCheckFailures tracks health check failures
	HealthCheckFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "health",
			Name:      "check_failures_total",
			Help:      "Total number of health check failures",
		},
		[]string{"check"},
	)
)

// Cache metrics
var (
	// CacheHits tracks cache hits
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cache",
			Name:      "hits_total",
			Help:      "Total number of cache hits",
		},
		[]string{"operation"},
	)

	// CacheMisses tracks cache misses
	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cache",
			Name:      "misses_total",
			Help:      "Total number of cache misses",
		},
		[]string{"operation"},
	)

	// CacheErrors tracks cache errors
	CacheErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cache",
			Name:      "errors_total",
			Help:      "Total number of cache errors",
		},
		[]string{"operation"},
	)
)

// Rate limit metrics
var (
	// RateLimitHits tracks rate limit enforcement
	RateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "ratelimit",
			Name:      "hits_total",
			Help:      "Total number of rate limit enforcements",
		},
		[]string{"key_type", "path"},
	)

	// RateLimitAllowed tracks allowed requests
	RateLimitAllowed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "ratelimit",
			Name:      "allowed_total",
			Help:      "Total number of allowed requests",
		},
		[]string{"key_type"},
	)
)

// PrometheusMiddleware returns Echo middleware for Prometheus metrics
func PrometheusMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Capture request info
			req := c.Request()
			res := c.Response()

			 // Get content length if available
			requestSize := req.ContentLength
			if requestSize < 0 {
				requestSize = 0
			}

			// Execute handler
			err := next(c)

			// Capture response info after handler
			duration := time.Since(start).Seconds()
			status := strconv.Itoa(res.Status)
			path := c.Path()
			method := req.Method

			// Record metrics
			HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
			HTTPRequestDuration.WithLabelValues(method, path, status).Observe(duration)
			HTTPRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))
			HTTPResponseSize.WithLabelValues(method, path, status).Observe(float64(res.Size))

			return err
		}
	}
}

// RecordValidation records MCP validation metrics
func RecordValidation(tool string, result string, duration time.Duration) {
	MCPValidationsTotal.WithLabelValues(tool, result).Inc()
	MCPValidationDuration.WithLabelValues(tool).Observe(duration.Seconds())
}

// RecordAuditEvent records audit event metrics
func RecordAuditEvent(eventType string, severity string) {
	AuditEventsTotal.WithLabelValues(eventType, severity).Inc()
}

// RecordAuditDrop records dropped audit event
func RecordAuditDrop() {
	AuditEventsDropped.Inc()
}

// RecordCircuitBreakerState updates circuit breaker state gauge
func RecordCircuitBreakerState(name string, state string) {
	var stateValue float64
	switch state {
	case "closed":
		stateValue = 0
	case "open":
		stateValue = 1
	case "half-open":
		stateValue = 2
	}
	CircuitBreakerState.WithLabelValues(name).Set(stateValue)
}

// RecordCircuitBreakerFailure records a circuit breaker failure
func RecordCircuitBreakerFailure(name string) {
	CircuitBreakerFailures.WithLabelValues(name).Inc()
}

// RecordCircuitBreakerSuccess records a circuit breaker success
func RecordCircuitBreakerSuccess(name string) {
	CircuitBreakerSuccesses.WithLabelValues(name).Inc()
}

// RecordHealthCheck records health check metrics
func RecordHealthCheck(check string, duration time.Duration, failed bool) {
	HealthCheckDuration.WithLabelValues(check).Observe(duration.Seconds())
	if failed {
		HealthCheckFailures.WithLabelValues(check).Inc()
	}
}

// RecordCacheHit records a cache hit
func RecordCacheHit(operation string) {
	CacheHits.WithLabelValues(operation).Inc()
}

// RecordCacheMiss records a cache miss
func RecordCacheMiss(operation string) {
	CacheMisses.WithLabelValues(operation).Inc()
}

// RecordCacheError records a cache error
func RecordCacheError(operation string) {
	CacheErrors.WithLabelValues(operation).Inc()
}

// RecordRateLimitHit records a rate limit enforcement
func RecordRateLimitHit(keyType string, path string) {
	RateLimitHits.WithLabelValues(keyType, path).Inc()
}

// RecordRateLimitAllowed records an allowed request
func RecordRateLimitAllowed(keyType string) {
	RateLimitAllowed.WithLabelValues(keyType).Inc()
}

// IncrementActiveSessions increments active session count
func IncrementActiveSessions() {
	MCPSessionsActive.Inc()
	MCPSessionsCreatedTotal.Inc()
}

// DecrementActiveSessions decrements active session count
func DecrementActiveSessions() {
	MCPSessionsActive.Dec()
}

// RecordSessionExpired records a session expiration
func RecordSessionExpired() {
	MCPSessionsExpiredTotal.Inc()
}
