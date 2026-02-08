package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/thearchitectit/guardrail-mcp/internal/audit"
	"github.com/thearchitectit/guardrail-mcp/internal/cache"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
)

// MCPServer wraps the MCP server with guardrail dependencies
type MCPServer struct {
	echo        *echo.Echo
	cfg         *config.Config
	db          *database.DB
	cache       *cache.Client
	auditLogger *audit.Logger
	mcpServer   server.MCPServer
	sessions    map[string]*Session
	sessionsMu  sync.RWMutex
}

// Session represents an MCP client session
type Session struct {
	ID            string
	ProjectSlug   string
	AgentType     string
	ClientVersion string
	CreatedAt     time.Time
	LastActivity  time.Time
}

// NewMCPServer creates a new MCP server
func NewMCPServer(cfg *config.Config, db *database.DB, cacheClient *cache.Client, auditLogger *audit.Logger) *MCPServer {
	s := &MCPServer{
		cfg:         cfg,
		db:          db,
		cache:       cacheClient,
		auditLogger: auditLogger,
		sessions:    make(map[string]*Session),
	}

	// Create MCP server using the default server
	s.mcpServer = server.NewDefaultServer("guardrail-mcp", "1.0.0")

	// Register tool handlers
	s.registerTools()

	return s
}

// registerTools registers all MCP tool handlers
func (s *MCPServer) registerTools() {
	// Handle tool list requests
	s.mcpServer.HandleListTools(func(ctx context.Context, cursor *string) (*mcp.ListToolsResult, error) {
		return &mcp.ListToolsResult{
			Tools: []mcp.Tool{
				{
					Name:        "guardrail_init_session",
					Description: "Initialize a validation session for a project",
					InputSchema: mcp.ToolInputSchema{
						Type: "object",
						Properties: mcp.ToolInputSchemaProperties{
							"project_slug": map[string]interface{}{
								"type":        "string",
								"description": "Project identifier",
							},
							"agent_type": map[string]interface{}{
								"type":        "string",
								"description": "Agent type (claude-code, opencode, cursor)",
							},
							"client_version": map[string]interface{}{
								"type":        "string",
								"description": "Client version",
							},
						},
					},
				},
				{
					Name:        "guardrail_validate_bash",
					Description: "Validate bash command against forbidden patterns",
					InputSchema: mcp.ToolInputSchema{
						Type: "object",
						Properties: mcp.ToolInputSchemaProperties{
							"session_token": map[string]interface{}{
								"type":        "string",
								"description": "Session token from init_session",
							},
							"command": map[string]interface{}{
								"type":        "string",
								"description": "Bash command to validate",
							},
							"working_directory": map[string]interface{}{
								"type":        "string",
								"description": "Current working directory",
							},
						},
					},
				},
				{
					Name:        "guardrail_validate_file_edit",
					Description: "Validate file edit operation",
					InputSchema: mcp.ToolInputSchema{
						Type: "object",
						Properties: mcp.ToolInputSchemaProperties{
							"session_token": map[string]interface{}{
								"type":        "string",
								"description": "Session token",
							},
							"file_path": map[string]interface{}{
								"type":        "string",
								"description": "File path",
							},
							"old_string": map[string]interface{}{
								"type":        "string",
								"description": "Original string",
							},
							"new_string": map[string]interface{}{
								"type":        "string",
								"description": "New string",
							},
						},
					},
				},
				{
					Name:        "guardrail_validate_git_operation",
					Description: "Validate git command against guardrails",
					InputSchema: mcp.ToolInputSchema{
						Type: "object",
						Properties: mcp.ToolInputSchemaProperties{
							"session_token": map[string]interface{}{
								"type":        "string",
								"description": "Session token",
							},
							"command": map[string]interface{}{
								"type":        "string",
								"description": "Git command (push, commit, merge, rebase, reset)",
							},
							"is_force": map[string]interface{}{
								"type":        "boolean",
								"description": "Whether this is a force operation",
							},
						},
					},
				},
				{
					Name:        "guardrail_pre_work_check",
					Description: "Run pre-work checklist from failure registry",
					InputSchema: mcp.ToolInputSchema{
						Type: "object",
						Properties: mcp.ToolInputSchemaProperties{
							"session_token": map[string]interface{}{
								"type":        "string",
								"description": "Session token",
							},
							"affected_files": map[string]interface{}{
								"type":        "array",
								"description": "Files that will be modified",
							},
						},
					},
				},
				{
					Name:        "guardrail_get_context",
					Description: "Get guardrail context for the session's project",
					InputSchema: mcp.ToolInputSchema{
						Type: "object",
						Properties: mcp.ToolInputSchemaProperties{
							"session_token": map[string]interface{}{
								"type":        "string",
								"description": "Session token",
							},
						},
					},
				},
			},
		}, nil
	})

	// Handle tool calls
	s.mcpServer.HandleCallTool(s.handleToolCall)

	// Handle resource list requests
	s.mcpServer.HandleListResources(func(ctx context.Context, cursor *string) (*mcp.ListResourcesResult, error) {
		return &mcp.ListResourcesResult{
			Resources: []mcp.Resource{
				{
					Uri:         "guardrail://quick-reference",
					Name:        "Quick Reference",
					Description: "Quick reference card for guardrails",
					MimeType:    "application/json",
				},
				{
					Uri:         "guardrail://rules/active",
					Name:        "Active Prevention Rules",
					Description: "Currently active prevention rules",
					MimeType:    "application/json",
				},
			},
		}, nil
	})

	// Handle resource read requests
	s.mcpServer.HandleReadResource(s.handleReadResource)
}

