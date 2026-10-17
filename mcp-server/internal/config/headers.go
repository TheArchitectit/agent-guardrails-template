package config

// HSTSHeaderValue is the emitted HTTP Strict-Transport-Security policy.
// Spec 12 Sec. 3.6 requires the header only on a verified public HTTPS origin.
const HSTSHeaderValue = "max-age=31536000; includeSubDomains"

// EmitsHSTS reports whether this deployment may emit HSTS (Spec 12 Sec. 3.6,
// Spec 17 R17-03: "HSTS may be emitted only for a verified public HTTPS
// origin, never as a substitute for TLS").
//
// Both conditions must hold:
//
//   - TLS is enabled for this listener (TLS_ENABLED=true). Emitting HSTS over
//     plaintext would pin a scheme the listener cannot serve, and the header is
//     ignored by clients over HTTP anyway — the danger is the reverse case
//     below, not this one.
//   - The effective deployment profile is `public`. A `local` or `tailnet`
//     deployment is not a public HTTPS origin, so pinning HSTS there would
//     assert a posture the operator never declared.
//
// A profile that fails to normalize is treated as non-public (no header):
// the failure mode of an unparsed profile must not be an emitted security
// claim. Config.Validate rejects such a profile at load, so this is a
// belt-and-braces read for a directly-constructed Config.
func (c *Config) EmitsHSTS() bool {
	if !c.TLSEnabled {
		return false
	}
	profile, err := NormalizeDeploymentProfile(c.DeploymentProfile)
	if err != nil {
		return false
	}
	return profile == DeploymentProfilePublic
}
