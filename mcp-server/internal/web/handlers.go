package web

import (
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thearchitectit/guardrail-mcp/internal/auth"
)

// Pagination and validation constants
const (
	defaultPageLimit   = 20
	maxPageLimit       = 100
	maxSearchResults   = 50
	defaultSearchLimit = 20
)

// RuleSyncStatus tracks the status of the last rule sync operation
type RuleSyncStatus struct {
	Status       string    `json:"status"`
	LastSync     time.Time `json:"last_sync"`
	RulesAdded   int       `json:"rules_added"`
	RulesUpdated int       `json:"rules_updated"`
	RulesDeleted int       `json:"rules_deleted"`
	Errors       []string  `json:"errors"`
}

var (
	lastRuleSyncStatus     RuleSyncStatus
	lastRuleSyncStatusLock sync.RWMutex
)

func callerFromContext(c echo.Context) auth.Caller {
	if caller, ok := c.Get("caller").(auth.Caller); ok {
		return caller
	}
	return auth.Caller{}
}
