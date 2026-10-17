package web

import (
	"net/http"
	"strings"
)

// ExposureClass is the declared exposure of a route (Spec 12 Sec. 3.1).
//
// Every route is assigned exactly one class in the single registry below, which
// is the source of truth for BOTH authentication (APIKeyAuth) and rate limiting
// (RateLimitMiddleware). Before this registry existed each middleware carried a
// hand-copied list of skip rules; the two lists could drift, and a route added
// to one was not automatically added to the other.
type ExposureClass string

const (
	// ExposurePublic — reachable with no credential, on the declared methods
	// only.
	ExposurePublic ExposureClass = "public"
	// ExposureAuthenticated — requires a valid bearer key; no anonymous access.
	// This is also the deny-by-default class for a route that is not declared
	// public (Spec 12 Sec. 3.1 rule 1), so adding a route cannot accidentally make
	// it public.
	ExposureAuthenticated ExposureClass = "authenticated"
)

// staticAssetExtensions is the reviewed set of file extensions served as public
// static assets. It is a declared list, not an inline heuristic scattered
// across two middlewares: the suffix rule applies only to genuine static
// fetches (GET/HEAD, never under /api/) and every entry is a browser-loadable
// asset type the Web UI ships.
var staticAssetExtensions = []string{
	".js", ".css", ".html", ".svg", ".png", ".jpg", ".ico", ".woff", ".woff2", ".ttf",
}

// exposureRule is one declared entry in the single exposure registry. A rule
// matches when every non-empty constraint matches.
type exposureRule struct {
	name    string
	class   ExposureClass
	methods []string // empty = any method

	// patternExact / patternPrefix constrain the matched Echo route pattern
	// (c.Path()), which is what a route declaration is about.
	patternExact  string
	patternPrefix string

	// pathExact / pathPrefix constrain the actual request path.
	pathExact  string
	pathPrefix string

	// assetSuffix marks the declared static-asset rule: GET/HEAD only, never
	// under /api/, and only for a declared asset extension.
	assetSuffix bool

	// staticMount marks a file-serving catch-all route (echo.Static). Such a
	// route is declared here so a route enumeration can account for it, but it
	// never classifies a request by its pattern: a request through it is
	// classified by the request-path rules, and an undeclared path under /api/
	// stays authenticated.
	staticMount bool

	rationale string
}

