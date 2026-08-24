package metrics

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
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

// RecordPanic records a recovered panic
func RecordPanic(path string) {
	PanicsTotal.WithLabelValues(path).Inc()
}

// RecordDBStats records database connection pool statistics
func RecordDBStats(stats struct {
	Open         int
	InUse        int
	Idle         int
	WaitDuration float64
	WaitCount    int64
}) {
	DBConnectionsActive.WithLabelValues("open").Set(float64(stats.Open))
	DBConnectionsActive.WithLabelValues("in_use").Set(float64(stats.InUse))
	DBConnectionsActive.WithLabelValues("idle").Set(float64(stats.Idle))
	DBConnectionsWaitDuration.Add(stats.WaitDuration)
	DBConnectionsWaitCount.Add(float64(stats.WaitCount))
}

// RecordDBQuery records database query duration
func RecordDBQuery(operation, table string, duration time.Duration) {
	DBQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordCacheOperation records cache operation duration
func RecordCacheOperation(operation string, duration time.Duration) {
	CacheOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordSLOCompliance records SLO compliance status
func RecordSLOCompliance(sloName string, compliant bool) {
	value := 0.0
	if compliant {
		value = 1.0
	}
	SLOCompliance.WithLabelValues(sloName).Set(value)
}

// RecordErrorBudgetBurnRate records error budget burn rate
func RecordErrorBudgetBurnRate(sloName, window string, rate float64) {
	ErrorBudgetBurnRate.WithLabelValues(sloName, window).Set(rate)
}

// RecordSLI records SLI value
func RecordSLI(sloName string, value float64) {
	SLIValue.WithLabelValues(sloName).Set(value)
}

// RecordTeamToolCall records a team tool call
func RecordTeamToolCall(tool string, success bool) {
	result := "success"
	if !success {
		result = "error"
	}
	TeamToolCallsTotal.WithLabelValues(tool, result).Inc()
}

// RecordTeamToolDuration records team tool execution duration
func RecordTeamToolDuration(tool string, duration time.Duration) {
	TeamToolDuration.WithLabelValues(tool).Observe(duration.Seconds())
}

// RecordTeamToolError records a team tool error
func RecordTeamToolError(tool string, errorType string) {
	TeamToolErrorsTotal.WithLabelValues(tool, errorType).Inc()
}

// IncrementTeamToolActive increments active team tool operations
func IncrementTeamToolActive(tool string) {
	TeamToolActiveOperations.WithLabelValues(tool).Inc()
}

// DecrementTeamToolActive decrements active team tool operations
func DecrementTeamToolActive(tool string) {
	TeamToolActiveOperations.WithLabelValues(tool).Dec()
}

// RecordTeamToolPythonExec records Python script execution duration
func RecordTeamToolPythonExec(command string, duration time.Duration) {
	TeamToolPythonExecDuration.WithLabelValues(command).Observe(duration.Seconds())
}

// RecordPerformanceOperation records a performance operation metric (OPS-008)
func RecordPerformanceOperation(operation string, duration time.Duration, success bool) {
	result := "success"
	if !success {
		result = "error"
	}
	PerformanceOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
	PerformanceOperationTotal.WithLabelValues(operation, result).Inc()
}

// RecordPerformanceOperationError records a performance operation error (OPS-008)
func RecordPerformanceOperationError(operation string, errorType string) {
	PerformanceOperationErrors.WithLabelValues(operation, errorType).Inc()
}
