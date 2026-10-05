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
	"github.com/thearchitectit/guardrail-mcp/internal/auth"
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

// Context keys for approved audit fields propagated from the auth layer.
type ctxKey string

const (
	ctxKeyRequestID    ctxKey = "request_id"
	ctxKeyPrincipalID  ctxKey = "principal_id"
	ctxKeyCredentialID ctxKey = "credential_id"
	ctxKeyCaller       ctxKey = "authz_caller"
)

// callerFromContext returns the server-controlled principal resolved at the
// auth boundary. Identity is never taken from tool arguments.
func callerFromContext(ctx context.Context) (auth.Caller, bool) {
	c, ok := ctx.Value(ctxKeyCaller).(auth.Caller)
	return c, ok
}

// safeResourceIDKeys are the only argument keys whose values may appear in
// logs as a resource identifier, and only when they look like opaque IDs.
var safeResourceIDKeys = []string{"project_id", "session_id", "document_id", "rule_id", "resource_id", "review_id", "budget_id", "webhook_id"}

// safeResourceID extracts an opaque identifier from allowlisted argument keys.
// It never returns free-form argument values, so a secret planted in any other
// (or nested) argument cannot reach the log via this field.
func safeResourceID(args map[string]interface{}) string {
	for _, key := range safeResourceIDKeys {
		v, ok := args[key].(string)
		if !ok || v == "" || len(v) > 128 {
			continue
		}
		for _, r := range v {
			if !(r == '-' || r == '_' || r == '.' || r == ':' ||
				(r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
				return ""
			}
		}
		return v
	}
	return ""
}

func (s *MCPServer) handleToolCall(ctx context.Context, name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	// Record only the approved audit fields (R16-05). Raw tool arguments are
	// never logged: they routinely carry credentials and other secrets.
	requestID, _ := ctx.Value(ctxKeyRequestID).(string)
	principalID, _ := ctx.Value(ctxKeyPrincipalID).(string)
	credentialID, _ := ctx.Value(ctxKeyCredentialID).(string)

	// Spec 11 section 4.4 intersection before every effect. Missing/unknown
	// inputs deny with a stable non-leaking permission error.
	if denied := s.authorizeToolCall(ctx, name, args); denied != nil {
		slog.Info("tool_call",
			"operation", name,
			"request_id", requestID,
			"principal_id", principalID,
			"credential_id", credentialID,
			"resource", auth.SafeResourceID(args),
			"reason", "permission_denied",
			"outcome", "denied",
			"policy_version", auth.PolicyVersion,
		)
		return denied, nil
	}

	result, err := s.dispatchToolCall(ctx, name, args)

	outcome := "success"
	if err != nil {
		outcome = "error"
	} else if result != nil && result.IsError {
		outcome = "rejected"
	}

	slog.Info("tool_call",
		"operation", name,
		"request_id", requestID,
		"principal_id", principalID,
		"credential_id", credentialID,
		"resource", auth.SafeResourceID(args),
		"reason", "tool_dispatch",
		"outcome", outcome,
		"policy_version", auth.PolicyVersion,
	)
	return result, err
}

// authorizeToolCall applies the scope × role × resource intersection before a
// tool effect runs. It returns a stable permission-denied tool result on deny,
// or nil when the call may proceed. Security-sensitive mutations fail closed
// if their required durable audit record cannot be written.
func (s *MCPServer) authorizeToolCall(ctx context.Context, name string, args map[string]interface{}) *mcp.CallToolResult {
	caller, ok := callerFromContext(ctx)
	if !ok || caller.PrincipalID == "" {
		// No server-controlled principal: deny. Never fall back to
		// argument-sourced identity.
		return permissionDeniedResult()
	}

	var action auth.ActionRequest
	if caller.LegacyClass != "" {
		// Legacy MCP key: named read/validate tools only (Spec 16 section 4).
		la, ok := auth.LegacyMCPCallAllowed(caller.LegacyClass, name)
		if !ok {
			s.mcpAuditDecision(ctx, caller, auth.ActionRequest{Name: name, Kind: "unknown"},
				auth.Decision{Allow: false, Code: auth.ReasonLegacyNotAllowed, Reason: "legacy tool not on approved table"}, false)
			return permissionDeniedResult()
		}
		action = la
		action.Resource = auth.SafeResourceID(args)
	} else {
		action = auth.ClassifyMCPTool(name, args)
	}

	decision := auth.Decide(caller, action)
	if !decision.Allow {
		s.mcpAuditDecision(ctx, caller, action, decision, false)
		return permissionDeniedResult()
	}

	// Security-sensitive mutations require a durable audit record first.
	required := auth.IsSensitiveMCPTool(name) || action.Kind == auth.ActionAdmin
	if err := s.mcpAuditDecision(ctx, caller, action, decision, required); err != nil {
		slog.Error("required audit failed; denying security-sensitive tool call",
			"error", err,
			"principal_id", caller.PrincipalID,
			"tool", name,
		)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "authorization audit unavailable"}},
			IsError: true,
		}
	}
	return nil
}

