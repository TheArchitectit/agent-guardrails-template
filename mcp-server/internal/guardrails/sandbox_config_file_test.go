package guardrails

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec 03 Sec. 3.2/3.3 — the documented sandbox configuration file.
//
// The defect these tests exist for: `DefaultSandboxConfig()` was the ONLY
// SandboxConfig in the tree. No file, no environment variable, no caller. An
// operator who wrote the YAML in Sec. 3.2/3.3 changed nothing, and the artifact
// is a security boundary — `network: {enabled: false}` that silently does not
// apply produces a review trail asserting a control the process never had.
//
// These tests drive the REAL loader and then the REAL manager, so a config that
// parses but does not bind fails here rather than in production.

func writeTempConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sandbox.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

// documentedSandboxYAML is a verbatim copy of the shape Spec 03 Sec. 3.2 and
// Sec. 3.3 document, including the policy-level `read_write_paths` /
// `read_only_paths` placement that a naive `resource_limits`-only decoder
// refuses as an unknown field.
const documentedSandboxYAML = `
sandbox:
  enabled: true
  fallback_enabled: false
  image: "alpine:3.19"
  global_defaults:
    cpu:
      max_percent: 50
      max_cores: 2
    memory:
      max_mb: 1024
      swap_allowed: false
    disk:
      max_write_mb: 500
      read_only_paths:
        - "/etc"
        - "/usr"
      read_write_paths:
        - "/tmp"
        - "/workspace"
    network:
      enabled: false
      allowed_hosts: []
    time:
      max_seconds: 300
      max_wall_seconds: 600
  sandbox_policies:
    bash:
      default_level: "L1"
      resource_limits:
        cpu:
          max_percent: 50
        memory:
          max_mb: 512
        network:
          enabled: false
      read_write_paths:
        - "/tmp"
        - "/workspace"
      read_only_paths:
        - "/etc"
        - "/usr"
    git:
      default_level: "L1"
      resource_limits:
        cpu:
          max_percent: 25
        memory:
          max_mb: 256
        network:
          enabled: true
          allowed_hosts: ["github.com", "gitlab.com"]
      read_write_paths:
        - "/workspace/.git"
        - "/workspace"
    file_edit:
      default_level: "L0"
      resource_limits:
        cpu:
          max_percent: 10
        memory:
          max_mb: 128
`

// TestDocumentedSandboxShapeLoads proves the spec's own example is the input
// the loader accepts, not a shape it refuses.
func TestDocumentedSandboxShapeLoads(t *testing.T) {
	path := writeTempConfig(t, documentedSandboxYAML)
	cfg, err := LoadSandboxConfigFromYAML(path)
	if err != nil {
		t.Fatalf("the documented Sec. 3.2/3.3 shape did not load: %v", err)
	}

	if !cfg.Enabled {
		t.Errorf("enabled = false, want true")
	}
	// `fallback_enabled: false` is explicit and must win over the shipped true.
	if cfg.FallbackEnabled {
		t.Errorf("fallback_enabled = true; an explicit false was ignored, so the chain may still downgrade to L0")
	}
	if cfg.Image != "alpine:3.19" {
		t.Errorf("image = %q, want alpine:3.19", cfg.Image)
	}
	if cfg.GlobalDefaults.Memory.MaxMB != 1024 || cfg.GlobalDefaults.Memory.SwapAllowed {
		t.Errorf("global memory = %d MB swap=%v, want 1024 and swap refused",
			cfg.GlobalDefaults.Memory.MaxMB, cfg.GlobalDefaults.Memory.SwapAllowed)
	}
	if cfg.GlobalDefaults.Time.MaxWallSeconds != 600 {
		t.Errorf("global max_wall_seconds = %d, want 600", cfg.GlobalDefaults.Time.MaxWallSeconds)
	}

	if got, want := strings.Join(cfg.ToolPolicyNames(), ","), "bash,file_edit,git"; got != want {
		t.Fatalf("declared policies = %q, want %q", got, want)
	}

	bash, ok := cfg.ToolPolicies["bash"]
	if !ok {
		t.Fatal("bash policy missing")
	}
	if bash.DefaultLevel != LevelL1 {
		t.Errorf("bash level = %q, want L1", bash.DefaultLevel)
	}
	if bash.ResourceLimits.Memory.MaxMB != 512 {
		t.Errorf("bash max_mb = %d, want 512", bash.ResourceLimits.Memory.MaxMB)
	}
	// The policy-level path set is the documented placement; if only the nested
	// spelling were supported these would be the inherited globals instead.
	if got := strings.Join(bash.ResourceLimits.Disk.ReadWritePaths, ","); got != "/tmp,/workspace" {
		t.Errorf("bash read_write_paths = %q, want the policy-level set", got)
	}
	if got := strings.Join(bash.ResourceLimits.Disk.ReadOnlyPaths, ","); got != "/etc,/usr" {
		t.Errorf("bash read_only_paths = %q, want the policy-level set", got)
	}
	if bash.ResourceLimits.Network.Enabled {
		t.Errorf("bash network enabled; the policy declares enabled: false")
	}

	git, ok := cfg.ToolPolicies["git"]
	if !ok {
		t.Fatal("git policy missing")
	}
	if !git.ResourceLimits.Network.Enabled {
		t.Errorf("git network disabled; the policy declares enabled: true")
	}
	if got := strings.Join(git.ResourceLimits.Network.AllowedHosts, ","); got != "github.com,gitlab.com" {
		t.Errorf("git allowed_hosts = %q", got)
	}
	// git's policy-level set differs from the global set (/tmp,/workspace), so
	// this assertion can only pass if the policy-level spelling is actually
	// read. Asserting only `bash` would not: its documented paths happen to
	// equal the globals, and the mutation that ignores the policy-level key
	// survived exactly that check.
	if got := strings.Join(git.ResourceLimits.Disk.ReadWritePaths, ","); got != "/workspace/.git,/workspace" {
		t.Errorf("git read_write_paths = %q, want the policy-level set (globals are /tmp,/workspace)", got)
	}
}

