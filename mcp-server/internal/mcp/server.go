package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
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
	"github.com/thearchitectit/guardrail-mcp/internal/metrics"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
	"github.com/thearchitectit/guardrail-mcp/internal/validation"
)

// contextKey is a type-safe context key to avoid string allocation
// See: https://golang.org/pkg/context/#WithValue
type contextKey int

const (
	ctxKeySessionID contextKey = iota
)

// Pre-allocated byte slices for common SSE messages to reduce allocations
var (
	sseEndpointPrefix = []byte("event: endpoint\ndata: ")
	sseMessagePrefix  = []byte("event: message\ndata: ")
	sseDoubleNewline  = []byte("\n\n")
	ssePingComment    = []byte(": ping\n\n")
)

// jsonBufferPool provides reusable buffers for JSON encoding
var jsonBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 4096) // Pre-allocate 4KB buffers
	},
}

// MCPServer wraps the MCP server with guardrail dependencies
type MCPServer struct {
	echo             *echo.Echo
	cfg              *config.Config
	db               *database.DB
	cache            *cache.Client
	auditLogger      *audit.Logger
	validationEngine *validation.ValidationEngine
	mcpServer        server.MCPServer
	sessions         map[string]*Session
	sessionsMu       sync.RWMutex
}

// Session represents an MCP client session
type Session struct {
	ID            string
	ProjectSlug   string
	AgentType     string
	ClientVersion string
	CreatedAt     time.Time
	LastActivity  time.Time
	ResponseQueue chan []byte
	Closed        chan struct{}
}

