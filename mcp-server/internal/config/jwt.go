package config

import (
	"fmt"
	"time"
)

// JWT rotation/expiry posture (Spec 17 R17-10).
//
// R17-10 requires JWT signing material to rotate "with a staged replacement and
// explicit cutover" and requires JWT rollover to "identify the verifying key,
// bound token expiry and reject retired keys". The tree declared
// JWT_ROTATION_HOURS (default 168h / 7 days) and JWT_EXPIRY (default 15m) as
// environment settings but never bounded either: ValidateTimeout was applied to
// the shutdown/request/DB/circuit-breaker timers and NOT to the JWT settings, so
// an operator could set JWT_ROTATION_HOURS to any value — or, with a
// hand-built Config, to no value at all — and startup accepted it.
//
// This closed the open MEDIUM row 9 of docs/security/security-audit-config.md
// ("JWT Rotation Period Too Long": a long rotation period increases the
// exposure window of a compromised token, and there was "no mechanism for
// emergency rotation"). The audit proposed the exact bound enforced here.
//
// Honest limit, recorded rather than implied: this validates the *declared*
// posture. No JWT is minted or verified anywhere in this tree today — there is
// no JWT library in go.mod and JWTIssuer/JWTExpiry/JWTRotationHours have no
// runtime consumer (agent/session tokens are opaque, database-backed). The
// gate therefore bounds a posture an operator sets and that a future token
// implementation must honour; it does not claim a token path exists. The secret
// inventory records the same fact so a reader is not told internal/auth
// consumes a value nothing carries.

// MaxJWTRotationWindow is the maximum permitted JWT signing-key rotation window
// (Spec 17 R17-10, docs/security/security-audit-config.md row 9: "should not
// exceed 168 hours (7 days)"). A window longer than this widens the exposure of
// a compromised signing key and forces a longer token-validity ceiling.
const MaxJWTRotationWindow = 168 * time.Hour

// DefaultJWTRotationWindow mirrors the envDefault for JWT_ROTATION_HOURS. It is
// the value adopted when the field is unset/zero (a directly-constructed
// Config), never a silent widening of a configured value.
const DefaultJWTRotationWindow = 168 * time.Hour

// DefaultJWTTTL mirrors the envDefault for JWT_EXPIRY.
const DefaultJWTTTL = 15 * time.Minute

// EffectiveJWTRotationWindow returns the rotation window that applies to this
// configuration. A non-positive value means "unset" and adopts the documented
// default, so a hand-built Config is bounded rather than treated as
// "rotate never". The env layer already applies the default for an unset
// variable; this keeps a directly-constructed Config on the same path.
func (c *Config) EffectiveJWTRotationWindow() time.Duration {
	if c.JWTRotationHours <= 0 {
		return DefaultJWTRotationWindow
	}
	return c.JWTRotationHours
}

// EffectiveJWTTTL returns the token validity window that applies to this
// configuration, defaulting a non-positive value to the documented default.
func (c *Config) EffectiveJWTTTL() time.Duration {
	if c.JWTExpiry <= 0 {
		return DefaultJWTTTL
	}
	return c.JWTExpiry
}

// ValidateJWTPosture enforces the bounded rotation/expiry posture required by
// Spec 17 R17-10. It is called from Config.Validate so the posture gates
// startup; a misconfiguration stops the process rather than serving tokens
// under an unbounded policy. Both rules fail closed and name the requirement:
//
//   - the rotation window must not exceed MaxJWTRotationWindow; and
//   - a token's validity must not exceed the rotation window, because a token
//     that outlives the rotation of the key that signed it cannot have its
//     signing key retired within the window the operator declared.
func (c *Config) ValidateJWTPosture() error {
	rotation := c.EffectiveJWTRotationWindow()
	if rotation > MaxJWTRotationWindow {
		return fmt.Errorf(
			"JWT_ROTATION_HOURS must be at most %v (Spec 17 R17-10 / security-audit-config.md row 9: a longer rotation window widens the exposure of a compromised signing key): got %v",
			MaxJWTRotationWindow, rotation)
	}
	if ttl := c.EffectiveJWTTTL(); ttl > rotation {
		return fmt.Errorf(
			"JWT_EXPIRY (%v) must not exceed JWT_ROTATION_HOURS (%v) (Spec 17 R17-10: bound token expiry — a token must not outlive the rotation of the key that signed it)",
			ttl, rotation)
	}
	return nil
}
