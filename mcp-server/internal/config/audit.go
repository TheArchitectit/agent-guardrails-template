package config

import "fmt"

// Audit-logging posture (Spec 16 R16-08 audit durability).
//
// Three settings were declared, documented and (in one case) range-validated
// while nothing in the process read them:
//
//   - AUDIT_BUFFER_SIZE was validated to 100..10000 in Config.Validate, and
//     cmd/server/main.go then built the logger with a hardcoded literal `1000`.
//     An operator who raised the buffer to survive an audit burst got the same
//     1000-slot channel, and one who lowered it to bound memory got 1000 too.
//   - ENABLE_AUDIT_LOGGING had no consumer at all: audit logging could not be
//     turned off, and the documented "recommended for production" toggle was
//     inert in both directions.
//   - AUDIT_FLUSH_INTERVAL has no implementation anywhere in the tree (the
//     logger writes each event as it is dequeued; there is no batch and no
//     flush). It has been removed rather than re-documented, so the shipped
//     surface stops claiming a behaviour that does not exist.
//
// The durability half matters most: `Logger.LogDecision(..., required=true)`
// fails closed when there is no durable store, but cmd/server/main.go never
// called SetStore, so *every* decision the server recorded went to the process
// log only. A nonlocal deployment therefore ran with an audit trail that did
// not survive the process, while the control-plane probe reported exactly that
// ("audit logger emits to the process log only; no durable store"). This file
// makes the posture explicit and gates it at startup.

// AuditDurabilityRequired reports whether this deployment must run with audit
// logging enabled and a durable store behind it.
//
// Any nonlocal profile, or an explicit production mode, must have audit on: a
// public/tailnet deployment that silently accepts ENABLE_AUDIT_LOGGING=false
// is a deployment whose authorization decisions leave no durable record. The
// local profile keeps the documented development posture.
func (c *Config) AuditDurabilityRequired() bool {
	profile, err := NormalizeDeploymentProfile(c.DeploymentProfile)
	if err != nil {
		// An unknown profile is refused by ValidateDeploymentBinding; treat it
		// as nonlocal here so the stricter of the two rules always applies.
		return true
	}
	if c.ProductionMode {
		return true
	}
	return profile != DeploymentProfileLocal
}

// AuditBufferCapacity returns the configured audit channel capacity.
//
// The value is the operator's declared AUDIT_BUFFER_SIZE, already bounded by
// Config.Validate; it exists so the startup path cannot diverge from the
// validated setting (previously the literal 1000 did).
func (c *Config) AuditBufferCapacity() int {
	if c.AuditBufferSize <= 0 {
		return DefaultAuditBufferSize
	}
	return c.AuditBufferSize
}

// DefaultAuditBufferSize is the shipped capacity when nothing is configured.
// It is published so the startup wiring and the tests share one source of
// truth instead of repeating a literal.
const DefaultAuditBufferSize = 1000

// ValidateAuditPosture refuses a deployment that cannot honour its audit
// contract, before any listener is opened.
//
// The refusal is deliberate rather than a silent re-enable: a deployment whose
// operator turned audit off, and whose profile says it must be on, is a
// misconfiguration that should stop the process, not run unaudited with a log
// line explaining it.
func (c *Config) ValidateAuditPosture() error {
	if !c.EnableAuditLogging && c.AuditDurabilityRequired() {
		return fmt.Errorf(
			"ENABLE_AUDIT_LOGGING=false is refused for deployment profile %q (production=%t): "+
				"a nonlocal deployment must record authorization decisions durably (Spec 16 R16-08)",
			c.DeploymentProfile, c.ProductionMode)
	}
	return nil
}
