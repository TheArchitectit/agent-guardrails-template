package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
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

	// SSE endpoint
	s.echo.GET("/mcp/v1/sse", s.handleSSE)

	// Message endpoint
	s.echo.POST("/mcp/v1/message", s.handleMessage)

	slog.Info("Starting MCP SSE server", "addr", addr)
	return s.echo.Start(addr)
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
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	// Set up SSE session
	sessionID := generateSessionID()
	messageEndpoint := fmt.Sprintf("/mcp/v1/message?session_id=%s", sessionID)

	// Send endpoint event
	fmt.Fprintf(c.Response(), "event: endpoint\ndata: %s\n\n", messageEndpoint)
	c.Response().Flush()

	// Keep connection open
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Send keepalive
			fmt.Fprintf(c.Response(), "event: ping\ndata: {}\n\n")
			c.Response().Flush()
		case <-c.Request().Context().Done():
			return nil
		}
	}
}

// handleMessage handles incoming messages
func (s *MCPServer) handleMessage(c echo.Context) error {
	var request server.JSONRPCRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, server.JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{
				Code:    -32700,
				Message: "Parse error",
			},
		})
	}

	// Process request through MCP server
	response := s.mcpServer.Request(c.Request().Context(), request)

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

func generateSessionID() string {
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}
