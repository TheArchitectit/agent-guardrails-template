package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
)

// Ingress posture policy (Spec 17 R17-02).
//
// R17-02: before starting either listener the process MUST fail startup — a
// nonzero exit and no app socket — on an external bind or publish without an
// approved profile, on `public` without proven TLS termination and
// direct-backend denial, or on `tailnet` without verified private ingress plus
// encrypted authenticated transport. Crucially, it adds: "A true
// `PRODUCTION_MODE` flag, `TLS_ENABLED=true` or presence of certificate paths
// alone never satisfies this gate."
//
// Before this file existed the tree enforced only the *bind* half (R17-01, via
// ValidateDeploymentBinding): a `public` deployment was admitted because
// TRUSTED_PROXIES was non-empty, and a `tailnet` deployment was admitted
// because the bind host was not a wildcard. Neither check says anything about
// TLS termination, backend isolation or transport encryption, so a public
// profile could start on a plaintext, directly reachable backend while every
// other gate stayed green.
//
// The evidence form matters. Spec 17 §4 states plainly that "an operator-
// supplied JSON assertion without an actual external reachability/certificate
// test is NOT evidence". So this gate does not accept a free-text env
// attestation: a nonlocal profile must present a commit-bound preflight record
// that carries the *proven* results of named positive and negative probes. A
// record that is absent, malformed, permissive, for the wrong profile, for the
// wrong commit, missing a required proven result, or carrying no negative probe
// fails closed.

// ingressProbe is one proven result inside a preflight record. A probe with no
// detail is an assertion without evidence, so a proven probe must name what was
// actually observed.
type ingressProbe struct {
	Proven bool   `json:"proven"`
	Detail string `json:"detail"`
}

// ingressRecord is the commit-bound deployment preflight record for a nonlocal
// profile. Field names are part of the contract; an unknown field is a typo in
// evidence and is refused (DisallowUnknownFields).
type ingressRecord struct {
	ContractVersion     string        `json:"contract_version"`
	Profile             string        `json:"profile"`
	Commit              string        `json:"commit"`
	TLSTermination      *ingressProbe `json:"tls_termination"`
	DirectBackendDenied *ingressProbe `json:"direct_backend_denied"`
	PrivateIngress      *ingressProbe `json:"private_ingress"`
	EncryptedTransport  *ingressProbe `json:"encrypted_transport"`
	NegativeProbes      []string      `json:"negative_probes"`
}

// ingressContractVersion is the record schema this build understands. A record
// written for another contract is not silently reinterpreted.
const ingressContractVersion = "guardrail-ingress-preflight/v1"

// buildVCSRevision returns the VCS revision stamped into this binary, or "" when
// the build carries no stamp (for example `go test`). The record's commit is
// always required to be a full revision; the equality check applies only when a
// stamp exists, so a local test run cannot be made to depend on the build host.
func buildVCSRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return s.Value
		}
	}
	return ""
}

