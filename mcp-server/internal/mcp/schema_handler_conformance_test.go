package mcp

// Spec 21 — Tool schema ↔ handler conformance gate.
//
// This test enumerates the tools this package registers (from the source of
// every []mcp.Tool-returning function and method, i.e. the registry as
// compiled — never a hand-maintained list) and compares each tool's
// advertised InputSchema to the argument keys its handler actually reads
// (the dispatch switch case in dispatchToolCall / VisionToolSet.dispatch plus
// every package-local helper those cases pass the args map to).
//
// It fails on any drift not present in testdata/tool_contract_drift_allowlist.json,
// and also fails if the allowlist contains an entry that no longer drifts
// (stale allowlist). Per spec 21 R21.3 every allowlisted drift names a tracked
// reason. Per R21.5 there is no skip, no continue-on-error and no threshold.
//
// Drift here is a real, measured defect: the published schemas are known to be
// unreliable (see docs/mcp-server/tools-reference.md, "The published schemas
// are not trustworthy"). The gate makes that measurable and blocks new drift.

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type schemaFacts struct {
	props    map[string]bool
	required map[string]bool
}

type driftEntry struct {
	UndeclaredReads []string `json:"undeclared_reads"`
	DeclaredUnused  []string `json:"declared_unused"`
}

type driftAllowlist struct {
	Reason  string                `json:"reason"`
	Entries map[string]driftEntry `json:"entries"`
}

// parsePackageSource parses every non-test .go file in the current package
// directory. The test runs with cwd = this package directory.
func parsePackageSource(t *testing.T) (*token.FileSet, map[string]*ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	files := map[string]*ast.File{}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, perr := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
		if perr != nil {
			t.Fatalf("parse %s: %v", name, perr)
		}
		files[name] = f
	}
	if len(files) == 0 {
		t.Fatal("parsed no source files; conformance gate cannot run")
	}
	return fset, files
}

func isToolSlice(expr ast.Expr) bool {
	at, ok := expr.(*ast.ArrayType)
	if !ok {
		return false
	}
	sel, ok := at.Elt.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Tool"
}

func parseToolLiteral(cl *ast.CompositeLit) (string, schemaFacts) {
	facts := schemaFacts{props: map[string]bool{}, required: map[string]bool{}}
	name := ""
	for _, el := range cl.Elts {
		kv, ok := el.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch key.Name {
		case "Name":
			if bl, ok := kv.Value.(*ast.BasicLit); ok {
				name, _ = strconv.Unquote(bl.Value)
			}
		case "InputSchema":
			sl, ok := kv.Value.(*ast.CompositeLit)
			if !ok {
				continue
			}
			for _, se := range sl.Elts {
				skv, ok := se.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				sk, _ := skv.Key.(*ast.Ident)
				if sk == nil {
					continue
				}
				if sk.Name == "Properties" {
					if pl, ok := skv.Value.(*ast.CompositeLit); ok {
						for _, pe := range pl.Elts {
							if pkv, ok := pe.(*ast.KeyValueExpr); ok {
								if bl, ok := pkv.Key.(*ast.BasicLit); ok {
									s, _ := strconv.Unquote(bl.Value)
									facts.props[s] = true
								}
							}
						}
					}
				}
				if sk.Name == "Required" {
					if rl, ok := skv.Value.(*ast.CompositeLit); ok {
						for _, re := range rl.Elts {
							if bl, ok := re.(*ast.BasicLit); ok {
								s, _ := strconv.Unquote(bl.Value)
								facts.required[s] = true
							}
						}
					}
				}
			}
		}
	}
	return name, facts
}

