package auth

import (
	"fmt"
	"net/http"
	"strings"
)

// Scope catalog (Spec 16 §4). Unknown scopes never grant authority.
const (
	ScopeMCPRead     = "mcp:read"
	ScopeMCPValidate = "mcp:validate"
	ScopeMCPMutate   = "mcp:mutate"
	ScopeRESTRead    = "rest:read"
	ScopeRESTWrite   = "rest:write"
	ScopeIDEValidate = "ide:validate"
	ScopeAdminManage = "admin:manage"
)

// Role catalog (Spec 16 §4). Unknown roles deny.
const (
	RoleReader           = "reader"
	RoleDeveloper        = "developer"
	RoleSecurityOperator = "security-operator"
	RoleAdministrator    = "administrator"
)

// KnownScopes is the published scope catalog. Used for unknown-scope rejection
// at the decision boundary (missing/unknown inputs deny).
var KnownScopes = map[string]bool{
	ScopeMCPRead:     true,
	ScopeMCPValidate: true,
	ScopeMCPMutate:   true,
	ScopeRESTRead:    true,
	ScopeRESTWrite:   true,
	ScopeIDEValidate: true,
	ScopeAdminManage: true,
}

// KnownRoles is the published role catalog.
var KnownRoles = map[string]bool{
	RoleReader:           true,
	RoleDeveloper:        true,
	RoleSecurityOperator: true,
	RoleAdministrator:    true,
}

// ActionKind classifies the privilege class of an operation.
type ActionKind string

const (
	ActionRead     ActionKind = "read"
	ActionValidate ActionKind = "validate"
	ActionMutate   ActionKind = "mutate"
	ActionAdmin    ActionKind = "admin"
)

// Decision outcome values recorded in audit.
const (
	DecisionAllow = "allow"
	DecisionDeny  = "deny"
)

// Reason codes for audit and denial. Stable and non-leaking.
const (
	ReasonAllowed             = "allowed"
	ReasonUnauthenticated     = "unauthenticated"
	ReasonUnknownScope        = "unknown_scope"
	ReasonMissingScope        = "scope_denied"
	ReasonUnknownRole         = "unknown_role"
	ReasonRoleDenied          = "role_denied"
	ReasonResourceDenied      = "resource_denied"
	ReasonUnknownAction       = "unknown_action"
	ReasonLegacyNotAllowed    = "legacy_not_allowed"
	ReasonLegacyWrongClass    = "legacy_wrong_class"
	ReasonAuditUnavailable    = "audit_unavailable"
	ReasonPolicyVersion       = "policy_version"
	ReasonInvalidCredential   = "invalid_credential"
	ReasonCredentialRevoked   = "credential_revoked"
	ReasonMissingRoleResource = "missing_role_or_resource"
)

// PolicyVersion is stamped on every authorization decision record.
const PolicyVersion = "authz-1"

// Caller is the server-controlled principal identity resolved from a
// credential. Identity is never taken from tool arguments or request content.
type Caller struct {
	PrincipalID  string
	CredentialID string
	Scopes       []string
	Role         string
	Resources    []string
	// LegacyClass is "", "mcp", or "ide". Non-empty marks a legacy shared key.
	LegacyClass string
}

// ActionRequest describes the action being attempted and the resource it
// targets. Resource may be empty when the action does not address one.
type ActionRequest struct {
	// Name is the tool name or HTTP action label (for audit).
	Name string
	// Kind is the privilege class of the action.
	Kind ActionKind
	// Resource is an opaque project/resource ID when the action targets one.
	Resource string
	// Scopes that may satisfy this action (any one is sufficient).
	Scopes []string
}

// Decision is the result of an authorization evaluation.
type Decision struct {
	Allow   bool
	Reason  string
	Code    string // stable machine-readable reason code
}