// exposureRegistry is the single source of truth for both middlewares
// (Spec 12 Sec. 3.1 rule 2). Ordered: first match wins. Rules reproduce the exact
// pre-registry behaviour (Spec 12 Sec. 5: "Registry first … with behavior identical
// to today"); tightening the public set is a separate, reviewed change.
var exposureRegistry = []exposureRule{
	// --- Anonymous operational endpoints (Sec. 3.2, current posture) ------------
	{name: "health-live", class: ExposurePublic, patternExact: "/health/live",
		rationale: "Container/orchestrator liveness probe; no secret material."},
	{name: "health-ready", class: ExposurePublic, patternExact: "/health/ready",
		rationale: "Container/orchestrator readiness probe; already omits failing components."},
	{name: "metrics", class: ExposurePublic, patternExact: "/metrics",
		rationale: "Prometheus scrape. Spec 12 Sec. 3.2 proposes `authenticated` in the public profile; that tightening is the owner decision recorded in Sec. 7, so the current posture declares it public."},
	{name: "api-docs", class: ExposurePublic, patternExact: "/docs",
		rationale: "Schema disclosure is low risk in dev; Sec. 3.2 proposes authenticated in production."},
	{name: "openapi", class: ExposurePublic, patternExact: "/openapi.yaml",
		rationale: "Same posture as /docs."},
	{name: "version", class: ExposurePublic, methods: []string{http.MethodGet, http.MethodOptions}, patternExact: "/version",
		rationale: "No secret material; used by clients."},

	// --- Web UI shell and static assets (Sec. 3.1 rule 3) -----------------------
	{name: "web-root", class: ExposurePublic, patternExact: "/",
		rationale: "Web UI shell; the browser must load it before auth."},
	{name: "web-index", class: ExposurePublic, patternExact: "/index.html",
		rationale: "Web UI shell."},
	{name: "web-spa-fallback", class: ExposurePublic, patternExact: "/web/*",
		rationale: "SPA shell fallback for client-side routes; data routes stay authenticated."},
	{name: "static-root-mount", class: ExposurePublic, patternExact: "/*", staticMount: true,
		rationale: "echo.Static(\"/\", …) file mount. Declared so route enumeration accounts for it; a request through it is classified by the request-path rules below, and an undeclared path under /api/ stays authenticated."},
	{name: "static-prefix-pattern", class: ExposurePublic, patternPrefix: "/static/",
		rationale: "Declared static asset mount."},

	// --- Static request paths (Sec. 3.1 rule 3: explicit, not a suffix guess) ---
	{name: "static-prefix-path", class: ExposurePublic, pathPrefix: "/static/",
		rationale: "Declared static asset prefix."},
	{name: "web-prefix-path", class: ExposurePublic, pathPrefix: "/web/",
		rationale: "Web UI assets and SPA routes under the /web mount."},
	{name: "assets-prefix-path", class: ExposurePublic, pathPrefix: "/assets/",
		rationale: "Declared asset prefix."},
	{name: "js-prefix-path", class: ExposurePublic, pathPrefix: "/js/",
		rationale: "Declared asset prefix."},
	{name: "css-prefix-path", class: ExposurePublic, pathPrefix: "/css/",
		rationale: "Declared asset prefix."},
	{name: "web-exact-path", class: ExposurePublic, pathExact: "/web",
		rationale: "SPA entry point."},
	{name: "index-exact-path", class: ExposurePublic, pathExact: "/index.html",
		rationale: "Web UI shell."},
	{name: "root-exact-path", class: ExposurePublic, pathExact: "/",
		rationale: "Web UI shell."},
	{name: "static-asset-suffix", class: ExposurePublic, assetSuffix: true,
		rationale: "GET/HEAD of a declared asset extension outside /api/. The extension list is reviewed; the rule is method- and prefix-constrained, so an asset-shaped /api/ path is never public (the old unbounded suffix exemption is what made POST /api/ingest/x.js reach its handler)."},

	// --- Declared public read-only API (Sec. 3.2, current posture) --------------
	// Sec. 3.2 proposes moving these to `authenticated` unless a named anonymous
	// consumer exists. That re-decision is the owner call recorded in Sec. 7; the
	// registry is the single place it will be applied.
	{name: "read-documents", class: ExposurePublic, methods: []string{http.MethodGet, http.MethodOptions}, patternExact: "/api/documents",
		rationale: "Public read-only browsing (current posture)."},
	{name: "read-documents-search", class: ExposurePublic, methods: []string{http.MethodGet, http.MethodOptions}, patternExact: "/api/documents/search",
		rationale: "Public read-only browsing (current posture)."},
	{name: "read-documents-by-id", class: ExposurePublic, methods: []string{http.MethodGet, http.MethodOptions}, patternPrefix: "/api/documents/",
		rationale: "Public read-only browsing (current posture)."},
	{name: "read-rules", class: ExposurePublic, methods: []string{http.MethodGet, http.MethodOptions}, patternExact: "/api/rules",
		rationale: "Public read-only browsing (current posture)."},
	{name: "read-rules-sub", class: ExposurePublic, methods: []string{http.MethodGet, http.MethodOptions}, patternPrefix: "/api/rules/",
		rationale: "Public read-only browsing (current posture); note this also covers GET /api/rules/sync/status."},
	{name: "read-stats", class: ExposurePublic, methods: []string{http.MethodGet, http.MethodOptions}, patternExact: "/api/stats",
		rationale: "Public read-only browsing (current posture)."},
	{name: "read-stats-sub", class: ExposurePublic, methods: []string{http.MethodGet, http.MethodOptions}, patternPrefix: "/api/stats/",
		rationale: "Public read-only browsing (current posture)."},
	{name: "read-projects", class: ExposurePublic, methods: []string{http.MethodGet, http.MethodOptions}, patternExact: "/api/projects",
		rationale: "Public read-only browsing (current posture)."},
	{name: "read-projects-sub", class: ExposurePublic, methods: []string{http.MethodGet, http.MethodOptions}, patternPrefix: "/api/projects/",
		rationale: "Public read-only browsing (current posture)."},
	{name: "read-failures", class: ExposurePublic, methods: []string{http.MethodGet, http.MethodOptions}, patternExact: "/api/failures",
		rationale: "Public read-only browsing (current posture)."},
	{name: "read-failures-sub", class: ExposurePublic, methods: []string{http.MethodGet, http.MethodOptions}, patternPrefix: "/api/failures/",
		rationale: "Public read-only browsing (current posture)."},
	{name: "read-ingest-status", class: ExposurePublic, methods: []string{http.MethodGet, http.MethodOptions}, patternExact: "/api/ingest/status",
		rationale: "Public operational status (current posture)."},
	{name: "read-ingest-orphans", class: ExposurePublic, methods: []string{http.MethodGet, http.MethodOptions}, patternExact: "/api/ingest/orphans",
		rationale: "Public operational status (current posture)."},
	{name: "read-updates-status", class: ExposurePublic, methods: []string{http.MethodGet, http.MethodOptions}, patternExact: "/api/updates/status",
		rationale: "Public operational status (current posture)."},
}

