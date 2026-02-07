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
	mcpServer   *server.MCPServer
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

	// Create MCP server
	s.mcpServer = server.NewMCPServer(
		"guardrail-mcp",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithToolCapabilities(true),
	)

	// Register tools
	s.registerTools()

	// Register resources
	s.registerResources()

	return s
}

// registerTools registers all MCP tools
func (s *MCPServer) registerTools() {
	// Initialize session tool
	initTool := mcp.NewTool("guardrail_init_session",
		mcp.WithDescription("Initialize a validation session for a project"),
		mcp.WithString("project_slug", mcp.Required(), mcp.Description("Project identifier")),
		mcp.WithString("agent_type", mcp.Description("Agent type (claude-code, opencode, cursor)"), mcp.Enum("claude-code", "opencode", "cursor", "other")),
		mcp.WithString("client_version", mcp.Description("Client version")),
	)
	s.mcpServer.AddTool(initTool, s.handleInitSession)

	// Validate bash command tool
	validateBashTool := mcp.NewTool("guardrail_validate_bash",
		mcp.WithDescription("Validate bash command against forbidden patterns"),
		mcp.WithString("session_token", mcp.Required(), mcp.Description("Session token from init_session")),
		mcp.WithString("command", mcp.Required(), mcp.Description("Bash command to validate")),
		mcp.WithString("working_directory", mcp.Description("Current working directory")),
	)
	s.mcpServer.AddTool(validateBashTool, s.handleValidateBash)

	// Validate file edit tool
	validateEditTool := mcp.NewTool("guardrail_validate_file_edit",
		mcp.WithDescription("Validate file edit operation"),
		mcp.WithString("session_token", mcp.Required()),
		mcp.WithString("file_path", mcp.Required()),
		mcp.WithString("old_string", mcp.Required()),
		mcp.WithString("new_string", mcp.Required()),
		mcp.WithString("change_description", mcp.Description("Description of the change")),
	)
	s.mcpServer.AddTool(validateEditTool, s.handleValidateFileEdit)

	// Validate git operation tool
	validateGitTool := mcp.NewTool("guardrail_validate_git_operation",
		mcp.WithDescription("Validate git command against guardrails"),
		mcp.WithString("session_token", mcp.Required()),
		mcp.WithString("command", mcp.Required(), mcp.Enum("push", "commit", "merge", "rebase", "reset")),
		mcp.WithArray("args", mcp.Description("Command arguments")),
		mcp.WithBoolean("is_force", mcp.Description("Whether this is a force operation")),
	)
	s.mcpServer.AddTool(validateGitTool, s.handleValidateGit)

	// Pre-work check tool
	preWorkTool := mcp.NewTool("guardrail_pre_work_check",
		mcp.WithDescription("Run pre-work checklist from failure registry"),
		mcp.WithString("session_token", mcp.Required()),
		mcp.WithArray("affected_files", mcp.Required(), mcp.Description("Files that will be modified")),
	)
	s.mcpServer.AddTool(preWorkTool, s.handlePreWorkCheck)

	// Get context tool
	getContextTool := mcp.NewTool("guardrail_get_context",
		mcp.WithDescription("Get guardrail context for the session's project"),
		mcp.WithString("session_token", mcp.Required()),
	)
	s.mcpServer.AddTool(getContextTool, s.handleGetContext)
}

// registerResources registers all MCP resources
func (s *MCPServer) registerResources() {
	// Quick reference resource
	quickRefResource := mcp.NewResource(
		"guardrail://quick-reference",
		"Quick Reference",
		mcp.WithResourceDescription("Quick reference card for guardrails"),
		mcp.WithMIMEType("application/json"),
	)
	s.mcpServer.AddResource(quickRefResource, s.handleQuickReference)

	// Active rules resource
	rulesResource := mcp.NewResource(
		"guardrail://rules/active",
		"Active Prevention Rules",
		mcp.WithResourceDescription("Currently active prevention rules"),
		mcp.WithMIMEType("application/json"),
	)
	s.mcpServer.AddResource(rulesResource, s.handleActiveRules)

	// Document resource template
	docResource := mcp.NewResourceTemplate(
		"guardrail://docs/{slug}",
		"Guardrail Document",
		mcp.WithTemplateDescription("Guardrail document by slug"),
		mcp.WithMIMEType("text/markdown"),
	)
	s.mcpServer.AddResourceTemplate(docResource, s.handleDocument)
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
	var request mcp.JSONRPCRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, mcp.JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &mcp.JSONRPCError{
				Code:    -32700,
				Message: "Parse error",
			},
		})
	}

	// Process request through MCP server
	response := s.mcpServer.HandleRequest(c.Request().Context(), request)

	return c.JSON(http.StatusOK, response)
}