// handleToolCall handles incoming tool calls
func (s *MCPServer) handleToolCall(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	switch name {
	case "guardrail_init_session":
		return s.handleInitSession(ctx, arguments)
	case "guardrail_validate_bash":
		return s.handleValidateBash(ctx, arguments)
	case "guardrail_validate_file_edit":
		return s.handleValidateFileEdit(ctx, arguments)
	case "guardrail_validate_git_operation":
		return s.handleValidateGit(ctx, arguments)
	case "guardrail_pre_work_check":
		return s.handlePreWorkCheck(ctx, arguments)
	case "guardrail_get_context":
		return s.handleGetContext(ctx, arguments)
	default:
		return &mcp.CallToolResult{
			Content: []interface{}{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Unknown tool: %s", name),
				},
			},
			IsError: true,
		}, nil
	}
}

// handleReadResource handles resource read requests
func (s *MCPServer) handleReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	switch uri {
	case "guardrail://quick-reference":
		content := map[string]interface{}{
			"forbidden_commands": []string{
				"rm -rf /",
				"git push --force",
				"git reset --hard",
			},
			"required_checks": []string{
				"pre_work_check",
				"validate_file_edit",
			},
		}
		contentJSON, _ := json.MarshalIndent(content, "", "  ")
		return &mcp.ReadResourceResult{
			Contents: []interface{}{
				mcp.TextResourceContents{
					Uri:      uri,
					MimeType: "application/json",
					Text:     string(contentJSON),
				},
			},
		}, nil

	case "guardrail://rules/active":
		ruleStore := database.NewRuleStore(s.db)
		rules, err := ruleStore.GetActiveRules(ctx)
		if err != nil {
			return nil, err
		}
		rulesJSON, _ := json.MarshalIndent(rules, "", "  ")
		return &mcp.ReadResourceResult{
			Contents: []interface{}{
				mcp.TextResourceContents{
					Uri:      uri,
					MimeType: "application/json",
					Text:     string(rulesJSON),
				},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown resource: %s", uri)
	}
}

// Start starts the MCP server
func (s *MCPServer) Start(addr string) error {
	s.echo = echo.New()
	s.echo.HideBanner = true
	s.echo.HidePort = true

	// Recovery from panics
	s.echo.Use(middleware.Recover())

	// Security headers middleware
	s.echo.Use(s.securityHeadersMiddleware())

	// Request timeout
	s.echo.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: s.cfg.RequestTimeout,
	}))

	// Body limit - prevent DoS via large payloads
	s.echo.Use(middleware.BodyLimit("1M"))

	// SSE endpoint
	s.echo.GET("/mcp/v1/sse", s.handleSSE)

	// Message endpoint
	s.echo.POST("/mcp/v1/message", s.handleMessage)

	slog.Info("Starting MCP SSE server", "addr", addr)
	return s.echo.Start(addr)
}