// enumerateRegisteredTools returns every tool name declared in any function or
// method that builds a []mcp.Tool, regardless of which activation flag guards
// it. Conditional tools are therefore included in both enabled and disabled
// states (spec 21 R21.4).
func enumerateRegisteredTools(files map[string]*ast.File) map[string]schemaFacts {
	schemas := map[string]schemaFacts{}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok || !isToolSlice(cl.Type) {
				return true
			}
			for _, el := range cl.Elts {
				inner, ok := el.(*ast.CompositeLit)
				if !ok {
					continue
				}
				name, facts := parseToolLiteral(inner)
				if name != "" {
					schemas[name] = facts
				}
			}
			return true
		})
	}
	return schemas
}

func argsReadsInNode(n ast.Node) map[string]bool {
	set := map[string]bool{}
	ast.Inspect(n, func(m ast.Node) bool {
		ix, ok := m.(*ast.IndexExpr)
		if !ok {
			return true
		}
		id, ok := ix.X.(*ast.Ident)
		if !ok || (id.Name != "args" && id.Name != "a" && id.Name != "arguments") {
			return true
		}
		lit, ok := ix.Index.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		if v, err := strconv.Unquote(lit.Value); err == nil {
			set[v] = true
		}
		return true
	})
	return set
}

// funcReads maps each function/method name to the args keys it reads directly.
func funcReads(files map[string]*ast.File) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, f := range files {
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			set := argsReadsInNode(fn.Body)
			if len(set) > 0 {
				for k := range set {
					if out[fn.Name.Name] == nil {
						out[fn.Name.Name] = map[string]bool{}
					}
					out[fn.Name.Name][k] = true
				}
			}
		}
	}
	return out
}

// callsPassingArgs returns names of package-local functions/methods invoked
// with the args map, so reads inside helpers are attributed to the tool.
func callsPassingArgs(n ast.Node) []string {
	var out []string
	ast.Inspect(n, func(m ast.Node) bool {
		call, ok := m.(*ast.CallExpr)
		if !ok {
			return true
		}
		passes := false
		for _, a := range call.Args {
			if id, ok := a.(*ast.Ident); ok && id.Name == "args" {
				passes = true
				break
			}
		}
		if !passes {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			out = append(out, fn.Name)
		case *ast.SelectorExpr:
			out = append(out, fn.Sel.Name)
		}
		return true
	})
	return out
}

// handlerReads walks every switch statement, and for each case whose single
// label is a registered tool name, unions the direct args reads in the case
// body with the args reads of every helper the case calls with args.
func handlerReads(files map[string]*ast.File, schemas map[string]schemaFacts) map[string]map[string]bool {
	freads := funcReads(files)
	toolReads := map[string]map[string]bool{}
	add := func(dst map[string]bool, src map[string]bool) {
		for k := range src {
			dst[k] = true
		}
	}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			sw, ok := n.(*ast.SwitchStmt)
			if !ok {
				return true
			}
			for _, c := range sw.Body.List {
				cc, ok := c.(*ast.CaseClause)
				if !ok || len(cc.List) != 1 {
					continue
				}
				lit, ok := cc.List[0].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				tool, _ := strconv.Unquote(lit.Value)
				if _, registered := schemas[tool]; !registered {
					continue
				}
				if toolReads[tool] == nil {
					toolReads[tool] = map[string]bool{}
				}
				for _, st := range cc.Body {
					add(toolReads[tool], argsReadsInNode(st))
					for _, fn := range callsPassingArgs(st) {
						add(toolReads[tool], freads[fn])
					}
				}
			}
			return true
		})
	}
	return toolReads
}

// computeDrift is the pure drift function; it is unit-tested independently so
// the gate's red-on-drift behaviour is proven without editing the tree.
func computeDrift(schemas map[string]schemaFacts, reads map[string]map[string]bool) map[string]driftEntry {
	drift := map[string]driftEntry{}
	for tool, facts := range schemas {
		read := reads[tool]
		var undeclared, unused []string
		for k := range read {
			if !facts.props[k] {
				undeclared = append(undeclared, k)
			}
		}
		for k := range facts.props {
			if !read[k] {
				unused = append(unused, k)
			}
		}
		if len(undeclared) == 0 && len(unused) == 0 {
			continue
		}
		sort.Strings(undeclared)
		sort.Strings(unused)
		if undeclared == nil {
			undeclared = []string{}
		}
		if unused == nil {
			unused = []string{}
		}
		drift[tool] = driftEntry{UndeclaredReads: undeclared, DeclaredUnused: unused}
	}
	return drift
}

