// Package mutationkill is a hermetic mutation-kill harness for the Phase 0
// security matrix (Spec 19 R19-04).
//
// It proves the negative suites actually detect a control bypass: for each
// mutation it copies the module to a scratch directory, confirms the real
// negative suite PASSES against the protected source, deterministically
// injects a bypass into the copy, and then requires the same suite to FAIL.
// A suite that still passes against an injected bypass is reported as a failed
// mutation kill.
//
// The scratch copy guarantees the protected tree is never left mutated: the
// fixture lifecycle required by R19-04 ("restored before publishing PASS") is
// satisfied structurally, because the mutation never touches the working tree.
package mutationkill

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/thearchitectit/guardrail-mcp"

// mutationCase describes one deterministic injected bypass and the existing
// negative suite that must detect it.
type mutationCase struct {
	name string
	// file is the module-relative source file holding the control.
	file string
	// old must occur exactly once in file; it is replaced by new.
	old string
	new string
	// pkg and testRegex select the existing negative suite.
	pkg       string
	testRegex string
}

var mutationCases = []mutationCase{
	{
		name: "webhook-ssrf-guard-always-permits",
		file: "internal/mcp/tools_notifications.go",
		old: `func checkWebhookIP(ip net.IP, host string) error {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() {
		return fmt.Errorf("refusing webhook to non-public address %s for host %q", ip, host)
	}
	return nil
}`,
		new: `func checkWebhookIP(ip net.IP, host string) error {
	_ = ip
	_ = host
	return nil
}`,
		pkg:       "./internal/mcp",
		testRegex: "^TestValidateWebhookURL$",
	},
	{
		name: "registry-failclosed-swallows-load-error",
		file: "internal/auth/registry.go",
		old: `	records, err := readRecords(inlineJSON, filePath)
	if err != nil {
		return nil, err
	}
	return New(verifierKey, records)`,
		new: `	records, _ := readRecords(inlineJSON, filePath)
	return New(verifierKey, records)`,
		pkg:       "./internal/auth",
		testRegex: "^TestLoadFromSourcesConfiguredInvalidFailsClosed$",
	},
	{
		name: "auth-bypass-always-permits",
		file: "internal/auth/authz.go",
		old: `func Decide(c Caller, a ActionRequest) Decision {
	// Authenticated: caller must carry a principal and credential ID.
	if c.PrincipalID == "" || c.CredentialID == "" {
		return Decision{Allow: false, Reason: "missing authenticated principal", Code: ReasonUnauthenticated}
	}`,
		new: `func Decide(c Caller, a ActionRequest) Decision {
	return Decision{Allow: true, Reason: ReasonAllowed, Code: ReasonAllowed}
	// Authenticated: caller must carry a principal and credential ID.
	if c.PrincipalID == "" || c.CredentialID == "" {
		return Decision{Allow: false, Reason: "missing authenticated principal", Code: ReasonUnauthenticated}
	}`,
		pkg:       "./internal/auth",
		testRegex: "^TestDecideScopeRoleResourceIntersection$",
	},
	{
		name: "mcp-authz-tools-call-always-authorizes",
		file: "internal/mcp/server.go",
		old: `func (s *MCPServer) authorizeToolCall(ctx context.Context, name string, args map[string]interface{}) *mcp.CallToolResult {
	caller, ok := callerFromContext(ctx)`,
		new: `func (s *MCPServer) authorizeToolCall(ctx context.Context, name string, args map[string]interface{}) *mcp.CallToolResult {
	_, _ = ctx, args
	return nil
}

func (s *MCPServer) authorizeToolCallDisabled(ctx context.Context, name string, args map[string]interface{}) *mcp.CallToolResult {
	caller, ok := callerFromContext(ctx)`,
		pkg:       "./internal/mcp",
		testRegex: "^TestStreamableHTTPToolsCallAuthorizesAndDenies$",
	},
	{
		name: "secret-fixture-leaks-into-output",
		file: "internal/mcp/resource_registration.go",
		old:  `configJSON, err := json.MarshalIndent(s.config.PublicView(), "", "  ")`,
		new:  `configJSON, err := json.MarshalIndent(s.config, "", "  ")`,
		pkg:       "./internal/mcp",
		testRegex: "^TestSecretLeakRedactionAcrossRealPaths$",
	},
}

// findModuleRoot walks up from the test working directory to the directory
// holding the mcp-server go.mod.
func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && strings.Contains(string(data), "module "+modulePath) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate module root (looking for module %s)", modulePath)
		}
		dir = parent
	}
}

// copyModule recursively copies src into dst.
func copyModule(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		// Never descend into VCS metadata.
		if d.IsDir() && d.Name() == ".git" {
			return fs.SkipDir
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, in); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("copy module %s -> %s: %v", src, dst, err)
	}
}

