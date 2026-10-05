package auth

import (
	"net/http"
	"testing"
)

func callerFor(role string, scopes, resources []string) Caller {
	return Caller{
		PrincipalID:  "p-1",
		CredentialID: "c-1",
		Scopes:       scopes,
		Role:         role,
		Resources:    resources,
	}
}

// TestDecideScopeRoleResourceIntersection covers Spec 11 section 4.4:
// allow = authenticated AND scope_allows AND role_allows AND resource_allows.
// Missing/unknown inputs deny.
func TestDecideScopeRoleResourceIntersection(t *testing.T) {
	readAction := ActionRequest{Name: "read", Kind: ActionRead, Scopes: []string{ScopeRESTRead}}
	validateAction := ActionRequest{Name: "validate", Kind: ActionValidate, Scopes: []string{ScopeMCPValidate}}
	mutateAction := ActionRequest{Name: "mutate", Kind: ActionMutate, Scopes: []string{ScopeRESTWrite}}
	adminAction := ActionRequest{Name: "delete", Kind: ActionAdmin, Resource: "proj-a", Scopes: []string{ScopeAdminManage}}

	cases := []struct {
		name    string
		caller  Caller
		action  ActionRequest
		want    bool
		reason  string
	}{
		{
			name:   "reader + rest:read allows read",
			caller: callerFor(RoleReader, []string{ScopeRESTRead}, []string{"*"}),
			action: readAction,
			want:   true,
		},
		{
			name:   "reader + rest:read denies mutate",
			caller: callerFor(RoleReader, []string{ScopeRESTRead}, []string{"*"}),
			action: mutateAction,
			want:   false,
			reason: ReasonMissingScope,
		},
		{
			name:   "mcp:read-only key + admin role denies project delete",
			caller: callerFor(RoleAdministrator, []string{ScopeMCPRead}, []string{"*"}),
			action: adminAction,
			want:   false,
			reason: ReasonMissingScope,
		},
		{
			name:   "mcp:mutate key + reader role denies mutation",
			caller: callerFor(RoleReader, []string{ScopeMCPMutate}, []string{"*"}),
			action: ActionRequest{Name: "m", Kind: ActionMutate, Scopes: []string{ScopeMCPMutate}},
			want:   false,
			reason: ReasonRoleDenied,
		},
		{
			name:   "developer + rest:write allows mutate",
			caller: callerFor(RoleDeveloper, []string{ScopeRESTWrite}, []string{"*"}),
			action: mutateAction,
			want:   true,
		},
		{
			name:   "developer + admin:manage denies admin",
			caller: callerFor(RoleDeveloper, []string{ScopeAdminManage}, []string{"*"}),
			action: adminAction,
			want:   false,
			reason: ReasonRoleDenied,
		},
		{
			name:   "administrator + admin:manage + grant allows delete",
			caller: callerFor(RoleAdministrator, []string{ScopeAdminManage}, []string{"proj-a"}),
			action: adminAction,
			want:   true,
		},
		{
			name:   "cross-project deny even with confirmation-class grant",
			caller: callerFor(RoleAdministrator, []string{ScopeAdminManage}, []string{"proj-a"}),
			action: ActionRequest{Name: "delete", Kind: ActionAdmin, Resource: "proj-b", Scopes: []string{ScopeAdminManage}},
			want:   false,
			reason: ReasonResourceDenied,
		},
		{
			name:   "wildcard resource grant allows any",
			caller: callerFor(RoleAdministrator, []string{ScopeAdminManage}, []string{"*"}),
			action: ActionRequest{Name: "delete", Kind: ActionAdmin, Resource: "proj-b", Scopes: []string{ScopeAdminManage}},
			want:   true,
		},
		{
			name:   "missing role denies",
			caller: callerFor("", []string{ScopeRESTRead}, []string{"*"}),
			action: readAction,
			want:   false,
			reason: ReasonUnknownRole,
		},
		{
			name:   "unknown role denies",
			caller: callerFor("superuser", []string{ScopeRESTRead}, []string{"*"}),
			action: readAction,
			want:   false,
			reason: ReasonUnknownRole,
		},
		{
			name:   "unknown scope on credential denies",
			caller: callerFor(RoleAdministrator, []string{"mcp"}, []string{"*"}),
			action: readAction,
			want:   false,
			reason: ReasonUnknownScope,
		},
		{
			name:   "empty scopes deny",
			caller: callerFor(RoleAdministrator, nil, []string{"*"}),
			action: readAction,
			want:   false,
			reason: ReasonMissingScope,
		},
		{
			name:   "unauthenticated denies",
			caller: Caller{},
			action: readAction,
			want:   false,
			reason: ReasonUnauthenticated,
		},
		{
			name:   "unknown action kind denies",
			caller: callerFor(RoleAdministrator, []string{ScopeAdminManage}, []string{"*"}),
			action: ActionRequest{Name: "x", Kind: "weird", Scopes: []string{ScopeAdminManage}},
			want:   false,
			reason: ReasonUnknownAction,
		},
		{
			name:   "empty resource grant denies named resource",
			caller: callerFor(RoleAdministrator, []string{ScopeAdminManage}, nil),
			action: adminAction,
			want:   false,
			reason: ReasonResourceDenied,
		},
		{
			name:   "security-operator denies admin even with admin scope role check",
			caller: callerFor(RoleSecurityOperator, []string{ScopeAdminManage}, []string{"*"}),
			action: adminAction,
			want:   false,
			reason: ReasonRoleDenied,
		},
		{
			name:   "security-operator allows validate",
			caller: callerFor(RoleSecurityOperator, []string{ScopeMCPValidate}, []string{"*"}),
			action: validateAction,
			want:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := Decide(tc.caller, tc.action)
			if d.Allow != tc.want {
				t.Fatalf("Decide allow=%v want %v (reason %s)", d.Allow, tc.want, d.Code)
			}
			if !tc.want && tc.reason != "" && d.Code != tc.reason {
				t.Fatalf("reason=%s want %s", d.Code, tc.reason)
			}
		})
	}
}