func loadAllowlist(t *testing.T) driftAllowlist {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "tool_contract_drift_allowlist.json"))
	if err != nil {
		t.Fatalf("read drift allowlist: %v", err)
	}
	var a driftAllowlist
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatalf("parse drift allowlist: %v", err)
	}
	if a.Entries == nil {
		a.Entries = map[string]driftEntry{}
	}
	if strings.TrimSpace(a.Reason) == "" {
		t.Fatal("drift allowlist has no reason; spec 21 R21.3 requires a tracked reason")
	}
	return a
}

func eqStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestToolSchemaHandlerConformance is the blocking gate (spec 21 R21.1-R21.5).
func TestToolSchemaHandlerConformance(t *testing.T) {
	_, files := parsePackageSource(t)
	schemas := enumerateRegisteredTools(files)
	if len(schemas) == 0 {
		t.Fatal("enumerated zero registered tools; enumeration is not live")
	}
	reads := handlerReads(files, schemas)
	drift := computeDrift(schemas, reads)
	allow := loadAllowlist(t)

	// Every drifted tool must be allowlisted with an exactly-matching entry.
	for tool, d := range drift {
		entry, ok := allow.Entries[tool]
		if !ok {
			t.Errorf("schema/handler drift in %q is not allowlisted: undeclared_reads=%v declared_unused=%v",
				tool, d.UndeclaredReads, d.DeclaredUnused)
			continue
		}
		if !eqStrings(d.UndeclaredReads, entry.UndeclaredReads) || !eqStrings(d.DeclaredUnused, entry.DeclaredUnused) {
			t.Errorf("allowlist for %q is stale: actual undeclared_reads=%v declared_unused=%v, allowlist=%v/%v",
				tool, d.UndeclaredReads, d.DeclaredUnused, entry.UndeclaredReads, entry.DeclaredUnused)
		}
	}
	// A stale allowlist entry (no longer drifting) is also a failure: the
	// allowlist must describe today's tree, not yesterday's.
	for tool := range allow.Entries {
		if _, still := drift[tool]; !still {
			t.Errorf("allowlist entry %q no longer drifts; remove it so the gate stays honest", tool)
		}
	}

	t.Logf("enumerated %d registered tools; %d drifting, %d allowlisted",
		len(schemas), len(drift), len(allow.Entries))
}

// TestToolSchemaHandlerConformance_DetectsInjectedDrift proves the gate is
// red on injected drift (spec 21 R21.6): a handler read absent from the schema
// and a declared property the handler never reads must both be reported.
func TestToolSchemaHandlerConformance_DetectsInjectedDrift(t *testing.T) {
	schemas := map[string]schemaFacts{
		"injected_tool": {props: map[string]bool{"declared_only": true}, required: map[string]bool{}},
	}
	reads := map[string]map[string]bool{
		"injected_tool": {"undeclared_read": true, "declared_only": false},
	}
	drift := computeDrift(schemas, reads)
	d, ok := drift["injected_tool"]
	if !ok {
		t.Fatal("gate failed to detect injected drift")
	}
	if !eqStrings(d.UndeclaredReads, []string{"undeclared_read"}) {
		t.Fatalf("undeclared reads = %v, want [undeclared_read]", d.UndeclaredReads)
	}
	if !eqStrings(d.DeclaredUnused, []string{"declared_only"}) {
		t.Fatalf("declared-unused = %v, want [declared_only]", d.DeclaredUnused)
	}
}