// NewMCPServer creates a new MCP server
func NewMCPServer(cfg *config.Config, db *database.DB, cacheClient *cache.Client, auditLogger *audit.Logger, validationEngine *validation.ValidationEngine) *MCPServer {
	s := &MCPServer{
		cfg:              cfg,
		db:               db,
		cache:            cacheClient,
		auditLogger:      auditLogger,
		validationEngine: validationEngine,
		sessions:         make(map[string]*Session),
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
			{
				Name:        "guardrail_validate_scope",
				Description: "Check if a file path is within authorized scope",
				InputSchema: mcp.ToolInputSchema{
					Type: "object",
					Properties: mcp.ToolInputSchemaProperties{
						"file_path": map[string]interface{}{
							"type":        "string",
							"description": "The file path to validate",
						},
						"authorized_scope": map[string]interface{}{
							"type":        "string",
							"description": "The authorized scope prefix (e.g., /app/src)",
						},
					},
				},
			},
			{
				Name:        "guardrail_validate_commit",
				Description: "Validate commit message format compliance (conventional commits)",
				InputSchema: mcp.ToolInputSchema{
					Type: "object",
					Properties: mcp.ToolInputSchemaProperties{
						"message": map[string]interface{}{
							"type":        "string",
							"description": "The commit message to validate",
						},
					},
				},
			},
			{
				Name:        "guardrail_prevent_regression",
				Description: "Check failure registry for matching patterns to prevent regressions",
				InputSchema: mcp.ToolInputSchema{
					Type: "object",
					Properties: mcp.ToolInputSchemaProperties{
						"file_paths": map[string]interface{}{
							"type":        "array",
							"description": "Array of file paths that will be modified",
						},
						"code_content": map[string]interface{}{
							"type":        "string",
							"description": "Code content to check against regression patterns",
						},
					},
				},
			},
			{
				Name:        "guardrail_check_test_prod_separation",
				Description: "Verify test/production environment isolation",
				InputSchema: mcp.ToolInputSchema{
					Type: "object",
					Properties: mcp.ToolInputSchemaProperties{
						"file_path": map[string]interface{}{
							"type":        "string",
							"description": "The file path to check",
						},
						"environment": map[string]interface{}{
							"type":        "string",
							"description": "Environment type: test or prod",
							"enum":        []string{"test", "prod"},
						},
					},
				},
			},
			{
				Name:        "guardrail_validate_push",
				Description: "Validate git push safety conditions",
				InputSchema: mcp.ToolInputSchema{
					Type: "object",
					Properties: mcp.ToolInputSchemaProperties{
						"branch": map[string]interface{}{
							"type":        "string",
							"description": "The branch being pushed to",
						},
						"is_force": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether this is a force push",
						},
						"has_unpushed_commits": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether there are unpushed commits",
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
				{
					Uri:         "guardrail://docs/agent-guardrails",
					Name:        "Agent Guardrails",
					Description: "Core safety protocols and guardrails",
					MimeType:    "text/markdown",
				},
				{
					Uri:         "guardrail://docs/four-laws",
					Name:        "Four Laws of Agent Safety",
					Description: "The Four Laws of Agent Safety (canonical)",
					MimeType:    "text/markdown",
				},
				{
					Uri:         "guardrail://docs/halt-conditions",
					Name:        "Halt Conditions",
					Description: "When to stop and ask for help",
					MimeType:    "text/markdown",
				},
				{
					Uri:         "guardrail://docs/workflows",
					Name:        "Workflow Documentation",
					Description: "All workflow documentation index",
					MimeType:    "text/markdown",
				},
				{
					Uri:         "guardrail://docs/standards",
					Name:        "Standards Documentation",
					Description: "All standards documentation index",
					MimeType:    "text/markdown",
				},
				{
					Uri:         "guardrail://docs/pre-work-checklist",
					Name:        "Pre-Work Checklist",
					Description: "Mandatory pre-work regression checklist",
					MimeType:    "text/markdown",
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
	case "guardrail_validate_scope":
		return s.handleValidateScope(ctx, arguments)
	case "guardrail_validate_commit":
		return s.handleValidateCommit(ctx, arguments)
	case "guardrail_prevent_regression":
		return s.handlePreventRegression(ctx, arguments)
	case "guardrail_check_test_prod_separation":
		return s.handleCheckTestProdSeparation(ctx, arguments)
	case "guardrail_validate_push":
		return s.handleValidatePush(ctx, arguments)
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
		// Use compact JSON instead of indented for better performance
		// Pre-allocated response to avoid map allocations
		contentJSON := []byte(`{"forbidden_commands":["rm -rf /","git push --force","git reset --hard"],"required_checks":["pre_work_check","validate_file_edit"]}`)
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
		// Use compact JSON marshaling for better performance
		rulesJSON, _ := json.Marshal(rules)
		return &mcp.ReadResourceResult{
			Contents: []interface{}{
				mcp.TextResourceContents{
					Uri:      uri,
					MimeType: "application/json",
					Text:     string(rulesJSON),
				},
			},
		}, nil

	case "guardrail://docs/agent-guardrails":
		return s.readAgentGuardrailsResource(ctx, uri)

	case "guardrail://docs/workflows":
		return s.readWorkflowsResource(ctx, uri)

	case "guardrail://docs/standards":
		return s.readStandardsResource(ctx, uri)

	case "guardrail://principles/four-laws", "guardrail://docs/four-laws":
		return s.readFourLawsResource(ctx, uri)

	case "guardrail://halt-conditions", "guardrail://docs/halt-conditions":
		return s.readHaltConditionsResource(ctx, uri)

	case "guardrail://checklist/pre-work", "guardrail://docs/pre-work-checklist":
		return s.readPreWorkChecklistResource(ctx, uri)

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

	// Body limit - prevent DoS via large payloads (skip for SSE which has no body)
	s.echo.Use(middleware.BodyLimit("1M"))

	// SSE endpoint - no timeout, long-lived connection
	s.echo.GET("/mcp/v1/sse", s.handleSSE)

	// Message endpoint - with timeout for request processing
	// Note: Timeout applied at handler level to allow SSE to stay open
	s.echo.POST("/mcp/v1/message", s.handleMessage, middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: s.cfg.RequestTimeout,
	}))

	// Start session cleanup goroutine with panic recovery
	go s.runSessionCleanup()

	slog.Info("Starting MCP SSE server", "addr", addr)
	return s.echo.Start(addr)
}

// runSessionCleanup runs the session cleanup loop with panic recovery
func (s *MCPServer) runSessionCleanup() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Session cleanup goroutine panicked, restarting", "panic", r)
			// Restart the cleanup goroutine after a delay
			time.Sleep(5 * time.Second)
			go s.runSessionCleanup()
		}
	}()
	s.sessionCleanup()
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