// TestClassifyMCPToolUnknownDenies ensures unknown tool names never map to an
// authorized action.
func TestClassifyMCPToolUnknownDenies(t *testing.T) {
	a := ClassifyMCPTool("not_a_tool", nil)
	d := Decide(callerFor(RoleAdministrator, []string{ScopeAdminManage}, []string{"*"}), a)
	if d.Allow {
		t.Fatal("unknown tool must deny")
	}
}

// TestLegacyMCPCallAllowedNamedOnly covers Spec 16 section 4: legacy MCP key
// is limited to named read/validate tools and never mutates or administers.
func TestLegacyMCPCallAllowedNamedOnly(t *testing.T) {
	caller, ok := LegacyPrincipal("mcp")
	if !ok {
		t.Fatal("legacy mcp principal required")
	}
	cases := []struct {
		tool string
		want bool
	}{
		{"guardrail_get_context", true},
		{"guardrail_validate_bash", true},
		{"guardrail_init_session", false},
		{"guardrail_project_delete", false},
		{"force_agent_state", false},
		{"guardrail_team_assign", false},
		{"no_such_tool", false},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			action, ok := LegacyMCPCallAllowed(caller.LegacyClass, tc.tool)
			if !ok {
				if tc.want {
					t.Fatalf("%s should be on the legacy named table", tc.tool)
				}
				return
			}
			d := Decide(caller, action)
			if d.Allow != tc.want {
				t.Fatalf("%s: allow=%v want %v", tc.tool, d.Allow, tc.want)
			}
		})
	}
}

// TestMatchLegacyRouteNoWildcards replaces the old any-GET /ide/ prefix
// behaviour: unlisted methods and paths deny.
func TestMatchLegacyRouteNoWildcards(t *testing.T) {
	cases := []struct {
		class, method, path string
		want                bool
	}{
		{"mcp", http.MethodGet, "/api/stats", true},
		{"mcp", http.MethodGet, "/api/secrets", false},
		{"mcp", http.MethodHead, "/api/stats", false},
		{"mcp", http.MethodOptions, "/api/stats", false},
		{"mcp", http.MethodPost, "/api/ingest", false},
		{"mcp", http.MethodPost, "/api/ingest/sync", false},
		{"mcp", http.MethodPost, "/api/updates/check", true},
		{"mcp", http.MethodPost, "/ide/validate/file", false},
		{"ide", http.MethodPost, "/ide/validate/file", true},
		{"ide", http.MethodPost, "/ide/validate/selection", true},
		{"ide", http.MethodPost, "/ide/validate/other", false},
		{"ide", http.MethodGet, "/ide/health", true},
		{"ide", http.MethodGet, "/ide/secret", false},
		{"ide", http.MethodGet, "/api/stats", false},
	}
	for _, tc := range cases {
		t.Run(tc.class+" "+tc.method+" "+tc.path, func(t *testing.T) {
			if _, ok := MatchLegacyRoute(tc.class, tc.method, tc.path); ok != tc.want {
				t.Fatalf("MatchLegacyRoute(%s,%s,%s)=%v want %v", tc.class, tc.method, tc.path, ok, tc.want)
			}
		})
	}
}

// TestClassifyRESTActions documents method/path → action classification used
// before every REST effect.
func TestClassifyRESTActions(t *testing.T) {
	cases := []struct {
		method, path string
		kind         ActionKind
	}{
		{http.MethodGet, "/api/rules", ActionRead},
		{http.MethodPost, "/api/rules", ActionMutate},
		{http.MethodPost, "/api/updates/check", ActionValidate},
		{http.MethodDelete, "/api/projects/p1", ActionAdmin},
		{http.MethodPut, "/api/projects/p1", ActionMutate},
		{http.MethodPost, "/ide/validate/file", ActionValidate},
	}
	for _, tc := range cases {
		a := ClassifyREST(tc.method, tc.path)
		if a.Kind != tc.kind {
			t.Fatalf("ClassifyREST(%s,%s)=%s want %s", tc.method, tc.path, a.Kind, tc.kind)
		}
	}
}
