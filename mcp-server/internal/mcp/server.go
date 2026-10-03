package mcp

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/thearchitectit/guardrail-mcp/internal/audit"
	"github.com/thearchitectit/guardrail-mcp/internal/budget"
	"github.com/thearchitectit/guardrail-mcp/internal/cache"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
	"github.com/thearchitectit/guardrail-mcp/internal/notifications"
	"github.com/thearchitectit/guardrail-mcp/internal/validation"
)

// MCPServer handles MCP protocol requests
type MCPServer struct {
	mcpServer            *server.MCPServer
	httpServer           *server.StreamableHTTPServer
	rawServer            *http.Server
	db                   *database.DB
	cache                *cache.Client
	audit                *audit.Logger
	validator            *validation.ValidationEngine
	config               *config.Config
	guardrailsEngine     *guardrails.Engine
	version              string
	visionTools          *VisionToolSet
	webhookStore         *database.WebhookStore
	webhookDispatcher    *notifications.Dispatcher
	budgetStore          *database.BudgetStore
	budgetGovernor       *budget.Governor
	agentStateStore      *database.AgentStateStore
	fileReadStore        *database.FileReadStore
	taskAttemptStore     *database.TaskAttemptStore
	haltEventStore       *database.HaltEventStore
	uncertaintyStore     *database.UncertaintyStore
	sessions             map[string]*models.Session
	sessionsMu           sync.RWMutex
	productionCodeStore  *database.ProductionCodeStore
	fixVerificationStore *database.FixVerificationStore
}

// SetWebhookStore sets the webhook store for notification tools.
func (s *MCPServer) SetWebhookStore(store *database.WebhookStore) {
	s.webhookStore = store
}

// SetWebhookDispatcher sets the webhook dispatcher for notification delivery.
func (s *MCPServer) SetWebhookDispatcher(dispatcher *notifications.Dispatcher) {
	s.webhookDispatcher = dispatcher
}

// SetBudget sets the budget store and governor for budget management tools.
func (s *MCPServer) SetBudget(store *database.BudgetStore, governor *budget.Governor) {
	s.budgetStore = store
	s.budgetGovernor = governor
}

// SetAgentStateStore sets the agent state store for lifecycle tools.
func (s *MCPServer) SetAgentStateStore(store *database.AgentStateStore) {
	s.agentStateStore = store
}

// NewMCPServer creates a new MCP server instance.
func NewMCPServer(cfg *config.Config, db *database.DB, cacheClient *cache.Client, auditLogger *audit.Logger, validator *validation.ValidationEngine, fileReadStore *database.FileReadStore, taskAttemptStore *database.TaskAttemptStore, haltEventStore *database.HaltEventStore) *MCPServer {
	s := &MCPServer{
		mcpServer: server.NewMCPServer(
			"Guardrail Enforcement Server",
			cfg.SchemaVersion,
			server.WithResourceCapabilities(true, true),
		),
		db:                   db,
		cache:                cacheClient,
		audit:                auditLogger,
		validator:            validator,
		config:               cfg,
		fileReadStore:        fileReadStore,
		taskAttemptStore:     taskAttemptStore,
		haltEventStore:       haltEventStore,
		uncertaintyStore:     database.NewUncertaintyStore(db.DB),
		sessions:             make(map[string]*models.Session),
		productionCodeStore:  database.NewProductionCodeStore(db),
		fixVerificationStore: database.NewFixVerificationStore(db),
	}

	// Initialize vision tools if configured
	if os.Getenv("VISION_ENABLED") == "true" {
		vts, err := NewVisionToolSet()
		if err != nil {
			slog.Error("Failed to initialize vision tools", "error", err)
		} else {
			s.visionTools = vts
		}
	}

	s.setupHandlers()
	return s
}

// VisionTools returns the vision tool set if enabled, or nil.
func (s *MCPServer) VisionTools() *VisionToolSet {
	return s.visionTools
}

// SetGuardrailsEngine sets the guardrails engine for content classification and
// policy checks. The guardrail_classify_content and guardrail_check_policy tools
// are registered regardless, but return a "not configured" error until set.
func (s *MCPServer) SetGuardrailsEngine(engine *guardrails.Engine) {
	s.guardrailsEngine = engine
}

// Start starts the MCP server on the given address.
func (s *MCPServer) Start(addr string) error {
	return s.Serve(addr)
}