// handleSSE handles SSE connections with optimized string building
func (s *MCPServer) handleSSE(c echo.Context) error {
	// Validate origin for CORS - only allow specific origins
	origin := c.Request().Header.Get("Origin")
	originAllowed := isOriginAllowed(origin, s.cfg.ProductionMode)

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

	now := time.Now()
	session := &Session{
		ID:            sessionID,
		CreatedAt:     now,
		LastActivity:  now,
		ResponseQueue: make(chan []byte, 100),
		Closed:        make(chan struct{}),
	}

	// Store session in map (sessions are created during SSE connection).
	s.sessionsMu.Lock()
	s.sessions[sessionID] = session
	s.sessionsMu.Unlock()

	defer func() {
		s.sessionsMu.Lock()
		if current, ok := s.sessions[sessionID]; ok && current == session {
			delete(s.sessions, sessionID)
			close(session.Closed)
		}
		s.sessionsMu.Unlock()
	}()

	// Build endpoint URL using strings.Builder for efficiency
	var sb strings.Builder
	// Pre-allocate capacity: scheme + "://" + host + path + session_id (~100 chars)
	sb.Grow(100)
	if c.Request().TLS != nil {
		sb.WriteString("https://")
	} else {
		sb.WriteString("http://")
	}
	sb.WriteString(c.Request().Host)
	sb.WriteString("/mcp/v1/message?session_id=")
	sb.WriteString(sessionID)
	messageEndpoint := sb.String()

	slog.Debug("SSE connection established", "session_id", sessionID)

	// Send endpoint event using pre-allocated buffers
	// Format: event: endpoint\ndata: <absolute_url>\n\n
	if err := writeSSEEvent(c.Response(), sseEndpointPrefix, messageEndpoint); err != nil {
		slog.Warn("SSE endpoint write failed", "session_id", sessionID, "error", err)
		return nil
	}
	c.Response().Flush()

	// Track connection state
	clientGone := c.Request().Context().Done()

	// Send initial keep-alive comment. Use SSE comments instead of a custom
	// `event: ping` payload because some SDKs treat all event data as JSON-RPC
	// and fail on non-message events.
	if err := writeSSEComment(c.Response(), ssePingComment); err != nil {
		slog.Warn("SSE initial keep-alive write failed", "session_id", sessionID, "error", err)
		return nil
	}
	c.Response().Flush()

	// Keep connection open with periodic pings (every 30 seconds)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case payload := <-session.ResponseQueue:
			if err := writeSSEEvent(c.Response(), sseMessagePrefix, string(payload)); err != nil {
				slog.Debug("SSE response write failed, client disconnected", "session_id", sessionID, "error", err)
				return nil
			}
			c.Response().Flush()
		case <-ticker.C:
			if err := writeSSEComment(c.Response(), ssePingComment); err != nil {
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

// isOriginAllowed checks if the origin is in the allowed list
func isOriginAllowed(origin string, isProduction bool) bool {
	allowedOrigins := []string{"http://localhost:*", "https://localhost:*"}
	if isProduction {
		allowedOrigins = []string{"http://localhost:8081", "https://localhost:8081"}
	}

	for _, allowed := range allowedOrigins {
		if strings.HasSuffix(allowed, ":*") {
			prefix := strings.TrimSuffix(allowed, ":*")
			if strings.HasPrefix(origin, prefix) {
				return true
			}
		} else if origin == allowed || allowed == "*" {
			return true
		}
	}
	return false
}

// writeSSEEvent writes an SSE event efficiently
func writeSSEEvent(w http.ResponseWriter, prefix []byte, data string) error {
	if _, err := w.Write(prefix); err != nil {
		return err
	}
	if _, err := w.Write([]byte(data)); err != nil {
		return err
	}
	_, err := w.Write(sseDoubleNewline)
	return err
}

// writeSSEComment writes an SSE comment line for keep-alive without emitting
// an event payload.
func writeSSEComment(w http.ResponseWriter, comment []byte) error {
	_, err := w.Write(comment)
	return err
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
	session, sessionExists := s.sessions[sessionID]
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

	// Create context with session ID for tool handlers using typed key
	ctx := context.WithValue(c.Request().Context(), ctxKeySessionID, sessionID)

	// Process request through MCP server
	response := s.mcpServer.Request(ctx, request)

	// Update session last activity if session exists
	if sessionExists {
		s.sessionsMu.Lock()
		if sess, ok := s.sessions[sessionID]; ok {
			sess.LastActivity = time.Now()
			session = sess
		}
		s.sessionsMu.Unlock()
	}

	// For SSE sessions, queue JSON-RPC responses onto the SSE stream as
	// `event: message` payloads.
	if sessionExists && session != nil && session.ResponseQueue != nil {
		// Notifications (no ID) do not require a response payload.
		if request.ID == nil {
			return c.NoContent(http.StatusAccepted)
		}

		responseJSON, err := json.Marshal(response)
		if err != nil {
			slog.Error("Failed to marshal JSON-RPC response", "session_id", sessionID, "error", err)
			return c.JSON(http.StatusInternalServerError, server.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Error: &struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}{
					Code:    -32603,
					Message: "Internal error: failed to encode response",
				},
			})
		}

		select {
		case session.ResponseQueue <- responseJSON:
			return c.NoContent(http.StatusAccepted)
		case <-session.Closed:
			slog.Warn("SSE session closed before response enqueue", "session_id", sessionID)
			return c.JSON(http.StatusGone, server.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Error: &struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}{
					Code:    -32000,
					Message: "Session closed",
				},
			})
		case <-time.After(1 * time.Second):
			slog.Warn("SSE response queue full", "session_id", sessionID)
			return c.JSON(http.StatusServiceUnavailable, server.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Error: &struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}{
					Code:    -32000,
					Message: "Session busy",
				},
			})
		}
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

	// Record metrics
	metrics.IncrementActiveSessions()

	// Audit log
	s.auditLogger.LogSession(ctx, audit.EventSessionCreated, sessionID, projectSlug)

	// Get project context with timeout
	projCtx, projCancel := context.WithTimeout(ctx, 5*time.Second)
	defer projCancel()

	projStore := database.NewProjectStore(s.db)
	proj, projErr := projStore.GetBySlug(projCtx, projectSlug)
	if projErr != nil {
		slog.Warn("Failed to get project context", "project_slug", projectSlug, "error", projErr)
	}

	contextStr := ""
	if proj != nil {
		contextStr = proj.GuardrailContext
	}

	// Get active rules count with timeout
	rulesCtx, rulesCancel := context.WithTimeout(ctx, 5*time.Second)
	defer rulesCancel()

	ruleStore := database.NewRuleStore(s.db)
	rules, rulesErr := ruleStore.GetActiveRules(rulesCtx)
	if rulesErr != nil {
		slog.Error("Failed to get active rules", "error", rulesErr)
		rules = []models.PreventionRule{}
	}

	// Use strings.Builder for efficient JSON string construction
	// This avoids reflection overhead of json.Marshal for simple structures
	var sb strings.Builder
	sb.Grow(256) // Pre-allocate estimated size
	sb.WriteString(`{"session_token":"`)
	sb.WriteString(sessionID)
	sb.WriteString(`","expires_at":"`)
	sb.WriteString(time.Now().Add(s.cfg.JWTExpiry).Format(time.RFC3339))
	sb.WriteString(`","project_context":"`)
	// Escape the context string for JSON
	jsonEscape(&sb, contextStr)
	sb.WriteString(`","active_rules_count":`)
	sb.WriteString(strconv.Itoa(len(rules)))
	sb.WriteString(`,"capabilities":["bash_validation","git_validation","edit_validation"]}`)

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: sb.String()}},
	}, nil
}