// Tool handlers

func (s *MCPServer) handleInitSession(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectSlug, _ := request.Params.Arguments["project_slug"].(string)
	agentType, _ := request.Params.Arguments["agent_type"].(string)
	clientVersion, _ := request.Params.Arguments["client_version"].(string)

	if projectSlug == "" {
		return mcp.NewToolResultError("project_slug is required"), nil
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
	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (s *MCPServer) handleValidateBash(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	command, _ := request.Params.Arguments["command"].(string)

	// TODO: Implement actual validation against prevention rules
	// For now, return success
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
	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (s *MCPServer) handleValidateFileEdit(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filePath, _ := request.Params.Arguments["file_path"].(string)
	newString, _ := request.Params.Arguments["new_string"].(string)

	// TODO: Implement actual validation
	result := map[string]interface{}{
		"valid":      true,
		"violations": []interface{}{},
		"meta": map[string]interface{}{
			"checked_at":     time.Now().Format(time.RFC3339),
			"rules_evaluated": 0,
			"file":           filePath,
			"changes_size":   len(newString),
		},
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (s *MCPServer) handleValidateGit(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	command, _ := request.Params.Arguments["command"].(string)
	isForce, _ := request.Params.Arguments["is_force"].(bool)

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
		return mcp.NewToolResultText(string(resultJSON)), nil
	}

	result := map[string]interface{}{
		"valid":      true,
		"violations": []interface{}{},
	}
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (s *MCPServer) handlePreWorkCheck(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	affectedFiles, _ := request.Params.Arguments["affected_files"].([]interface{})

	// Convert to string slice
	files := make([]string, len(affectedFiles))
	for i, f := range affectedFiles {
		files[i], _ = f.(string)
	}

	// Get active failures for these files
	failStore := database.NewFailureStore(s.db)
	failures, err := failStore.GetActiveByFiles(ctx, files)

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to check failures: %v", err)), nil
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
	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (s *MCPServer) handleGetContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionToken, _ := request.Params.Arguments["session_token"].(string)

	s.sessionsMu.RLock()
	session, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		return mcp.NewToolResultError("Invalid session token"), nil
	}

	// Get project context
	projStore := database.NewProjectStore(s.db)
	proj, err := projStore.GetBySlug(ctx, session.ProjectSlug)

	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("# Default Guardrails\n\nNo project-specific context found for %s", session.ProjectSlug)), nil
	}

	return mcp.NewToolResultText(proj.GuardrailContext), nil
}

// Resource handlers

func (s *MCPServer) handleQuickReference(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
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
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "guardrail://quick-reference",
			MIMEType: "application/json",
			Text:     string(contentJSON),
		},
	}, nil
}

func (s *MCPServer) handleActiveRules(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	ruleStore := database.NewRuleStore(s.db)
	rules, err := ruleStore.GetActiveRules(ctx)

	if err != nil {
		return nil, err
	}

	rulesJSON, _ := json.MarshalIndent(rules, "", "  ")
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "guardrail://rules/active",
			MIMEType: "application/json",
			Text:     string(rulesJSON),
		},
	}, nil
}

func (s *MCPServer) handleDocument(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	slug := request.Params.URI // Extract slug from URI

	docStore := database.NewDocumentStore(s.db)
	doc, err := docStore.GetBySlug(ctx, slug)

	if err != nil {
		return nil, fmt.Errorf("document not found: %s", slug)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "text/markdown",
			Text:     doc.Content,
		},
	}, nil
}

func generateSessionID() string {
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}