func (r exposureRule) allowsMethod(method string) bool {
	if len(r.methods) == 0 {
		return true
	}
	for _, m := range r.methods {
		if m == method {
			return true
		}
	}
	return false
}

// matches reports whether this rule classifies a request. A staticMount rule
// never classifies a request (see the field comment).
func (r exposureRule) matches(method, pattern, requestPath string) bool {
	if r.staticMount {
		return false
	}
	if !r.allowsMethod(method) {
		return false
	}
	if r.patternExact != "" && pattern != r.patternExact {
		return false
	}
	if r.patternPrefix != "" && !strings.HasPrefix(pattern, r.patternPrefix) {
		return false
	}
	if r.pathExact != "" && requestPath != r.pathExact {
		return false
	}
	if r.pathPrefix != "" && !strings.HasPrefix(requestPath, r.pathPrefix) {
		return false
	}
	if r.assetSuffix {
		if strings.HasPrefix(requestPath, "/api/") {
			return false
		}
		if method != http.MethodGet && method != http.MethodHead {
			return false
		}
		if !hasStaticAssetSuffix(requestPath) {
			return false
		}
	}
	return true
}

// hasStaticAssetSuffix reports whether path ends with a declared asset
// extension.
func hasStaticAssetSuffix(path string) bool {
	for _, ext := range staticAssetExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// ExposureFor returns the declared exposure class for a request. Both the Echo
// route pattern (c.Path()) and the actual request path are consulted, exactly
// as the pre-registry middlewares did; the difference is that the rules now
// live in one table. A route that is not declared public is authenticated
// (deny by default, Spec 12 Sec. 3.1 rule 1).
func ExposureFor(method, pattern, requestPath string) ExposureClass {
	if rule, ok := lookupRule(method, pattern, requestPath); ok {
		return rule.class
	}
	return ExposureAuthenticated
}

// lookupRule returns the first declared rule that classifies the request.
func lookupRule(method, pattern, requestPath string) (exposureRule, bool) {
	for _, r := range exposureRegistry {
		if r.matches(method, pattern, requestPath) {
			return r, true
		}
	}
	return exposureRule{}, false
}

// IsDeclaredPublic reports whether a route is explicitly declared public in the
// registry. It is the "written decision" half of Spec 12 Sec. 3.1 rule 3: a route
// is public only because a rule says so, never because of an inherited
// heuristic. Static file mounts are not public by pattern (their requests are
// classified by request path), so they report false here.
func IsDeclaredPublic(method, pattern string) bool {
	for _, r := range exposureRegistry {
		if r.staticMount || r.class != ExposurePublic {
			continue
		}
		// A route is declared public only when the pattern alone decides it;
		// rules that need a request path are covered by ExposureFor.
		if r.assetSuffix || r.pathExact != "" || r.pathPrefix != "" {
			continue
		}
		if r.matches(method, pattern, pattern) {
			return true
		}
	}
	return false
}

// IsDeclaredStaticMount reports whether a catch-all route pattern is a declared
// file-serving mount (echo.Static). Such a route is accounted for in the
// registry even though it never classifies a request by pattern.
func IsDeclaredStaticMount(pattern string) bool {
	for _, r := range exposureRegistry {
		if r.staticMount && r.patternExact == pattern {
			return true
		}
	}
	return false
}
