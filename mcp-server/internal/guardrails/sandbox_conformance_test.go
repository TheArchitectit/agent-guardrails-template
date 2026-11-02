package guardrails

import (
	"context"
	"strings"
	"testing"
)

func TestRunSandboxEscapeConformance_AllFourTiersCertify(t *testing.T) {
	mgr := NewSandboxManager(DefaultSandboxConfig(), nil)
	report, err := RunSandboxEscapeConformance(context.Background(), mgr, DefaultSandboxEscapeFixtures())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.Certified {
		t.Fatalf("expected all four tiers to certify, failures: %v", report.Failures)
	}
	if !strings.HasPrefix(report.FixtureDigest, "sha256:") {
		t.Fatalf("expected sha256 fixture digest, got %q", report.FixtureDigest)
	}
	if len(report.Tiers) != 4 {
		t.Fatalf("expected 4 tiers in report, got %d", len(report.Tiers))
	}
	for _, tc := range report.Tiers {
		if !tc.Certified || tc.EscapesSucceeded != 0 || tc.FixturesBlocked != 4 {
			t.Fatalf("tier %s failed certification: %+v", tc.Tier, tc)
		}
	}
}

func TestRunSandboxEscapeConformance_FailsWhenEscapeSucceeds(t *testing.T) {
	mgr := NewSandboxManager(DefaultSandboxConfig(), nil)
	fixtures := DefaultSandboxEscapeFixtures()
	// Replace the restricted network fixture with an allowed host ("github.com:443")
	// to simulate an escape that is NOT blocked by the allowlist.
	for i := range fixtures {
		if fixtures[i].Tier == TierRestricted && fixtures[i].Class == EscapeUnauthorizedNetwork {
			fixtures[i].TargetHost = "github.com:443"
		}
	}

	report, err := RunSandboxEscapeConformance(context.Background(), mgr, fixtures)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Certified {
		t.Fatal("expected certification to fail when a network escape fixture reaches an allowed host")
	}
	if len(report.Failures) == 0 {
		t.Fatal("expected failure reason in report")
	}
}

func TestRunSandboxEscapeConformance_FailsWhenClassMissing(t *testing.T) {
	mgr := NewSandboxManager(DefaultSandboxConfig(), nil)
	var incomplete []SandboxEscapeFixture
	for _, f := range DefaultSandboxEscapeFixtures() {
		if f.Tier == TierEphemeral && f.Class == EscapeUndeclaredSecret {
			continue
		}
		incomplete = append(incomplete, f)
	}

	report, err := RunSandboxEscapeConformance(context.Background(), mgr, incomplete)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Certified {
		t.Fatal("expected certification to fail when a tier is missing a required escape class")
	}
}