// securityHeadersMiddleware adds security headers to all responses
func (s *MCPServer) securityHeadersMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			c.Response().Header().Set("X-XSS-Protection", "1; mode=block")
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			return next(c)
		}
	}
}

// Shutdown gracefully shuts down the server
func (s *MCPServer) Shutdown(ctx context.Context) error {
	if s.echo != nil {
		return s.echo.Shutdown(ctx)
	}
	return nil
}

// handleSSE handles SSE connections
func (s *MCPServer) handleSSE(c echo.Context) error {
	// Validate origin for CORS - only allow specific origins
	origin := c.Request().Header.Get("Origin")
	allowedOrigins := []string{"http://localhost:*", "https://localhost:*"}
	if s.cfg.DBSSLMode == "require" {
		// In production, be more restrictive
		allowedOrigins = []string{"http://localhost:8081", "https://localhost:8081"}
	}

	// Check if origin is allowed
	originAllowed := false
	for _, allowed := range allowedOrigins {
		if strings.HasSuffix(allowed, ":*") {
			// Handle wildcard port matching
			prefix := strings.TrimSuffix(allowed, ":*")
			if strings.HasPrefix(origin, prefix) {
				originAllowed = true
				break
			}
		} else if origin == allowed || allowed == "*" {
			originAllowed = true
			break
		}
	}

	// Set SSE headers
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no") // Disable Nginx buffering

	// Only set CORS headers if origin is allowed
	if originAllowed && origin != "" {
		c.Response().Header().Set("Access-Control-Allow-Origin", origin)
		c.Response().Header().Set("Access-Control-Allow-Methods", "GET")
		c.Response().Header().Set("Vary", "Origin")
	}

	c.Response().WriteHeader(http.StatusOK)

	// Generate cryptographically secure session ID
	sessionID := generateSessionID()

	// Build full message endpoint URL with session ID per MCP 2024-11-05 spec
	// The endpoint must be an absolute URL that client uses to POST messages
	scheme := "http"
	if c.Request().TLS != nil {
		scheme = "https"
	}
	host := c.Request().Host
	messageEndpoint := fmt.Sprintf("%s://%s/mcp/v1/message?session_id=%s", scheme, host, sessionID)

	slog.Debug("SSE connection established", "session_id", sessionID)

	// Send endpoint event with full URI per MCP spec
	// Format: event: endpoint\ndata: <absolute_url>\n\n
	if _, err := fmt.Fprintf(c.Response(), "event: endpoint\ndata: %s\n\n", messageEndpoint); err != nil {
		slog.Warn("SSE endpoint write failed", "session_id", sessionID, "error", err)
		return nil
	}
	c.Response().Flush()

	// Track connection state
	clientGone := c.Request().Context().Done()

	// Send initial ping with proper JSON-RPC notification format per MCP spec
	pingData := `{"jsonrpc":"2.0","method":"ping"}`
	if _, err := fmt.Fprintf(c.Response(), "event: ping\ndata: %s\n\n", pingData); err != nil {
		slog.Warn("SSE initial ping write failed", "session_id", sessionID, "error", err)
		return nil
	}
	c.Response().Flush()

	// Keep connection open with periodic pings (every 30 seconds)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pingData := `{"jsonrpc":"2.0","method":"ping"}`
			if _, err := fmt.Fprintf(c.Response(), "event: ping\ndata: %s\n\n", pingData); err != nil {
				slog.Debug("SSE write failed, client disconnected", "session_id", sessionID, "error", err)
				return nil
			}
			c.Response().Flush()
		case <-clientGone:
			slog.Debug("SSE client disconnected", "session_id", sessionID)
			return nil
		}
	}
}