// TestLoadedPolicyBindsAtExecution is the test that would have failed for every
// prior "config exists" landing: parsing is not control. It drives the REAL
// SandboxManager built from the loaded config and reads back the limits the
// executor will actually use.
func TestLoadedPolicyBindsAtExecution(t *testing.T) {
	path := writeTempConfig(t, documentedSandboxYAML)
	cfg, err := LoadSandboxConfigFromYAML(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	m := NewSandboxManager(cfg, slog.Default())

	active := m.Config()
	level, limits := active.ResolveToolPolicy("bash")
	if level != LevelL1 {
		t.Errorf("resolved bash level = %q, want L1", level)
	}
	if limits.Memory.MaxMB != 512 {
		t.Errorf("resolved bash max_mb = %d, want the policy's 512 (not the global 1024)", limits.Memory.MaxMB)
	}
	if limits.Time.MaxWallSeconds != 600 {
		t.Errorf("resolved bash max_wall_seconds = %d, want the inherited global 600", limits.Time.MaxWallSeconds)
	}

	// A tool type the file did not declare gets the global defaults — never a
	// zero deadline, which used to mean "killed before it started".
	_, unknown := active.ResolveToolPolicy("undeclared_tool")
	if unknown.Time.MaxSeconds != 300 {
		t.Errorf("undeclared tool max_seconds = %d, want the global 300", unknown.Time.MaxSeconds)
	}
}

// TestSandboxConfigRefusals covers every way the file can look configured while
// constraining nothing.
func TestSandboxConfigRefusals(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "no sandbox section",
			body: "sandbox_config:\n",
			want: "declares no sandbox section",
		},
		{
			name: "empty file",
			body: "",
			want: "is empty",
		},
		{
			name: "unknown key",
			body: "sandbox:\n  enabled: true\n  max_wall_second: 600\n",
			want: "max_wall_second",
		},
		{
			// An unknown TOP-level key is refused by the same rule, before the
			// missing-section check can run. Either refusal is a refusal to
			// start; the point is that a typo is never silently discarded.
			name: "unknown top level key",
			body: "other: 1\n",
			want: "field other not found",
		},
		{
			name: "both policy tables",
			body: "sandbox:\n  tool_policies: {}\n  sandbox_policies: {}\n",
			want: "both `tool_policies` and `sandbox_policies`",
		},
		{
			name: "both sandbox keys",
			body: "sandbox:\n  enabled: true\nsandbox_config:\n  enabled: true\n",
			want: "declares sandbox twice",
		},
		{
			name: "policy without a level",
			body: "sandbox:\n  sandbox_policies:\n    bash:\n      resource_limits:\n        memory:\n          max_mb: 128\n",
			want: "declares no default_level",
		},
		{
			name: "unknown level",
			body: "sandbox:\n  sandbox_policies:\n    bash:\n      default_level: \"L9\"\n",
			want: "unknown default_level",
		},
		{
			name: "empty tool type",
			body: "sandbox:\n  sandbox_policies:\n    \"\":\n      default_level: \"L1\"\n",
			want: "empty tool type",
		},
		{
			name: "malformed mount path",
			body: "sandbox:\n  sandbox_policies:\n    bash:\n      default_level: \"L1\"\n      read_write_paths: [\"/tmp:ro\"]\n",
			want: "must not contain ':' or ','",
		},
		{
			name: "path declared in both places",
			body: `sandbox:
  sandbox_policies:
    bash:
      default_level: "L1"
      resource_limits:
        disk:
          read_write_paths: ["/tmp"]
      read_write_paths: ["/workspace"]
`,
			want: "both at the policy level and under resource_limits.disk",
		},
		{
			name: "out of range cpu",
			body: "sandbox:\n  global_defaults:\n    cpu:\n      max_percent: 400\n",
			want: "max_cpu_percent must be 0-100",
		},
		{
			name: "invalid image",
			body: "sandbox:\n  image: \"alpine latest\"\n",
			want: "must not contain whitespace",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTempConfig(t, tc.body)
			_, err := LoadSandboxConfigFromYAML(path)
			if err == nil {
				t.Fatalf("config loaded with no error; want a refusal naming %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to name %q", err.Error(), tc.want)
			}
		})
	}
}

