package guardrails

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

// TestNetworkViolationDetection_DoesNotFabricateHosts pins gap-spec 03 Sec.
// 3.4 (violation detection: "connections to non-whitelisted hosts").
//
// The detector is best-effort: it reads the hostnames a tool reported talking
// to and flags any that the policy did not allow. That makes it, like every
// output-derived control in this tree, a potential false-positive generator,
// and here the consequence is severe: a non-empty violation set becomes
// ErrSandboxViolation, so a benign command that merely PRINTED a comment or a
// sentence was reported to the caller as a security breach.
//
// Measured before the fix, through the real SandboxManager, with the SHIPPED
// `git` policy (network enabled, AllowedHosts github.com+gitlab.com):
//
//	"main.go:12: // increment the counter"  => network access to non-allowed host: main.go? no —
//	                                           "// docs" => non-allowed host: docs
//	"next step to: nothing-here"            => non-allowed host: nothing-here
//	"path //server/share/file.txt"          => non-allowed host: server
//
// Two alternations caused it: a bare `//` (comments, UNC paths, division) and
// a bare `to` (ordinary prose). Both are removed; the patterns now require an
// explicit scheme or an explicit connection/resolution phrase, and a candidate
// must look like a hostname (at least one dot, hostname alphabet).
func TestNetworkViolationDetection_DoesNotFabricateHosts(t *testing.T) {
	cfg := DefaultSandboxConfig()
	policyLevel, limits := cfg.ResolveToolPolicy("git")
	if !limits.Network.Enabled || len(limits.Network.AllowedHosts) == 0 {
		t.Fatalf("fixture precondition: the shipped git policy must declare network egress, got enabled=%v hosts=%v",
			limits.Network.Enabled, limits.Network.AllowedHosts)
	}
	// The shipped `git` policy is L1. This host has no usable user namespaces,
	// so an L1 execution stops in the isolation floor before any command runs
	// and produces no output to analyse — a property of the machine, not of
	// the detector. The limits below are the SHIPPED policy's limits; only the
	// level is chosen for what the host can execute, which is exactly what the
	// violation detector inspects.
	t.Logf("shipped git policy: level=%s network=%v hosts=%v", policyLevel, limits.Network.Enabled, limits.Network.AllowedHosts)
	mgr := NewSandboxManager(cfg, slog.Default())
	ctx := context.Background()

	benign := []struct {
		name string
		cmd  string
	}{
		{"go_source_comment", `printf 'main.go:12: // increment the counter\n'`},
		{"go_block_comment", `printf '// build tag: ignore\n'`},
		{"docs_reference", `printf 'see //docs/api for details\n'`},
		{"unc_path", `printf 'path //server/share/file.txt\n'`},
		{"ordinary_prose", `printf 'next step to: nothing-here\n'`},
		{"diff_header", `printf 'diff --git a/x.go b/x.go\n'`},
		{"version_decimal", `printf 'resolved 1.2 and 3.4\n'`},
		{"trailing_slash", `printf 'wrote ./out/\n'`},
	}

	for _, tc := range benign {
		t.Run(tc.name, func(t *testing.T) {
			res, err := mgr.Execute(ctx, tc.cmd, LevelL0, limits)
			if res == nil {
				t.Fatalf("expected a result, got nil (err=%v)", err)
			}
			if len(res.SandboxViolations) != 0 {
				t.Fatalf("benign output produced a security verdict %v — a fabricated violation teaches operators to disable the detector", res.SandboxViolations)
			}
			if err != nil {
				t.Fatalf("benign output returned an error: %v", err)
			}
		})
	}

	// Positive controls: real evidence of egress must still be caught, or the
	// fix would have been "make the detector match nothing".
	real := []struct {
		name string
		cmd  string
		host string
	}{
		{"explicit_https_url", `printf 'GET https://evil.example.com/x -> 403\n'`, "evil.example.com"},
		{"explicit_ssh_url", `printf 'ssh://other.example.net/repo\n'`, "other.example.net"},
		{"resolve_failure", `printf 'curl: (6) Could not resolve host: evil.example.com\n'`, "evil.example.com"},
		{"connection_failure", `printf 'failed to connect to metrics.evil-example.org port 443\n'`, "metrics.evil-example.org"},
	}
	for _, tc := range real {
		t.Run(tc.name, func(t *testing.T) {
			res, err := mgr.Execute(ctx, tc.cmd, LevelL0, limits)
			if res == nil {
				t.Fatalf("expected a result, got nil (err=%v)", err)
			}
			found := false
			for _, v := range res.SandboxViolations {
				if strings.Contains(v, tc.host) {
					found = true
				}
			}
			if !found {
				t.Fatalf("real egress evidence %q was NOT reported; violations=%v", tc.host, res.SandboxViolations)
			}
			if err == nil || !errors.Is(err, ErrSandboxViolation) {
				t.Fatalf("expected ErrSandboxViolation for egress to %s, got %v", tc.host, err)
			}
		})
	}

	// An allowned host is not a violation even when detected, so the allowlist
	// is genuinely consulted rather than the detector merely firing.
	t.Run("allowed_host_is_not_a_violation", func(t *testing.T) {
		res, err := mgr.Execute(ctx, `printf 'clone https://github.com/org/repo\n'`, LevelL0, limits)
		if res == nil {
			t.Fatalf("expected a result, got nil (err=%v)", err)
		}
		if len(res.SandboxViolations) != 0 || err != nil {
			t.Fatalf("an allowlisted host was reported as a violation: %v (err=%v)", res.SandboxViolations, err)
		}
	})
}