// roleActionAllowed is the role × action privilege matrix (Spec 16 §4).
// Roles never widen a key scope; scopes never widen a role.
var roleActionAllowed = map[string]map[ActionKind]bool{
	RoleReader: {
		ActionRead: true,
	},
	RoleDeveloper: {
		ActionRead:     true,
		ActionValidate: true,
		ActionMutate:   true,
	},
	// security-operator may run validation and read; policy mutation requires
	// a dedicated admin scope plus administrator (or an approved delegated
	// permission). Spec 16 keeps that restrictive for the first implementation.
	RoleSecurityOperator: {
		ActionRead:     true,
		ActionValidate: true,
	},
	RoleAdministrator: {
		ActionRead:     true,
		ActionValidate: true,
		ActionMutate:   true,
		ActionAdmin:    true,
	},
}

// Decide evaluates Spec 11 §4.4:
//
//	allow(P, K, A, R) =
//	    authenticated(K)
//	    AND not_revoked(K)
//	    AND scope_allows(K.scopes, A)
//	    AND role_allows(P.role, A)
//	    AND resource_policy_allows(P, A, R)
//
// Missing or unknown authorization inputs deny by default.
func Decide(c Caller, a ActionRequest) Decision {
	// Authenticated: caller must carry a principal and credential ID.
	if c.PrincipalID == "" || c.CredentialID == "" {
		return Decision{Allow: false, Reason: "missing authenticated principal", Code: ReasonUnauthenticated}
	}

	// Unknown action denies.
	if a.Kind != ActionRead && a.Kind != ActionValidate && a.Kind != ActionMutate && a.Kind != ActionAdmin {
		return Decision{Allow: false, Reason: "unknown action class", Code: ReasonUnknownAction}
	}
	if len(a.Scopes) == 0 {
		return Decision{Allow: false, Reason: "action declares no scopes", Code: ReasonUnknownAction}
	}

	// Unknown role denies.
	if c.Role == "" || !KnownRoles[c.Role] {
		return Decision{Allow: false, Reason: "missing or unknown role", Code: ReasonUnknownRole}
	}

	// Unknown scope on the credential denies (empty catalog membership).
	for _, s := range c.Scopes {
		if s == "" || !KnownScopes[s] {
			return Decision{Allow: false, Reason: "credential has empty or unknown scope", Code: ReasonUnknownScope}
		}
	}
	if len(c.Scopes) == 0 {
		return Decision{Allow: false, Reason: "credential has no scopes", Code: ReasonMissingScope}
	}

	// scope_allows: at least one required scope is present on the credential.
	if !scopeAllows(c.Scopes, a.Scopes) {
		return Decision{Allow: false, Reason: "credential scope does not allow action", Code: ReasonMissingScope}
	}

	// role_allows: authorize by permission class, not role-name string compare.
	if !roleActionAllowed[c.Role][a.Kind] {
		return Decision{Allow: false, Reason: "role does not allow action", Code: ReasonRoleDenied}
	}

	// resource_policy_allows: unknown/missing grants deny when a resource is
	// targeted; a principal with no grants cannot act on any named resource.
	if !resourceAllows(c.Resources, a.Resource) {
		return Decision{Allow: false, Reason: "resource policy does not allow action", Code: ReasonResourceDenied}
	}

	return Decision{Allow: true, Reason: ReasonAllowed, Code: ReasonAllowed}
}

func scopeAllows(have, need []string) bool {
	set := make(map[string]bool, len(have))
	for _, s := range have {
		set[s] = true
	}
	for _, n := range need {
		if set[n] {
			return true
		}
	}
	return false
}

// resourceAllows reports whether the principal's resource grants cover the
// requested resource. An empty requested resource is a non-resource-scoped
// action and is allowed once scope and role pass. A named resource requires
// an explicit grant or a wildcard grant.
func resourceAllows(grants []string, resource string) bool {
	if resource == "" {
		return true
	}
	for _, g := range grants {
		if g == "*" || g == resource {
			return true
		}
	}
	return false
}