// runGoTest runs `go test <pkg> -run <regex> -count=1` inside dir and returns
// the combined output and the command error (nil on PASS).
func runGoTest(dir, pkg, regex string) (string, error) {
	cmd := exec.Command("go", "test", pkg, "-run", regex, "-count=1")
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// TestNegativeSuitesKillInjectedBypass is the R19-04 mutation-kill control.
func TestNegativeSuitesKillInjectedBypass(t *testing.T) {
	if testing.Short() {
		t.Skip("mutation harness compiles the module; skipped under -short")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("go toolchain not on PATH: %v", err)
	}
	root := findModuleRoot(t)

	for _, mc := range mutationCases {
		mc := mc
		t.Run(mc.name, func(t *testing.T) {
			modDir := filepath.Join(t.TempDir(), "mcp-server")
			copyModule(t, root, modDir)

			// Baseline: the real negative suite must PASS against protected
			// source, otherwise the kill result would be meaningless.
			out, err := runGoTest(modDir, mc.pkg, mc.testRegex)
			if err != nil {
				t.Fatalf("baseline suite must pass before mutation, got error %v:\n%s", err, out)
			}

			// Inject the bypass deterministically.
			path := filepath.Join(modDir, filepath.FromSlash(mc.file))
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", mc.file, err)
			}
			// Normalize CRLF so the mutation site matches regardless of the
			// checkout's autocrlf setting; Go is indifferent to line endings.
			src := strings.ReplaceAll(string(data), "\r\n", "\n")
			if n := strings.Count(src, mc.old); n != 1 {
				t.Fatalf("mutation site in %s matched %d times, want exactly 1", mc.file, n)
			}
			mutated := strings.Replace(src, mc.old, mc.new, 1)
			if err := os.WriteFile(path, []byte(mutated), 0o644); err != nil {
				t.Fatalf("write mutated %s: %v", mc.file, err)
			}

			// The same suite must now FAIL: the control is proven to detect the
			// bypass rather than passing regardless.
			out, err = runGoTest(modDir, mc.pkg, mc.testRegex)
			if err == nil {
				t.Fatalf("negative suite still PASSED against the injected bypass; mutation %q was not killed.\n%s", mc.name, out)
			}
			t.Logf("mutation %q killed: suite flipped PASS->FAIL\n%s", mc.name, firstLines(out, 12))
		})
	}
}

// firstLines trims captured output for readable logs.
func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-n)
}

// TestDocsGateCatchesBrokenInternalLink is the docs-gate half of R19-04
// (R19-11). The control is the real checked-in link checker
// (scripts/check-doc-links.sh), invoked here exactly as CI invokes it. A clean
// tree must pass; injecting a broken internal Markdown link must flip the same
// checker to FAIL.
//
// The script is fed on stdin with the scratch tree as the working directory so
// the check is portable across the Linux CI runner and a Windows host (WSL
// bash does not translate Windows-style path arguments).
func TestDocsGateCatchesBrokenInternalLink(t *testing.T) {
	if testing.Short() {
		t.Skip("docs-gate mutation harness spawns a shell; skipped under -short")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skipf("bash not on PATH; docs-gate checker cannot be exercised here: %v", err)
	}
	scriptPath := filepath.Join(filepath.Dir(findModuleRoot(t)), "scripts", "check-doc-links.sh")
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read docs-gate script %s: %v", scriptPath, err)
	}

	scratch := t.TempDir()
	writeDoc := func(name, body string) {
		if err := os.WriteFile(filepath.Join(scratch, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	writeDoc("a.md", "[good](./b.md)\n[external](https://example.com/x.md)\n")
	writeDoc("b.md", "target\n")

	// Baseline: a clean tree must pass, otherwise the kill result is vacuous.
	if out, err := runDocsGate(bash, scratch, script); err != nil {
		t.Fatalf("baseline docs gate must pass on a clean tree, got %v:\n%s", err, out)
	}

	// Inject the broken internal link (the bypass).
	writeDoc("a.md", "[broken](./missing.md)\n")

	out, err := runDocsGate(bash, scratch, script)
	if err == nil {
		t.Fatalf("docs gate PASSED against a broken internal link; mutation %q was not killed.\n%s",
			"broken-internal-doc-link", out)
	}
	t.Logf("docs-gate mutation killed: broken link flipped PASS->FAIL\n%s", firstLines(out, 8))
}

// runDocsGate runs the docs-gate link checker with dir as its working directory,
// passing the script body on stdin. A non-nil error means the gate reported a
// finding (or failed to run).
func runDocsGate(bash, dir string, script []byte) (string, error) {
	cmd := exec.Command(bash, "-s")
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(script)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}