// TestNetworkViolationDetection_GatedOnEgressBeingPermitted pins the second
// half of the same requirement.
//
// When `network.enabled` is false the execution runs with `--network=none`.
// There is no egress channel to violate, so a "network access" verdict about
// that execution would describe a control that was never reachable — the same
// defect class as reporting "fork bomb" with no process bound in force
// (gap-spec 03, fixed for the process half on 2026-10-19).
//
// Before this gate the detector keyed off `len(AllowedHosts) > 0` alone, so a
// policy declaring `enabled: false` with a whitelist "if network needed" — a
// shape Sec. 3.2 itself illustrates — was scanned for violations on a
// network=none execution.
func TestNetworkViolationDetection_GatedOnEgressBeingPermitted(t *testing.T) {
	cfg := DefaultSandboxConfig()
	mgr := NewSandboxManager(cfg, slog.Default())
	ctx := context.Background()

	var disabledLimits ResourceLimits
	disabledLimits.Network.Enabled = false
	disabledLimits.Network.AllowedHosts = []string{"github.com"}

	// Direct detector assertion: with egress disabled, the network branch must
	// not run at all.
	probe := &SandboxResult{ExitCode: 0, Stdout: "GET https://evil.example.com/x"}
	mgr.detectViolations(probe, LevelL0, disabledLimits, ctx, nil)
	for _, v := range probe.SandboxViolations {
		if strings.Contains(v, "network access") {
			t.Fatalf("network violation reported with egress disabled: %v", probe.SandboxViolations)
		}
	}

	// The same output under a policy that DOES permit egress must be a
	// violation, so the gate cannot be satisfied by disabling the detector.
	enabledLimits := disabledLimits
	enabledLimits.Network.Enabled = true
	probe2 := &SandboxResult{ExitCode: 0, Stdout: "GET https://evil.example.com/x"}
	mgr.detectViolations(probe2, LevelL0, enabledLimits, ctx, nil)
	if len(probe2.SandboxViolations) == 0 {
		t.Fatal("network violation NOT reported with egress enabled — the gate is on the wrong condition")
	}
}

// TestExtractNetworkTargets_HostnameShapes pins the extractor directly so a
// future pattern change is measured, not guessed.
func TestExtractNetworkTargets_HostnameShapes(t *testing.T) {
	cases := []struct {
		in   string
		want []string
		why  string
	}{
		{"https://evil.example.com/x", []string{"evil.example.com"}, "explicit scheme"},
		{"//evil.example.com/x", nil, "protocol-relative URLs are indistinguishable from comments"},
		{"main.go:12: // increment", nil, "source comment"},
		{"//server/share/f.txt", nil, "UNC path"},
		{"next step to: nothing-here", nil, "ordinary prose"},
		{"a 6/2 = 3", nil, "division"},
		{"Could not resolve host: evil.example.com", []string{"evil.example.com"}, "resolution failure"},
		{"host: docs", nil, "single-label token is a word, not a host"},
		{"connected to localhost:5432", nil, "single-label; and localhost inside a namespace is not egress"},
		{"version 1.2.3", nil, "numeric labels are versions, not hosts"},
		{"https://EVIL.Example.COM./x", []string{"evil.example.com"}, "case + trailing dot normalised"},
		{"again https://evil.example.com/x and https://evil.example.com/y", []string{"evil.example.com"}, "deduplicated"},
	}
	for _, tc := range cases {
		got := extractNetworkTargets(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("extractNetworkTargets(%q) = %v, want %v (%s)", tc.in, got, tc.want, tc.why)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("extractNetworkTargets(%q) = %v, want %v (%s)", tc.in, got, tc.want, tc.why)
			}
		}
	}
}
