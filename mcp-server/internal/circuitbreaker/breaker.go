package circuitbreaker

import (
	"time"

	"github.com/sony/gobreaker"
)

// Database circuit breaker
var DBBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
	Name:        "database",
	MaxRequests: 3,                // Half-open state probe count
	Interval:    10 * time.Second, // Statistical window
	Timeout:     30 * time.Second, // Request timeout
	ReadyToTrip: func(counts gobreaker.Counts) bool {
		failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
		return counts.Requests >= 3 && failureRatio >= 0.6
	},
})

// Redis circuit breaker
var RedisBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
	Name:        "redis",
	MaxRequests: 3,
	Interval:    10 * time.Second,
	Timeout:     5 * time.Second,
	ReadyToTrip: func(counts gobreaker.Counts) bool {
		failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
		return counts.Requests >= 3 && failureRatio >= 0.6
	},
})

// State returns the current state of the circuit breaker
func State(breaker *gobreaker.CircuitBreaker) string {
	state := breaker.State()
	switch state {
	case gobreaker.StateClosed:
		return "closed"
	case gobreaker.StateOpen:
		return "open"
	case gobreaker.StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}