// handleMessage handles incoming JSON-RPC messages per MCP specification
func (s *MCPServer) handleMessage(c echo.Context) error {
	// Extract session ID from query parameter per MCP spec
	sessionID := c.QueryParam("session_id")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, server.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{
				Code:    -32000,
				Message: "Missing session_id parameter: session_id query parameter is required",
			},
		})
	}

	// Validate session exists (sessions are created during SSE connection)
	s.sessionsMu.RLock()
	_, sessionExists := s.sessions[sessionID]
	s.sessionsMu.RUnlock()

	if !sessionExists {
		// Session may have expired or invalid session ID
		slog.Warn("Message received for invalid/expired session", "session_id", sessionID)
		// Still process the request as session validation is handled by individual tools
	}

	var request server.JSONRPCRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, server.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{
				Code:    -32700,
				Message: "Parse error: " + err.Error(),
			},
		})
	}

	// Validate JSON-RPC version
	if request.JSONRPC != "2.0" {
		return c.JSON(http.StatusBadRequest, server.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{
				Code:    -32600,
				Message: "Invalid Request: jsonrpc field must be '2.0'",
			},
		})
	}

	// Create context with session ID for tool handlers
	ctx := context.WithValue(c.Request().Context(), "session_id", sessionID)

	// Process request through MCP server
	response := s.mcpServer.Request(ctx, request)

	// Update session last activity if session exists
	if sessionExists {
		s.sessionsMu.Lock()
		if sess, ok := s.sessions[sessionID]; ok {
			sess.LastActivity = time.Now()
		}
		s.sessionsMu.Unlock()
	}

	return c.JSON(http.StatusOK, response)
}

// Tool handlers

func (s *MCPServer) handleInitSession(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	projectSlug, _ := args["project_slug"].(string)
	agentType, _ := args["agent_type"].(string)
	clientVersion, _ := args["client_version"].(string)

	if projectSlug == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "project_slug is required"}},
			IsError: true,
		}, nil
	}

	// Create session
	sessionID := generateSessionID()
	session := &Session{
		ID:            sessionID,
		ProjectSlug:   projectSlug,
		AgentType:     agentType,
		ClientVersion: clientVersion,
		CreatedAt:     time.Now(),
		LastActivity:  time.Now(),
	}

	s.sessionsMu.Lock()
	s.sessions[sessionID] = session
	s.sessionsMu.Unlock()

	// Audit log
	s.auditLogger.LogSession(ctx, audit.EventSessionCreated, sessionID, projectSlug)

	// Get project context
	projStore := database.NewProjectStore(s.db)
	proj, _ := projStore.GetBySlug(ctx, projectSlug)

	contextStr := ""
	if proj != nil {
		contextStr = proj.GuardrailContext
	}

	// Get active rules count
	ruleStore := database.NewRuleStore(s.db)
	rules, _ := ruleStore.GetActiveRules(ctx)

	result := map[string]interface{}{
		"session_token":      sessionID,
		"expires_at":         time.Now().Add(s.cfg.JWTExpiry).Format(time.RFC3339),
		"project_context":    contextStr,
		"active_rules_count": len(rules),
		"capabilities":       []string{"bash_validation", "git_validation", "edit_validation"},
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: string(resultJSON)}},
	}, nil
}

func (s *MCPServer) handleValidateBash(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	command, _ := args["command"].(string)

	// TODO: Implement actual validation against prevention rules
	result := map[string]interface{}{
		"valid":      true,
		"violations": []interface{}{},
		"meta": map[string]interface{}{
			"checked_at":       time.Now().Format(time.RFC3339),
			"rules_evaluated":  0,
			"duration_ms":      0,
			"command_analyzed": command,
		},
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: string(resultJSON)}},
	}, nil
}