// jsonEscape escapes a string for JSON embedding
// This is faster than json.Marshal for simple strings
func jsonEscape(sb *strings.Builder, s string) {
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\b':
			sb.WriteString(`\b`)
		case '\f':
			sb.WriteString(`\f`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			if r < 0x20 {
				sb.WriteString(`\u00`)
				sb.WriteByte(hexChar(byte(r) >> 4))
				sb.WriteByte(hexChar(byte(r) & 0x0F))
			} else {
				sb.WriteRune(r)
			}
		}
	}
}

// hexChar returns the hex character for a nibble
func hexChar(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'a' + n - 10
}

// jsonEscapeString escapes a string for safe inclusion in JSON
func jsonEscapeString(s string) string {
	var sb strings.Builder
	jsonEscape(&sb, s)
	return sb.String()
}

func (s *MCPServer) handleValidateBash(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	command, _ := args["command"].(string)

	if command == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"valid":false,"violations":[{"rule_id":"VALIDATION-001","severity":"error","message":"Command is required"}],"meta":{"checked_at":"` + time.Now().Format(time.RFC3339) + `","rules_evaluated":0}}`}},
			IsError: true,
		}, nil
	}

	// Validate against prevention rules for bash commands
	violations, err := s.validationEngine.ValidateInput(ctx, command, []string{"bash", "command"})
	if err != nil {
		slog.Error("Bash validation failed", "error", err, "command", command)
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"valid":false,"violations":[{"rule_id":"VALIDATION-ERROR","severity":"error","message":"Validation engine error: %s"}],"meta":{"checked_at":"%s","rules_evaluated":0}}`, jsonEscapeString(err.Error()), time.Now().Format(time.RFC3339))}},
			IsError: true,
		}, nil
	}

	// Build response using strings.Builder for efficiency
	var sb strings.Builder
	sb.Grow(512)

	valid := len(violations) == 0
	if valid {
		sb.WriteString(`{"valid":true,"violations":[],`)
	} else {
		sb.WriteString(`{"valid":false,"violations":[`)
		for i, v := range violations {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(`{"rule_id":"`)
			jsonEscape(&sb, v.RuleID)
			sb.WriteString(`","name":"`)
			jsonEscape(&sb, v.RuleName)
			sb.WriteString(`","severity":"`)
			jsonEscape(&sb, string(v.Severity))
			sb.WriteString(`","message":"`)
			jsonEscape(&sb, v.Message)
			sb.WriteString(`"}`)
		}
		sb.WriteString(`],`)
	}

	sb.WriteString(`"meta":{"checked_at":"`)
	sb.WriteString(time.Now().Format(time.RFC3339))
	sb.WriteString(`","rules_evaluated":`)
	sb.WriteString(strconv.Itoa(s.validationEngine.GetCachedRulesCount()))
	sb.WriteString(`,"command_analyzed":"`)
	jsonEscape(&sb, command)
	sb.WriteString(`"}}`)

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: sb.String()}},
	}, nil
}