// TestAbsentSwitchInheritsDefault proves absence and explicit-false are
// distinguishable. Decoding straight into the Go struct collapses both to the
// zero value, so a file naming only `max_mb` would silently disable the
// fallback chain it never mentioned.
func TestAbsentSwitchInheritsDefault(t *testing.T) {
	path := writeTempConfig(t, "sandbox:\n  global_defaults:\n    memory:\n      max_mb: 256\n")
	cfg, err := LoadSandboxConfigFromYAML(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !cfg.Enabled {
		t.Errorf("enabled = false; an omitted switch took the zero value instead of the shipped default")
	}
	if !cfg.FallbackEnabled {
		t.Errorf("fallback_enabled = false; an omitted switch took the zero value instead of the shipped default")
	}
	if cfg.GlobalDefaults.Memory.MaxMB != 256 {
		t.Errorf("max_mb = %d, want 256", cfg.GlobalDefaults.Memory.MaxMB)
	}
	// Absent numeric fields keep their shipped defaults rather than becoming 0.
	if cfg.GlobalDefaults.Time.MaxSeconds != 300 {
		t.Errorf("max_seconds = %d, want the shipped 300", cfg.GlobalDefaults.Time.MaxSeconds)
	}
	if cfg.GlobalDefaults.Memory.SwapAllowed {
		t.Errorf("swap_allowed = true; the shipped default is false and the file did not name it")
	}
}

// TestDeclaredPolicyTableReplacesShipped proves deletion is a deletion. Merging
// into the shipped table would keep running a policy the operator removed.
func TestDeclaredPolicyTableReplacesShipped(t *testing.T) {
	path := writeTempConfig(t, "sandbox:\n  sandbox_policies:\n    only_tool:\n      default_level: \"L2\"\n")
	cfg, err := LoadSandboxConfigFromYAML(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := strings.Join(cfg.ToolPolicyNames(), ","); got != "only_tool" {
		t.Errorf("policies = %q, want only the declared one; the shipped bash/git/file_edit survived a replacement", got)
	}
	if cfg.ToolPolicies["only_tool"].DefaultLevel != LevelL2 {
		t.Errorf("level = %q, want L2", cfg.ToolPolicies["only_tool"].DefaultLevel)
	}
}

// TestImageReferencesAccepted is the regression for the check this pass found:
// Validate() refused EVERY valid OCI reference — including the shipped
// default's documented values — because it treated the ':' separator as an
// argv split. That was invisible while no file could set the field.
func TestImageReferencesAccepted(t *testing.T) {
	for _, image := range []string{
		"alpine:latest",
		"alpine:3.19",
		"ghcr.io/org/img:v1",
		"registry.example.com:5000/team/img:v1.2.3",
		"alpine@sha256:" + strings.Repeat("a", 64),
	} {
		t.Run(image, func(t *testing.T) {
			cfg := DefaultSandboxConfig()
			cfg.Image = image
			if err := cfg.Validate(); err != nil {
				t.Errorf("a valid image reference was refused: %v", err)
			}
		})
	}
}

// TestImageReferencesRefused pins what must still be refused — the values that
// change the command line rather than naming an image.
func TestImageReferencesRefused(t *testing.T) {
	for _, image := range []string{
		"alpine latest",
		"alpine\tlatest",
		" alpine",
		"-rm",
		"alpine;rm -rf /",
		"$(id)",
	} {
		t.Run(image, func(t *testing.T) {
			cfg := DefaultSandboxConfig()
			cfg.Image = image
			if err := cfg.Validate(); err == nil {
				t.Errorf("image %q validated clean", image)
			}
		})
	}
}

// TestLoaderRefusesPathTraversalAtLoad proves a malformed mount path is a load
// error, not a first-execution error: a config discovered to be broken in
// production is a config that ran in production.
func TestLoaderRefusesPathTraversalAtLoad(t *testing.T) {
	path := writeTempConfig(t, "sandbox:\n  sandbox_policies:\n    bash:\n      default_level: \"L1\"\n      read_write_paths: [\"/tmp,/etc\"]\n")
	if _, err := LoadSandboxConfigFromYAML(path); err == nil {
		t.Fatal("a comma-bearing mount path loaded clean")
	}
}
