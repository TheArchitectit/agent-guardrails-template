package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/database"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// TestVerifyFixesIntact tests the handleVerifyFixesIntact function
func TestVerifyFixesIntact(t *testing.T) {
	// This is a simple test to validate the handler logic
	// In a real scenario, we would mock the database and validation

	// Test basic validation
	ctx := context.Background()

	// Test input validation (missing session_token)
	args := map[string]interface{}{
		"file_path": "test.go",
	}

	// Create a mock server for testing
	s := &MCPServer{
		sessions: make(map[string]*Session),
	}

	// Test that error is returned when session_token is missing
	result, err := s.handleVerifyFixesIntact(ctx, args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil || result.Content == nil {
		t.Fatal("Expected non-nil result with content")
	}

	// Check result format (should be a FixVerificationResult JSON)
	// With missing session_token, it should return error=true
	if !result.IsError {
		t.Error("Expected IsError=true for missing session_token")
	}

	// Test with valid session
	sessionID := "test-session-123"
	s.sessions[sessionID] = &Session{
		ID:           sessionID,
		ProjectSlug:  "test-project",
		AgentType:    "claude-code",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	args2 := map[string]interface{}{
		"session_token": sessionID,
		"file_path":     "test.go",
	}

	// This will fail because we don't have a real database connection
	// but it validates the handler structure
	result2, err2 := s.handleVerifyFixesIntact(ctx, args2)
	if err2 != nil {
		t.Logf("Expected error due to no database: %v", err2)
	}
	if result2 == nil {
		t.Fatal("Expected non-nil result")
	}
}

// TestFixVerificationModel tests the FixVerification model
func TestFixVerificationModel(t *testing.T) {
	// Test fix type validation
	if !models.IsValidFixType("regex") {
		t.Error("Expected regex to be a valid fix type")
	}
	if !models.IsValidFixType("code_change") {
		t.Error("Expected code_change to be a valid fix type")
	}
	if !models.IsValidFixType("config") {
		t.Error("Expected config to be a valid fix type")
	}
	if models.IsValidFixType("invalid") {
		t.Error("Expected invalid to not be a valid fix type")
	}

	// Test verification status validation
	if !models.IsValidVerificationStatus("confirmed") {
		t.Error("Expected confirmed to be a valid verification status")
	}
	if !models.IsValidVerificationStatus("modified") {
		t.Error("Expected modified to be a valid verification status")
	}
	if !models.IsValidVerificationStatus("removed") {
		t.Error("Expected removed to be a valid verification status")
	}
	if models.IsValidVerificationStatus("invalid") {
		t.Error("Expected invalid to not be a valid verification status")
	}
}

// TestComputeFixHash tests the ComputeFixHash function
func TestComputeFixHash(t *testing.T) {
	testContent := "test content"
	hash1 := database.ComputeFixHash(testContent)
	hash2 := database.ComputeFixHash(testContent)

	if hash1 != hash2 {
		t.Error("Expected same content to produce same hash")
	}

	// Different content should produce different hashes
	hash3 := database.ComputeFixHash("different content")
	if hash1 == hash3 {
		t.Error("Expected different content to produce different hash")
	}

	// Same content with different whitespace should produce different hashes
	hash4 := database.ComputeFixHash("test content ")
	if hash1 == hash4 {
		t.Error("Expected different content to produce different hash")
	}
}

// TestFixVerificationResultStructure tests the result structure
func TestFixVerificationResultStructure(t *testing.T) {
	result := models.FixVerificationResult{
		AllFixesIntact: true,
		VerifySummary:  "5/5 fixes verified intact",
		Fixes: []models.IndividualFixResult{
			{
				FailureID:           "test-123",
				Status:              models.StatusConfirmed,
				FixType:             models.FixTypeRegex,
				AffectedFile:        "test.go",
				VerificationMessage: "Fix still intact",
			},
		},
		Recommendation: "Continue - all fixes intact",
	}

	if !result.AllFixesIntact {
		t.Error("Expected AllFixesIntact to be true")
	}

	if len(result.Fixes) != 1 {
		t.Error("Expected 1 fix in result")
	}

	if result.Fixes[0].Status != models.StatusConfirmed {
		t.Error("Expected fix status to be confirmed")
	}
}