// RequiredScopeForKind maps an action kind and transport surface to the scope
// catalog entries that may authorize it. A role never widens a key scope; a
// key never widens a role.
func RequiredScopeForKind(kind ActionKind, surface string) []string {
	switch surface {
	case "mcp":
		switch kind {
		case ActionRead:
			return []string{ScopeMCPRead}
		case ActionValidate:
			return []string{ScopeMCPValidate}
		case ActionMutate:
			return []string{ScopeMCPMutate}
		case ActionAdmin:
			return []string{ScopeAdminManage}
		}
	case "ide":
		// All /ide/ validation routes sit behind ide:validate.
		return []string{ScopeIDEValidate}
	default: // rest
		switch kind {
		case ActionRead:
			return []string{ScopeRESTRead}
		case ActionValidate:
			return []string{ScopeRESTRead, ScopeMCPValidate}
		case ActionMutate:
			return []string{ScopeRESTWrite}
		case ActionAdmin:
			return []string{ScopeAdminManage}
		}
	}
	return nil
}

// MCP tool action table. Unknown tool names deny.
var mcpToolActions = map[string]ActionKind{
	// read
	"guardrail_get_context":        ActionRead,
	"guardrail_team_list":          ActionRead,
	"guardrail_team_config_get":    ActionRead,
	"guardrail_team_health":        ActionRead,
	"guardrail_advisor_list":       ActionRead,
	"guardrail_advisor_query":      ActionRead,
	"guardrail_check_policy":       ActionRead,
	"guardrail_classify_content":   ActionRead,
	"list_webhooks":                ActionRead,
	"get_webhook_deliveries":       ActionRead,
	"get_budget_status":            ActionRead,
	"list_budgets":                 ActionRead,
	"get_budget_history":           ActionRead,
	"get_agent_state":              ActionRead,
	"list_agent_sessions":          ActionRead,

	// validate
	"guardrail_validate_bash":               ActionValidate,
	"guardrail_validate_file_edit":          ActionValidate,
	"guardrail_validate_git_operation":      ActionValidate,
	"guardrail_pre_work_check":              ActionValidate,
	"guardrail_validate_scope":              ActionValidate,
	"guardrail_validate_commit":             ActionValidate,
	"guardrail_prevent_regression":          ActionValidate,
	"guardrail_check_test_prod_separation":  ActionValidate,
	"guardrail_validate_push":               ActionValidate,
	"guardrail_verify_file_read":            ActionValidate,
	"guardrail_validate_three_strikes":      ActionValidate,
	"guardrail_validate_exact_replacement":  ActionValidate,
	"guardrail_check_uncertainty":           ActionValidate,
	"guardrail_check_halt_conditions":       ActionValidate,
	"guardrail_validate_production_first":   ActionValidate,
	"guardrail_detect_feature_creep":        ActionValidate,
	"guardrail_verify_fixes_intact":         ActionValidate,
	"test_webhook":                          ActionValidate,

	// mutate
	"guardrail_init_session":          ActionMutate,
	"guardrail_record_file_read":      ActionMutate,
	"guardrail_record_attempt":        ActionMutate,
	"guardrail_reset_attempts":        ActionMutate,
	"guardrail_record_halt":           ActionMutate,
	"guardrail_acknowledge_halt":      ActionMutate,
	"guardrail_team_init":             ActionMutate,
	"guardrail_team_config_update":    ActionMutate,
	"guardrail_team_assign":           ActionMutate,
	"guardrail_team_remove":           ActionMutate,
	"guardrail_install_skills":        ActionMutate,
	"configure_webhook":               ActionMutate,
	"delete_webhook":                  ActionMutate,
	"configure_budget":                ActionMutate,
	"delete_budget":                   ActionMutate,
	"create_agent_session":            ActionMutate,
	"transition_agent_state":          ActionMutate,

	// admin (destructive / security-sensitive)
	"guardrail_project_delete": ActionAdmin,
	"force_agent_state":        ActionAdmin,
}

// SensitiveMCPTools are security-sensitive mutations that require a durable
// audit record before the effect proceeds (fail closed).
var SensitiveMCPTools = map[string]bool{
	"guardrail_project_delete": true,
	"force_agent_state":        true,
	"guardrail_team_config_update": true,
	"guardrail_team_assign":       true,
	"guardrail_team_remove":       true,
	"delete_webhook":              true,
	"delete_budget":               true,
	"transition_agent_state":      true,
	"guardrail_team_init":         true,
}

