package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// TeamToolMetrics provides instrumentation for team tool handlers
type TeamToolMetrics struct {
	tool  string
	start time.Time
}

// NewTeamToolMetrics creates a new team tool metrics recorder
func NewTeamToolMetrics(tool string) *TeamToolMetrics {
	IncrementTeamToolActive(tool)
	return &TeamToolMetrics{
		tool:  tool,
		start: time.Now(),
	}
}

// Done records the completion of a team tool operation
func (m *TeamToolMetrics) Done(success bool) {
	DecrementTeamToolActive(m.tool)
	RecordTeamToolDuration(m.tool, time.Since(m.start))
	RecordTeamToolCall(m.tool, success)
}

// RecordError records an error for the team tool operation
func (m *TeamToolMetrics) RecordError(errorType string) {
	RecordTeamToolError(m.tool, errorType)
}

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

	// CacheOperationDuration tracks cache operation latency
	CacheOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "cache",
			Name:      "operation_duration_seconds",
			Help:      "Cache operation latency in seconds",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
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

// Panic recovery metrics
var (
	// PanicsTotal tracks recovered panics
	PanicsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "runtime",
			Name:      "panics_total",
			Help:      "Total number of recovered panics",
		},
		[]string{"path"},
	)
)

// Database metrics
var (
	// DBConnectionsActive tracks active database connections
	DBConnectionsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "database",
			Name:      "connections_active",
			Help:      "Current number of active database connections",
		},
		[]string{"state"}, // state: open, in_use, idle
	)

	// DBConnectionsWaitDuration tracks time waiting for connection
	DBConnectionsWaitDuration = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "database",
			Name:      "connections_wait_duration_seconds_total",
			Help:      "Total time waited for database connections",
		},
	)

	// DBConnectionsWaitCount tracks number of waits for connection
	DBConnectionsWaitCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "database",
			Name:      "connections_wait_count_total",
			Help:      "Total number of waits for database connections",
		},
	)

	// DBQueryDuration tracks database query latency
	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "database",
			Name:      "query_duration_seconds",
			Help:      "Database query latency in seconds",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"operation", "table"},
	)
)

// SLO/Error budget metrics
var (
	// SLOCompliance tracks SLO compliance (1 = compliant, 0 = breached)
	SLOCompliance = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "slo",
			Name:      "compliance",
			Help:      "SLO compliance status (1 = compliant, 0 = breached)",
		},
		[]string{"slo_name"},
	)

	// ErrorBudgetBurnRate tracks error budget burn rate
	ErrorBudgetBurnRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "slo",
			Name:      "error_budget_burn_rate",
			Help:      "Error budget burn rate (1.0 = on track, >1 = burning too fast)",
		},
		[]string{"slo_name", "window"},
	)

	// SLIValue tracks SLI values
	SLIValue = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "slo",
			Name:      "sli_value",
			Help:      "SLI value (0-1 scale)",
		},
		[]string{"slo_name"},
	)
)

// Team tool metrics
var (
	// TeamToolCallsTotal tracks total team tool calls
	TeamToolCallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "team_tool",
			Name:      "calls_total",
			Help:      "Total number of team tool calls",
		},
		[]string{"tool", "result"},
	)

	// TeamToolDuration tracks team tool execution latency
	TeamToolDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "team_tool",
			Name:      "duration_seconds",
			Help:      "Team tool execution latency in seconds",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"tool"},
	)

	// TeamToolErrorsTotal tracks team tool errors
	TeamToolErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "team_tool",
			Name:      "errors_total",
			Help:      "Total number of team tool errors",
		},
		[]string{"tool", "error_type"},
	)

	// TeamToolActiveOperations tracks active team tool operations
	TeamToolActiveOperations = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "team_tool",
			Name:      "active_operations",
			Help:      "Number of active team tool operations",
		},
		[]string{"tool"},
	)

	// TeamToolPythonExecDuration tracks Python script execution time
	TeamToolPythonExecDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "team_tool",
			Name:      "python_exec_duration_seconds",
			Help:      "Python script execution latency in seconds",
			Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"command"},
	)
)

// Performance operation metrics (OPS-008)
var (
	// PerformanceOperationDuration tracks operation latency
	PerformanceOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "performance",
			Name:      "operation_duration_seconds",
			Help:      "Performance operation latency in seconds",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"operation"},
	)

	// PerformanceOperationTotal tracks total operations
	PerformanceOperationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "performance",
			Name:      "operations_total",
			Help:      "Total number of operations",
		},
		[]string{"operation", "result"},
	)

	// PerformanceOperationErrors tracks operation errors
	PerformanceOperationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "performance",
			Name:      "operation_errors_total",
			Help:      "Total number of operation errors",
		},
		[]string{"operation", "error_type"},
	)
)