// ValidateIngressPosture fails startup when a nonlocal profile cannot prove its
// ingress posture (Spec 17 R17-02). `local` (and the empty-profile default) is
// not constrained here: its exposure is bounded by the R17-01 loopback/bridge
// bind rule instead.
func (c *Config) ValidateIngressPosture() error {
	if c == nil {
		return nil
	}
	profile, err := NormalizeDeploymentProfile(c.DeploymentProfile)
	if err != nil {
		return err
	}
	if profile == DeploymentProfileLocal {
		return nil
	}

	path := strings.TrimSpace(c.DeploymentPreflightFile)
	if path == "" {
		return fmt.Errorf(
			"profile %q requires a commit-bound deployment preflight record (Spec 17 R17-02): set DEPLOYMENT_PREFLIGHT_FILE to a JSON record of the proven TLS-termination, backend-isolation and transport results; PRODUCTION_MODE, TLS_ENABLED and certificate paths alone never satisfy this gate",
			profile)
	}

	record, err := loadIngressRecord(path)
	if err != nil {
		return err
	}

	if record.ContractVersion != ingressContractVersion {
		return fmt.Errorf(
			"deployment preflight record %s declares contract_version %q; this build understands %q (Spec 17 R17-02: evidence is not reinterpreted across contract versions)",
			path, record.ContractVersion, ingressContractVersion)
	}
	if record.Profile != profile {
		return fmt.Errorf(
			"deployment preflight record %s was produced for profile %q but the effective profile is %q (Spec 17 R17-02: contradictory evidence blocks nonlocal readiness)",
			path, record.Profile, profile)
	}
	if !isFullRevision(record.Commit) {
		return fmt.Errorf(
			"deployment preflight record %s must carry a full 40-hex commit revision in \"commit\" (Spec 17 R17-02: the record is commit-bound), got %q",
			path, record.Commit)
	}
	if stamped := buildVCSRevision(); stamped != "" && !strings.EqualFold(stamped, record.Commit) {
		return fmt.Errorf(
			"deployment preflight record %s is bound to commit %s but this binary is commit %s (Spec 17 R17-02: a stale preflight is not evidence)",
			path, record.Commit, stamped)
	}
	if len(record.NegativeProbes) == 0 {
		return fmt.Errorf(
			"deployment preflight record %s carries no negative probes (Spec 17 R17-02: an assertion without an actual reachability/certificate test is NOT evidence)",
			path)
	}
	for i, probe := range record.NegativeProbes {
		if strings.TrimSpace(probe) == "" {
			return fmt.Errorf(
				"deployment preflight record %s has an empty negative probe at index %d (Spec 17 R17-02)", path, i)
		}
	}

	switch profile {
	case DeploymentProfilePublic:
		// Public exposure terminates TLS at a controlled ingress and keeps the
		// backend unreachable directly. Both must be proven; TLS_ENABLED only
		// describes this process, which is behind the proxy, so it is
		// deliberately not consulted.
		if err := requireProven(record.TLSTermination, path, "tls_termination"); err != nil {
			return err
		}
		if err := requireProven(record.DirectBackendDenied, path, "direct_backend_denied"); err != nil {
			return err
		}
	case DeploymentProfileTailnet:
		// Tailnet exposure needs a verified private ingress and an encrypted,
		// authenticated transport (TLS or a reviewed encrypted authenticated
		// tunnel terminating at the same trust boundary).
		if err := requireProven(record.PrivateIngress, path, "private_ingress"); err != nil {
			return err
		}
		if err := requireProven(record.EncryptedTransport, path, "encrypted_transport"); err != nil {
			return err
		}
	}
	return nil
}

// requireProven fails closed on an absent, unproven or unevidenced probe.
func requireProven(p *ingressProbe, path, field string) error {
	if p == nil {
		return fmt.Errorf(
			"deployment preflight record %s is missing the %q result (Spec 17 R17-02)", path, field)
	}
	if !p.Proven {
		return fmt.Errorf(
			"deployment preflight record %s records %q as NOT proven (Spec 17 R17-02: %s)", path, field, strings.TrimSpace(p.Detail))
	}
	if strings.TrimSpace(p.Detail) == "" {
		return fmt.Errorf(
			"deployment preflight record %s records %q as proven with no detail (Spec 17 R17-02: a proven result must name what was observed)", path, field)
	}
	return nil
}

// loadIngressRecord reads and parses the preflight record, failing closed on
// any access, permission or shape problem. A record other identities can
// rewrite is not evidence.
func loadIngressRecord(path string) (*ingressRecord, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("deployment preflight record %s is unreadable (Spec 17 R17-02): %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("deployment preflight record %s is not a regular file (Spec 17 R17-02)", path)
	}
	if info.Mode().Perm()&0o022 != 0 {
		return nil, fmt.Errorf(
			"deployment preflight record %s is group/world writable (mode %04o); a record other identities can rewrite is not evidence (Spec 17 R17-02)",
			path, info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("deployment preflight record %s is unreadable (Spec 17 R17-02): %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("deployment preflight record %s is empty (Spec 17 R17-02)", path)
	}

	var record ingressRecord
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&record); err != nil {
		return nil, fmt.Errorf(
			"deployment preflight record %s is not a valid %s record (Spec 17 R17-02): %w",
			path, ingressContractVersion, err)
	}
	return &record, nil
}

// isFullRevision reports whether s looks like a full 40-hex commit revision.
func isFullRevision(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// IngressPreflightPath exposes the configured record path (absolute where
// possible) for diagnostics that must not print the record's contents.
func (c *Config) IngressPreflightPath() string {
	if c == nil {
		return ""
	}
	p := strings.TrimSpace(c.DeploymentPreflightFile)
	if p == "" {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}