func (s *MCPServer) handleValidateFileEdit(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	filePath, _ := args["file_path"].(string)
	newString, _ := args["new_string"].(string)

	// TODO: Implement actual validation
	result := map[string]interface{}{
		"valid":      true,
		"violations": []interface{}{},
		"meta": map[string]interface{}{
			"checked_at":      time.Now().Format(time.RFC3339),
			"rules_evaluated": 0,
			"file":            filePath,
			"changes_size":    len(newString),
		},
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: string(resultJSON)}},
	}, nil
}

func (s *MCPServer) handleValidateGit(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	command, _ := args["command"].(string)
	isForce, _ := args["is_force"].(bool)

	// Check for force push
	if command == "push" && isForce {
		violation := map[string]interface{}{
			"rule_id":               "PREVENT-001",
			"rule_name":             "No Force Push",
			"severity":              "error",
			"message":               "git push --force violates guardrail: NO FORCE PUSH",
			"category":              "git_operation",
			"action":                "halt",
			"suggested_alternative": "Use git push --force-with-lease instead",
		}

		result := map[string]interface{}{
			"valid":      false,
			"violations": []interface{}{violation},
		}
		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: string(resultJSON)}},
		}, nil
	}

	result := map[string]interface{}{
		"valid":      true,
		"violations": []interface{}{},
	}
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: string(resultJSON)}},
	}, nil
}

func (s *MCPServer) handlePreWorkCheck(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	affectedFilesArg, _ := args["affected_files"].([]interface{})

	// Convert to string slice
	files := make([]string, len(affectedFilesArg))
	for i, f := range affectedFilesArg {
		files[i], _ = f.(string)
	}

	// Get active failures for these files
	failStore := database.NewFailureStore(s.db)
	failures, err := failStore.GetActiveByFiles(ctx, files)

	if err != nil {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Failed to check failures: %v", err)}},
			IsError: true,
		}, nil
	}

	// Convert failures to check results
	checks := make([]map[string]interface{}, len(failures))
	for i, f := range failures {
		checks[i] = map[string]interface{}{
			"failure_id":    f.FailureID,
			"severity":      f.Severity,
			"message":       f.ErrorMessage,
			"root_cause":    f.RootCause,
			"affected_file": f.AffectedFiles,
		}
	}

	result := map[string]interface{}{
		"passed":         len(failures) == 0,
		"checks":         checks,
		"files_affected": files,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: string(resultJSON)}},
	}, nil
}

func (s *MCPServer) handleGetContext(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionToken, _ := args["session_token"].(string)

	s.sessionsMu.RLock()
	session, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Invalid session token"}},
			IsError: true,
		}, nil
	}

	// Get project context
	projStore := database.NewProjectStore(s.db)
	proj, err := projStore.GetBySlug(ctx, session.ProjectSlug)

	if err != nil {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: fmt.Sprintf("# Default Guardrails\n\nNo project-specific context found for %s", session.ProjectSlug)}},
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: proj.GuardrailContext}},
	}, nil
}

// generateSessionID creates a cryptographically secure session ID
func generateSessionID() string {
	// Use crypto/rand for secure random generation instead of timestamp
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp only if crypto/rand fails (should never happen)
		slog.Error("Failed to generate secure random session ID, falling back to timestamp", "error", err)
		return fmt.Sprintf("sess_%d", time.Now().UnixNano())
	}
	return "sess_" + hex.EncodeToString(b)
}

// sessionCleanup periodically removes expired sessions to prevent memory leaks
func (s *MCPServer) sessionCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.sessionsMu.Lock()
		now := time.Now()
		expiredCount := 0
		for id, session := range s.sessions {
			// Remove sessions inactive for more than 1 hour
			if now.Sub(session.LastActivity) > time.Hour {
				delete(s.sessions, id)
				expiredCount++
			}
		}
		s.sessionsMu.Unlock()

		if expiredCount > 0 {
			slog.Debug("Cleaned up expired sessions", "count", expiredCount)
		}
	}
}