// permissionDeniedResult is the stable non-leaking MCP permission error shape.
// It never reveals whether a protected resource exists.
func permissionDeniedResult() *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: auth.FormatDenyError()}},
		IsError: true,
	}
}

// mcpAuditDecision records a structured allow/deny decision. Principal and
// opaque credential IDs only — never raw keys or hashAPIKey-as-identity.
func (s *MCPServer) mcpAuditDecision(ctx context.Context, caller auth.Caller, action auth.ActionRequest, d auth.Decision, required bool) error {
	if s.audit == nil {
		if required {
			return fmt.Errorf("audit logger not configured")
		}
		return nil
	}
	decision := auth.DecisionAllow
	if !d.Allow {
		decision = auth.DecisionDeny
	}
	return s.audit.LogDecision(ctx, audit.DecisionRecord{
		PrincipalID:   caller.PrincipalID,
		CredentialID:  caller.CredentialID,
		Action:        action.Name,
		Resource:      action.Resource,
		Decision:      decision,
		Reason:        d.Code,
		PolicyVersion: auth.PolicyVersion,
		Kind:          string(action.Kind),
		Surface:       "mcp",
	}, required)
}

// dispatchToolCall routes a tool invocation to its handler. Split from
// handleToolCall so the approved-field logging wraps every outcome path.
func (s *MCPServer) dispatchToolCall(ctx context.Context, name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
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
	// Load the credential-to-principal registry (Spec 15 / gr-xp-01). When a
	// registry is configured but fails to load, fail startup: never fall back
	// to unrestricted legacy access.
	registry, err := auth.LoadFromSourcesEx(s.config.CredentialRegistryJSON, s.config.CredentialRegistryFile, s.config.CredentialVerifierKey, s.config.CredentialVerifierKeyFile)
	if err != nil {
		slog.Error("credential registry configured but failed to load; refusing to start", "error", err)
		return fmt.Errorf("credential registry configured but failed to load: %w", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/mcp", requireBearerWithRegistry(s.config.MCPAPIKey, registry, nil, s.httpServer))
	s.rawServer = &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	return s.rawServer.ListenAndServe()
}

// requireBearer rejects requests lacking the configured MCP API key.
// An empty configured key fails closed (everything is rejected).
func requireBearer(key string, next http.Handler) http.Handler {
	return requireBearerWithRegistry(key, nil, nil, next)
}

// requireBearerWithRegistry authenticates the MCP endpoint. A credential that
// resolves in the registry is accepted for its registered principal; otherwise
// the legacy configured key is accepted unchanged — but only when that key is
// present, so registry-only cutover does not reinstate legacy access. When
// regErr is non-nil (a registry was configured but failed to load) every
// request is denied. Identity is never taken from the presented secret or a
// log hash.
func requireBearerWithRegistry(key string, registry *auth.Registry, regErr error, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Configured-but-broken registry: deny ALL traffic. No legacy fallback.
		if regErr != nil {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
		token := ""
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			token = parts[1]
		}
		// Fail closed when no credential is presented.
		if token == "" {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// A credential registered with a principal is accepted even when the
		// legacy key is absent (registry-only cutover).
		if registry.Enabled() {
			principal, ok := registry.Resolve(token)
			if ok {
				caller := principal.Caller()
				ctx := r.Context()
				ctx = context.WithValue(ctx, ctxKeyPrincipalID, caller.PrincipalID)
				ctx = context.WithValue(ctx, ctxKeyCredentialID, caller.CredentialID)
				ctx = context.WithValue(ctx, ctxKeyCaller, caller)
				if reqID := r.Header.Get("X-Request-ID"); reqID != "" {
					ctx = context.WithValue(ctx, ctxKeyRequestID, reqID)
				}
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		// The legacy configured key authenticates only when present. An absent
		// legacy key authenticates nobody. Legacy principals are constrained
		// to named read/validate MCP tools (Spec 16 section 4).
		legacyClass := ""
		if key != "" && subtle.ConstantTimeCompare([]byte(token), []byte(key)) == 1 {
			legacyClass = "mcp"
		}
		if legacyClass == "" {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		caller, ok := auth.LegacyPrincipal(legacyClass)
		if !ok {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, ctxKeyPrincipalID, caller.PrincipalID)
		ctx = context.WithValue(ctx, ctxKeyCredentialID, caller.CredentialID)
		ctx = context.WithValue(ctx, ctxKeyCaller, caller)
		if reqID := r.Header.Get("X-Request-ID"); reqID != "" {
			ctx = context.WithValue(ctx, ctxKeyRequestID, reqID)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
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