func (s *MCPServer) handleValidateFileEdit(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	filePath, _ := args["file_path"].(string)
	newString, _ := args["new_string"].(string)

	if filePath == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"valid":false,"violations":[{"rule_id":"VALIDATION-001","severity":"error","message":"File path is required"}],"meta":{"checked_at":"` + time.Now().Format(time.RFC3339) + `","rules_evaluated":0}}`}},
			IsError: true,
		}, nil
	}

	// Validate the new content against prevention rules (including security rules for secrets)
	violations, err := s.validationEngine.ValidateInput(ctx, newString, []string{"file_edit", "content", "edit", "security"})
	if err != nil {
		slog.Error("File edit validation failed", "error", err, "file_path", filePath)
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"valid":false,"violations":[{"rule_id":"VALIDATION-ERROR","severity":"error","message":"Validation engine error: %s"}],"meta":{"checked_at":"%s","rules_evaluated":0}}`, jsonEscapeString(err.Error()), time.Now().Format(time.RFC3339))}},
			IsError: true,
		}, nil
	}

	// Also validate the file path for path traversal or sensitive locations
	pathViolations, err := s.validationEngine.ValidateInput(ctx, filePath, []string{"file_path", "path"})
	if err != nil {
		slog.Error("File path validation failed", "error", err, "file_path", filePath)
	}
	violations = append(violations, pathViolations...)

	// Build response using strings.Builder for efficiency
	var sb strings.Builder
	sb.Grow(512)

	valid := len(violations) == 0
	if valid {
		sb.WriteString(`{"valid":true,"violations":[],`)
	} else {
		sb.WriteString(`{"valid":false,"violations":[`)
		for i, v := range violations {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(`{"rule_id":"`)
			jsonEscape(&sb, v.RuleID)
			sb.WriteString(`","name":"`)
			jsonEscape(&sb, v.RuleName)
			sb.WriteString(`","severity":"`)
			jsonEscape(&sb, string(v.Severity))
			sb.WriteString(`","message":"`)
			jsonEscape(&sb, v.Message)
			sb.WriteString(`"}`)
		}
		sb.WriteString(`],`)
	}

	sb.WriteString(`"meta":{"checked_at":"`)
	sb.WriteString(time.Now().Format(time.RFC3339))
	sb.WriteString(`","rules_evaluated":`)
	sb.WriteString(strconv.Itoa(s.validationEngine.GetCachedRulesCount()))
	sb.WriteString(`,"file":"`)
	jsonEscape(&sb, filePath)
	sb.WriteString(`","changes_size":`)
	sb.WriteString(strconv.Itoa(len(newString)))
	sb.WriteString(`}}`)

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: sb.String()}},
	}, nil
}

