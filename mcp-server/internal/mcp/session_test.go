package mcp

import (
	"testing"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// TestSessionLifecycle covers the token registry: a token issued by
// guardrail_init_session must be accepted by the tools that validate
// session_token, and an expired token must not be.
func TestSessionLifecycle(t *testing.T) {
	s := &MCPServer{sessions: make(map[string]*models.Session)}

	t.Run("registered session is valid", func(t *testing.T) {
		s.registerSession(&models.Session{
			Token:     "live-token",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		})

		if !s.sessionValid("live-token") {
			t.Fatal("a freshly registered session must be valid")
		}
		if _, ok := s.lookupSession("live-token"); !ok {
			t.Fatal("lookupSession must return a registered session")
		}
	})

	t.Run("unknown session is invalid", func(t *testing.T) {
		if s.sessionValid("never-issued") {
			t.Fatal("an unregistered token must be invalid")
		}
	})

	t.Run("expired session is invalid", func(t *testing.T) {
		s.registerSession(&models.Session{
			Token:     "stale-token",
			CreatedAt: time.Now().Add(-2 * time.Hour),
			ExpiresAt: time.Now().Add(-time.Hour),
		})

		if s.sessionValid("stale-token") {
			t.Fatal("an expired token must not be valid")
		}
	})

	t.Run("session with no expiry does not expire", func(t *testing.T) {
		s.registerSession(&models.Session{Token: "no-expiry", CreatedAt: time.Now()})

		if !s.sessionValid("no-expiry") {
			t.Fatal("a session without ExpiresAt must remain valid")
		}
	})

	t.Run("registering prunes expired sessions", func(t *testing.T) {
		// Seed the expired entry directly: registerSession prunes as a side
		// effect, so it cannot be used to establish this precondition.
		s.sessionsMu.Lock()
		s.sessions["stale-token"] = &models.Session{
			Token:     "stale-token",
			CreatedAt: time.Now().Add(-2 * time.Hour),
			ExpiresAt: time.Now().Add(-time.Hour),
		}
		s.sessionsMu.Unlock()

		s.registerSession(&models.Session{
			Token:     "trigger-prune",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		})

		if _, ok := s.sessions["stale-token"]; ok {
			t.Fatal("expired sessions should be evicted on registration")
		}
		if !s.sessionValid("trigger-prune") {
			t.Fatal("the newly registered session must survive pruning")
		}
	})
}