// Shutdown gracefully shuts down the MCP server.
func (s *MCPServer) Shutdown(ctx context.Context) error {
	if s.rawServer != nil {
		return s.rawServer.Shutdown(ctx)
	}
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *MCPServer) setupHandlers() {
	for _, tool := range s.toolList() {
		s.mcpServer.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := req.Params.Name
			arguments := req.GetArguments()
			if s.visionTools != nil {
				result, err := s.visionTools.dispatch(ctx, name, arguments)
				if err == nil {
					return result, nil
				}
			}
			return s.handleToolCall(ctx, name, arguments)
		})
	}

	s.setupResources()
}

func (s *MCPServer) handleToolCall(ctx context.Context, name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	slog.Info("Tool call received", "name", name, "args", args)

	switch name {
	case "guardrail_init_session":
		return s.handleInitSession(ctx, args)
	case "guardrail_validate_bash":
		return s.handleValidateBash(ctx, args)
	case "guardrail_validate_file_edit":
		return s.handleValidateFileEdit(ctx, args)
	case "guardrail_validate_git_operation":
		return s.handleValidateGitOperation(ctx, args)
	case "guardrail_pre_work_check":
		return s.handlePreWorkCheck(ctx, args)
	case "guardrail_get_context":
		return s.handleGetContext(ctx, args)
	case "guardrail_validate_scope":
		return s.handleValidateScope(ctx, args)
	case "guardrail_validate_commit":
		return s.handleValidateCommit(ctx, args)
	case "guardrail_prevent_regression":
		return s.handlePreventRegression(ctx, args)
	case "guardrail_check_test_prod_separation":
		return s.handleCheckTestProdSeparation(ctx, args)
	case "guardrail_validate_push":
		return s.handleValidatePush(ctx, args)
	case "guardrail_record_file_read":
		return s.handleRecordFileRead(ctx, args)
	case "guardrail_record_attempt":
		return s.handleRecordAttempt(ctx, args)
	case "guardrail_verify_file_read":
		return s.handleVerifyFileRead(ctx, args)
	case "guardrail_validate_three_strikes":
		return s.handleValidateThreeStrikes(ctx, args)
	case "guardrail_validate_exact_replacement":
		return s.handleValidateExactReplacement(ctx, args)
	case "guardrail_reset_attempts":
		return s.handleResetAttempts(ctx, args)
	case "guardrail_check_uncertainty":
		return s.handleCheckUncertainty(ctx, args)
	case "guardrail_check_halt_conditions":
		return s.handleCheckHaltConditions(ctx, args)
	case "guardrail_record_halt":
		return s.handleRecordHalt(ctx, args)
	case "guardrail_acknowledge_halt":
		return s.handleAcknowledgeHalt(ctx, args)
	case "guardrail_validate_production_first":
		return s.handleValidateProductionFirst(ctx, args)
	case "guardrail_detect_feature_creep":
		return s.handleDetectFeatureCreep(ctx, args)
	case "guardrail_verify_fixes_intact":
		return s.handleVerifyFixesIntact(ctx, args)
	case "guardrail_team_init":
		return s.handleTeamInit(ctx, args)
	case "guardrail_team_list":
		return s.handleTeamList(ctx, args)
	case "guardrail_team_config_get":
		return s.handleTeamConfigGet(ctx, args)
	case "guardrail_team_config_update":
		return s.handleTeamConfigUpdate(ctx, args)
	case "guardrail_advisor_list":
		return s.handleAdvisorList(ctx, args)
	case "guardrail_advisor_query":
		return s.handleAdvisorQuery(ctx, args)
	case "guardrail_team_assign":
		return s.handleTeamAssign(ctx, args)
	case "guardrail_team_remove":
		return s.handleTeamRemove(ctx, args)
	case "guardrail_project_delete":
		return s.handleProjectDelete(ctx, args)
	case "guardrail_team_health":
		return s.handleTeamHealth(ctx, args)
	case "guardrail_install_skills":
		return s.handleInstallSkills(ctx, args)
	case "guardrail_classify_content":
		return s.handleClassifyContent(ctx, args)
	case "guardrail_check_policy":
		return s.handleCheckPolicy(ctx, args)
	// Webhook notification tools
	case "configure_webhook":
		return s.handleConfigureWebhook(ctx, args)
	case "test_webhook":
		return s.handleTestWebhook(ctx, args)
	case "list_webhooks":
		return s.handleListWebhooks(ctx, args)
	case "delete_webhook":
		return s.handleDeleteWebhook(ctx, args)
	case "get_webhook_deliveries":
		return s.handleGetWebhookDeliveries(ctx, args)
	// Budget management tools
	case "configure_budget":
		return s.handleConfigureBudget(ctx, args)
	case "get_budget_status":
		return s.handleGetBudgetStatus(ctx, args)
	case "list_budgets":
		return s.handleListBudgets(ctx, args)
	case "get_budget_history":
		return s.handleGetBudgetHistory(ctx, args)
	case "delete_budget":
		return s.handleDeleteBudget(ctx, args)
	// Agent lifecycle tools
	case "create_agent_session":
		return s.handleCreateAgentSession(ctx, args)
	case "transition_agent_state":
		return s.handleTransitionAgentState(ctx, args)
	case "get_agent_state":
		return s.handleGetAgentState(ctx, args)
	case "list_agent_sessions":
		return s.handleListAgentSessions(ctx, args)
	case "force_agent_state":
		return s.handleForceAgentState(ctx, args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// buildToolResult removed — use the version in tools_extended.go
// which takes (result interface{}, isError bool)

// sessionTTL bounds how long a session issued by guardrail_init_session
// remains valid.
const sessionTTL = 8 * time.Hour

// lookupSession returns the session for a token, treating an expired session
// as absent. This is the single place expiry is enforced, so every tool that
// validates a session token agrees on the answer.
func (s *MCPServer) lookupSession(token string) (*models.Session, bool) {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	session, ok := s.sessions[token]
	if !ok {
		return nil, false
	}
	if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
		return nil, false
	}
	return session, true
}

// sessionValid reports whether a token refers to a live session.
func (s *MCPServer) sessionValid(token string) bool {
	_, ok := s.lookupSession(token)
	return ok
}

// registerSession records a newly issued session and evicts expired ones so
// the map does not grow without bound.
func (s *MCPServer) registerSession(session *models.Session) {
	now := time.Now()

	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()

	for token, existing := range s.sessions {
		if !existing.ExpiresAt.IsZero() && now.After(existing.ExpiresAt) {
			delete(s.sessions, token)
		}
	}
	s.sessions[session.Token] = session
}

func (s *MCPServer) handleInitSession(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	userID, _ := args["user_id"].(string)
	env, _ := args["environment"].(string)

	token := make([]byte, 24) // 192 bits — sufficient entropy for session tokens
	if _, err := rand.Read(token); err != nil {
		return buildToolResult(map[string]interface{}{
			"error": "failed to generate session token",
		}, true)
	}
	sessionID := hex.EncodeToString(token)

	now := time.Now()
	// Register the token before returning it. Nothing used to write to this
	// map, so every tool that validates session_token against it rejected
	// tokens issued here as invalid — making the read-before-edit flow and
	// the whole halt/attempt family unusable.
	s.registerSession(&models.Session{
		Token:     sessionID,
		CreatedAt: now,
		ExpiresAt: now.Add(sessionTTL),
	})

	result := models.SessionInfo{
		SessionID:   sessionID,
		UserID:      userID,
		Environment: env,
		StartTime:   now,
	}

	return buildToolResult(result, false)
}

// Serve HTTP requests (stateless StreamableHTTP for MCP)
func (s *MCPServer) Serve(addr string) error {
	s.httpServer = server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithEndpointPath("/mcp"),
		server.WithStateLess(true),
	)
	mux := http.NewServeMux()
	mux.Handle("/mcp", requireBearer(s.config.MCPAPIKey, s.httpServer))
	s.rawServer = &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	return s.rawServer.ListenAndServe()
}

// requireBearer rejects requests lacking the configured MCP API key.
// An empty configured key fails closed (everything is rejected).
func requireBearer(key string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
		if key == "" || len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") ||
			subtle.ConstantTimeCompare([]byte(parts[1]), []byte(key)) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *MCPServer) handleGetContext(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	path, _ := args["path"].(string)
	if path == "" {
		path, _ = os.Getwd()
	}

	// Guard the validator the way the validate_* tools do. Without this a
	// server built without a validation engine panics here instead of
	// returning an error.
	if s.validator == nil {
		return buildToolResult(map[string]interface{}{
			"path":      path,
			"error":     "validation engine not configured",
			"timestamp": time.Now().Format(time.RFC3339),
		}, true)
	}

	ruleCount := s.validator.GetCachedRulesCount()
	result := map[string]interface{}{
		"path":             path,
		"applicable_rules": ruleCount,
		"timestamp":        time.Now().Format(time.RFC3339),
	}

	return buildToolResult(result, false)
}