func (s *MCPServer) handleValidateGit(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	command, _ := args["command"].(string)
	isForce, _ := args["is_force"].(bool)

	if command == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"valid":false,"violations":[{"rule_id":"VALIDATION-001","severity":"error","message":"Command is required"}],"meta":{"checked_at":"` + time.Now().Format(time.RFC3339) + `","rules_evaluated":0}}`}},
			IsError: true,
		}, nil
	}

	var allViolations []validation.Violation

	// Validate the git command against prevention rules
	violations, err := s.validationEngine.ValidateInput(ctx, command, []string{"git", "git_operation"})
	if err != nil {
		slog.Error("Git validation failed", "error", err, "command", command)
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"valid":false,"violations":[{"rule_id":"VALIDATION-ERROR","severity":"error","message":"Validation engine error: %s"}],"meta":{"checked_at":"%s","rules_evaluated":0}}`, jsonEscapeString(err.Error()), time.Now().Format(time.RFC3339))}},
			IsError: true,
		}, nil
	}
	allViolations = append(allViolations, violations...)

	// Check for force push separately if is_force flag is set
	if isForce {
		allViolations = append(allViolations, validation.Violation{
			RuleID:   "PREVENT-FORCE-001",
			RuleName: "No Force Operation",
			Severity: models.SeverityError,
			Message:  "Force operations are not allowed. Use --force-with-lease or standard push instead.",
		})
	}

	// Build response using strings.Builder for efficiency
	var sb strings.Builder
	sb.Grow(512)

	valid := len(allViolations) == 0
	if valid {
		sb.WriteString(`{"valid":true,"violations":[],`)
	} else {
		sb.WriteString(`{"valid":false,"violations":[`)
		for i, v := range allViolations {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(`{"rule_id":"`)
			jsonEscape(&sb, v.RuleID)
			sb.WriteString(`","name":"`)
			jsonEscape(&sb, v.RuleName)
			sb.WriteString(`","severity":"`)
			jsonEscape(&sb, string(v.Severity))
			sb.WriteString(`","message":"`)
			jsonEscape(&sb, v.Message)
			sb.WriteString(`"}`)
		}
		sb.WriteString(`],`)
	}

	sb.WriteString(`"meta":{"checked_at":"`)
	sb.WriteString(time.Now().Format(time.RFC3339))
	sb.WriteString(`","rules_evaluated":`)
	sb.WriteString(strconv.Itoa(s.validationEngine.GetCachedRulesCount()))
	sb.WriteString(`,"command":"`)
	jsonEscape(&sb, command)
	sb.WriteString(`","is_force":`)
	if isForce {
		sb.WriteString("true")
	} else {
		sb.WriteString("false")
	}
	sb.WriteString(`}}`)

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: sb.String()}},
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

	// Use failures directly instead of creating intermediate maps
	// Use compact JSON marshaling for better performance
	result := map[string]interface{}{
		"passed":         len(failures) == 0,
		"checks":         failures,
		"files_affected": files,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		slog.Error("Failed to marshal pre-work check result", "error", err)
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Internal error: failed to format result: %v", err)}},
			IsError: true,
		}, nil
	}
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
// Uses batched deletion to minimize lock contention
func (s *MCPServer) sessionCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		expiredIDs := s.collectExpiredSessions(now)

		if len(expiredIDs) > 0 {
			s.deleteSessionsBatch(expiredIDs)
			slog.Debug("Cleaned up expired sessions", "count", len(expiredIDs))
		}
	}
}

// collectExpiredSessions identifies expired sessions without holding the lock
func (s *MCPServer) collectExpiredSessions(now time.Time) []string {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	// Pre-allocate with estimated capacity (10% of sessions)
	expiredCount := 0
	for _, session := range s.sessions {
		if now.Sub(session.LastActivity) > time.Hour {
			expiredCount++
		}
	}

	if expiredCount == 0 {
		return nil
	}

	expiredIDs := make([]string, 0, expiredCount)
	for id, session := range s.sessions {
		if now.Sub(session.LastActivity) > time.Hour {
			expiredIDs = append(expiredIDs, id)
		}
	}
	return expiredIDs
}

// deleteSessionsBatch removes multiple sessions with a single lock acquisition
func (s *MCPServer) deleteSessionsBatch(ids []string) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()

	for _, id := range ids {
		delete(s.sessions, id)
	}
}