// MCPToolAction returns the action class for a tool name. Unknown tools deny.
func MCPToolAction(name string) (ActionKind, bool) {
	k, ok := mcpToolActions[name]
	return k, ok
}

// ClassifyMCPTool builds an ActionRequest for a tool call, extracting a safe
// resource ID from allowlisted argument keys. Unknown tools produce an
// unknown-action request that Decide will deny.
func ClassifyMCPTool(name string, args map[string]interface{}) ActionRequest {
	kind, known := MCPToolAction(name)
	if !known {
		return ActionRequest{Name: name, Kind: "unknown", Scopes: nil}
	}
	return ActionRequest{
		Name:     name,
		Kind:     kind,
		Resource: SafeResourceID(args),
		Scopes:   RequiredScopeForKind(kind, "mcp"),
	}
}

// SafeResourceIDKeys are the only argument keys whose values may be used as a
// resource identifier.
var SafeResourceIDKeys = []string{
	"project_id", "project", "project_name", "project_slug",
	"session_id", "document_id", "rule_id", "resource_id",
	"review_id", "budget_id", "webhook_id", "agent_session_id",
}

// SafeResourceID extracts an opaque identifier from allowlisted argument keys.
func SafeResourceID(args map[string]interface{}) string {
	if args == nil {
		return ""
	}
	for _, key := range SafeResourceIDKeys {
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

// IsSensitiveMCPTool reports whether a tool is a security-sensitive mutation.
func IsSensitiveMCPTool(name string) bool {
	return SensitiveMCPTools[name]
}

// ---------------------------------------------------------------------------
// Legacy containment (Spec 16 §4 / R16-07)
// ---------------------------------------------------------------------------

// LegacyPrincipal returns the constrained server-controlled principal for a
// legacy shared key. Legacy principals never map to administrator and never
// carry mutate/admin scopes. Identity is stable and attributable, but the
// key is shared by one legacy workload.
func LegacyPrincipal(class string) (Caller, bool) {
	switch class {
	case "mcp":
		return Caller{
			PrincipalID:  "legacy:mcp",
			CredentialID: "legacy-mcp-key",
			Scopes:       []string{ScopeMCPRead, ScopeMCPValidate, ScopeRESTRead},
			Role:         RoleDeveloper,
			Resources:    []string{"*"},
			LegacyClass:  "mcp",
		}, true
	case "ide":
		return Caller{
			PrincipalID:  "legacy:ide",
			CredentialID: "legacy-ide-key",
			Scopes:       []string{ScopeIDEValidate},
			Role:         RoleDeveloper,
			Resources:    []string{"*"},
			LegacyClass:  "ide",
		}, true
	}
	return Caller{}, false
}

// LegacyRoute is one row of the explicit method/path allowlist. There is no
// safe-method wildcard and no /ide/ prefix wildcard.
type LegacyRoute struct {
	Method string
	// Path is an exact match when Prefix is false; otherwise Path is a prefix.
	Path   string
	Prefix bool
	// Class is the legacy key class allowed ("mcp" or "ide").
	Class string
	// Kind is the action class this route maps to under the privilege model.
	Kind ActionKind
	// ResourceFromPath, when true, uses the trailing path segment as resource.
	ResourceFromPath bool
}

// legacyAllowlist is the approved method/path/action table for legacy keys
// once a credential registry is configured (Spec 16 §4 legacy mapping).
//
//	mcp: named read/validate MCP-tool REST surface
//	ide: named IDE validation routes
//
// Neither class may invoke mutations beyond this table, admin tools, team
// assignment, forced state, or config/secret resources.
var legacyAllowlist = []LegacyRoute{
	// legacy MCP — named read/validate
	{Method: http.MethodGet, Path: "/api/stats", Prefix: true, Class: "mcp", Kind: ActionRead},
	{Method: http.MethodGet, Path: "/api/documents", Prefix: true, Class: "mcp", Kind: ActionRead},
	{Method: http.MethodGet, Path: "/api/documents/search", Class: "mcp", Kind: ActionRead},
	{Method: http.MethodGet, Path: "/api/documents/", Prefix: true, Class: "mcp", Kind: ActionRead, ResourceFromPath: true},
	{Method: http.MethodGet, Path: "/api/rules", Class: "mcp", Kind: ActionRead},
	{Method: http.MethodGet, Path: "/api/rules/", Prefix: true, Class: "mcp", Kind: ActionRead, ResourceFromPath: true},
	{Method: http.MethodGet, Path: "/api/rules/sync/status", Class: "mcp", Kind: ActionRead},
	{Method: http.MethodGet, Path: "/api/projects", Class: "mcp", Kind: ActionRead},
	{Method: http.MethodGet, Path: "/api/projects/", Prefix: true, Class: "mcp", Kind: ActionRead, ResourceFromPath: true},
	{Method: http.MethodGet, Path: "/api/failures", Class: "mcp", Kind: ActionRead},
	{Method: http.MethodGet, Path: "/api/failures/", Prefix: true, Class: "mcp", Kind: ActionRead, ResourceFromPath: true},
	{Method: http.MethodGet, Path: "/api/ingest/status", Class: "mcp", Kind: ActionRead},
	{Method: http.MethodGet, Path: "/api/ingest/orphans", Class: "mcp", Kind: ActionRead},
	{Method: http.MethodGet, Path: "/api/updates/status", Class: "mcp", Kind: ActionRead},
	{Method: http.MethodGet, Path: "/version", Class: "mcp", Kind: ActionRead},
	{Method: http.MethodPost, Path: "/api/updates/check", Class: "mcp", Kind: ActionValidate},

	// legacy IDE — named IDE validation routes
	{Method: http.MethodGet, Path: "/ide/health", Class: "ide", Kind: ActionRead},
	{Method: http.MethodGet, Path: "/ide/rules", Class: "ide", Kind: ActionRead},
	{Method: http.MethodGet, Path: "/ide/quick-reference", Class: "ide", Kind: ActionRead},
	{Method: http.MethodPost, Path: "/ide/validate/file", Class: "ide", Kind: ActionValidate},
	{Method: http.MethodPost, Path: "/ide/validate/selection", Class: "ide", Kind: ActionValidate},
}

// MatchLegacyRoute reports whether a legacy request is on the approved
// method/path table for its key class. Unlisted methods and paths deny —
// there is no safe-method or /ide/ prefix wildcard.
func MatchLegacyRoute(class, method, requestPath string) (LegacyRoute, bool) {
	// Normalize duplicate slashes and trailing dot segments lightly; do not
	// treat "…/." or "…/%2e" tricks as distinct from the clean path.
	p := normalizePath(requestPath)
	for _, r := range legacyAllowlist {
		if r.Class != class || !strings.EqualFold(r.Method, method) {
			continue
		}
		if r.Prefix {
			if strings.HasPrefix(p, r.Path) {
				return r, true
			}
			continue
		}
		if p == r.Path {
			return r, true
		}
	}
	return LegacyRoute{}, false
}

func normalizePath(p string) string {
	if p == "" {
		return "/"
	}
	// Collapse duplicate slashes.
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	// Strip a single trailing slash (except root) so /api/rules/ == /api/rules
	// only when the table row is exact; prefix rows keep their slash.
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		// keep trailing slash for prefix matching of /api/xxx/:id
		// exact rows will not match /api/rules/ which is correct.
	}
	return p
}

// LegacyRESTAction builds the ActionRequest for an allowlisted legacy REST
// request. Returns ok=false when the request is not on the table.
func LegacyRESTAction(class, method, requestPath string) (ActionRequest, bool) {
	r, ok := MatchLegacyRoute(class, method, requestPath)
	if !ok {
		return ActionRequest{Name: method + " " + requestPath, Kind: "unknown"}, false
	}
	resource := ""
	if r.ResourceFromPath {
		resource = lastSegment(requestPath)
	}
	surface := "rest"
	if r.Class == "ide" || strings.HasPrefix(normalizePath(requestPath), "/ide/") {
		surface = "ide"
	}
	return ActionRequest{
		Name:     method + " " + normalizePath(requestPath),
		Kind:     r.Kind,
		Resource: resource,
		Scopes:   RequiredScopeForKind(r.Kind, surface),
	}, true
}

// ClassifyREST builds an ActionRequest for a registered credential's REST
// mutation or read. Mutations on projects are admin-class.
func ClassifyREST(method, requestPath string) ActionRequest {
	p := normalizePath(requestPath)
	name := method + " " + p
	resource := restResourceFromPath(p)

	// IDE surface.
	if strings.HasPrefix(p, "/ide/") {
		kind := ActionValidate
		if method == http.MethodGet {
			kind = ActionRead
		}
		return ActionRequest{
			Name: name, Kind: kind, Resource: resource,
			Scopes: RequiredScopeForKind(kind, "ide"),
		}
	}

	// Admin / destructive.
	if method == http.MethodDelete && strings.HasPrefix(p, "/api/projects/") {
		return ActionRequest{
			Name: name, Kind: ActionAdmin, Resource: resource,
			Scopes: RequiredScopeForKind(ActionAdmin, "rest"),
		}
	}

	switch method {
	case http.MethodGet, http.MethodHead:
		return ActionRequest{
			Name: name, Kind: ActionRead, Resource: resource,
			Scopes: RequiredScopeForKind(ActionRead, "rest"),
		}
	case http.MethodPost:
		// updates/check is a validate-class check.
		if p == "/api/updates/check" || strings.HasPrefix(p, "/api/v1/policy/") {
			return ActionRequest{
				Name: name, Kind: ActionValidate, Resource: resource,
				Scopes: RequiredScopeForKind(ActionValidate, "rest"),
			}
		}
		return ActionRequest{
			Name: name, Kind: ActionMutate, Resource: resource,
			Scopes: RequiredScopeForKind(ActionMutate, "rest"),
		}
	case http.MethodPut, http.MethodPatch, http.MethodDelete:
		return ActionRequest{
			Name: name, Kind: ActionMutate, Resource: resource,
			Scopes: RequiredScopeForKind(ActionMutate, "rest"),
		}
	}
	// Unknown method denies.
	return ActionRequest{Name: name, Kind: "unknown"}
}

// restResourceFromPath pulls an opaque ID from /api/{collection}/{id}.
func restResourceFromPath(p string) string {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	// api, collection, id
	if len(parts) >= 3 && parts[0] == "api" {
		id := parts[2]
		// skip non-ID subpaths
		switch id {
		case "search", "sync", "status", "orphans", "check", "upload":
			return ""
		}
		if id != "" && len(id) <= 128 {
			return id
		}
	}
	return ""
}

func lastSegment(p string) string {
	p = strings.TrimRight(p, "/")
	i := strings.LastIndex(p, "/")
	if i < 0 || i == len(p)-1 {
		return ""
	}
	seg := p[i+1:]
	if len(seg) > 128 {
		return ""
	}
	return seg
}

// LegacyMCPCallAllowed applies the same privilege model to legacy MCP key
// calls: only named read/validate tools, never mutate/admin.
func LegacyMCPCallAllowed(class, toolName string) (ActionRequest, bool) {
	if class != "mcp" {
		return ActionRequest{Name: toolName, Kind: "unknown"}, false
	}
	kind, known := MCPToolAction(toolName)
	if !known || (kind != ActionRead && kind != ActionValidate) {
		return ActionRequest{Name: toolName, Kind: "unknown"}, false
	}
	return ActionRequest{
		Name:   toolName,
		Kind:   kind,
		Scopes: RequiredScopeForKind(kind, "mcp"),
	}, true
}

// FormatDenyError returns a stable, non-leaking denial message.
func FormatDenyError() string {
	return "permission denied"
}

// HTTPStatusForDecision maps a decision to 401 (unauthenticated) or 403
// (authenticated but forbidden).
func HTTPStatusForDecision(d Decision) int {
	if d.Code == ReasonUnauthenticated {
		return http.StatusUnauthorized
	}
	return http.StatusForbidden
}

// Describe returns a short audit-safe description of a caller.
func (c Caller) Describe() string {
	return fmt.Sprintf("%s/%s", c.PrincipalID, c.CredentialID)
}
