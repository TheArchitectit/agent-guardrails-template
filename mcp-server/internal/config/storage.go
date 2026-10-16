package config

import "fmt"

// Storage transit policy (Spec 17 R17-04).
//
// R17-04: unencrypted DB/Redis transit is permitted only inside one verified
// isolated local boundary; whenever either connection leaves that boundary
// (private overlay, cross-host, shared container network, or a public
// deployment with remote storage) the connection MUST use authenticated
// encryption and startup MUST fail if it is missing. Concretely, outside the
// local profile:
//
//   - PostgreSQL must use sslmode=verify-full (certificate + hostname
//     verification against a trusted CA), never disable/prefer, and never an
//     unverified require;
//   - Redis must use TLS (REDIS_USE_TLS=true) with the default certificate and
//     hostname verification (no skip-verify).
//
// The check is a hard error, not a silent upgrade: a misconfiguration must stop
// the process rather than quietly connect in the clear. `local` is deliberately
// unconstrained here — the isolated local bridge/loopback exception named in
// R17-04 — so the existing Compose posture keeps working.
//
// Before this existed, DBSSLMode and RedisUseTLS were validated only as
// well-formed values (config.go), never against the effective deployment
// profile, so a tailnet/public deployment could start with plaintext DB/Redis.

// ValidateStorageTransit fails startup when the storage transit posture does
// not meet R17-04 for the effective deployment profile.
func (c *Config) ValidateStorageTransit() error {
	profile, err := NormalizeDeploymentProfile(c.DeploymentProfile)
	if err != nil {
		return err
	}
	if profile == DeploymentProfileLocal {
		return nil
	}
	if c.DBSSLMode != "verify-full" {
		return fmt.Errorf(
			"profile %q requires authenticated encrypted database transit (Spec 17 R17-04): DB_SSLMODE must be verify-full (certificate and hostname verification against a trusted CA), got %q; plaintext or unverified storage transit is permitted only on a verified isolated local bridge",
			profile, c.DBSSLMode)
	}
	if !c.RedisUseTLS {
		return fmt.Errorf(
			"profile %q requires authenticated encrypted Redis transit (Spec 17 R17-04): REDIS_USE_TLS must be true (verified server certificate and hostname), got false; plaintext storage transit is permitted only on a verified isolated local bridge",
			profile)
	}
	return nil
}
