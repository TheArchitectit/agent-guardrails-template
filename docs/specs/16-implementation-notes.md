# OpenSpec 16 — Implementation notes (audit posture)

Companion to [`16-authentication-authorization-and-evidence-remediation.md`](16-authentication-authorization-and-evidence-remediation.md).
Split out on 2026-10-19 to keep the spec within the 500-line Markdown limit
(Spec 19 R19-11). Dated note; the spec body holds the requirements, not this file.

### R16-08 audit posture — the declared settings were inert and the store was never wired (2026-10-19)

R16-08 requires the decision/effect audit to be **recorded**, and this section
already recorded the durability boundary honestly: `Logger.LogDecision(required=true)`
fails closed when no durable store is configured. What no pass had checked is
whether the server ever *gave* the logger a store. It did not:
`cmd/server/main.go` called `audit.NewLogger(1000)` and never `SetStore`, so
`HasDurableStore()` was false in every deployment and every decision the server
recorded went to the process log only — the compliance probe's own text
("audit logger emits to the process log only; no durable store") was describing
the shipped process.

Three settings around it were declared, documented and inert:

- **`AUDIT_BUFFER_SIZE`** was range-validated (100..10000) inside `Validate`
  and then discarded: the startup path built the channel with a hardcoded
  `1000`. Raising it to survive an audit burst changed nothing.
- **`ENABLE_AUDIT_LOGGING`** had no consumer in either direction — audit
  logging could not be turned off, and the setting documented as
  "recommended for production" was decorative.
- **`AUDIT_FLUSH_INTERVAL`** had no implementation anywhere in the tree (the
  logger writes each event as it is dequeued; there is no batch and no flush
  path). It has been **removed** rather than re-documented, so the shipped
  surface stops claiming a behaviour that does not exist.

**Fixed at each cause.** The startup path gives the logger the durable store
(`database.NewAuditStore(db)`) and drains it (`auditLogger.Stop()`) **before**
the database handle closes, so events emitted during web/MCP shutdown are still
written. The channel capacity is `cfg.AuditBufferCapacity()`, derived from the
validated setting. `Config.ValidateAuditPosture` (called from `Config.Validate`,
so it gates startup) **refuses** a nonlocal or `PRODUCTION_MODE` deployment that
disables audit logging, naming the variable and Spec 16 R16-08 — a deliberate
refusal, not a silent re-enable, because a deployment that turned audit off and
whose profile requires it is a misconfiguration that should not run unaudited.
The local profile keeps the documented development posture.

**Evidence.** `internal/config/audit_posture_test.go` — local profile exempt;
`tailnet`/`public` refused naming the variable; `PRODUCTION_MODE` refused even on
`local`; the enabled nonlocal case allowed (negative control); the capacity
follows the configured value and falls back to the published default; and
`TestValidateCouplesAuditPostureToStartup` proves the gate runs inside
`Validate`. **Red-on-drift proven three ways**, each PASS→FAIL then restored:
removing the `ValidateAuditPosture` call from `Validate`; making
`AuditBufferCapacity` ignore the configured value; disabling the fail-closed
guard. One test fixture of mine was incomplete (`validMinimalConfig` omitted the
audit posture, so a production-shaped config defaulted to `false`) — the
**fixture** was corrected, never the gate.

**Named limit.** This pass wires and gates the durability path; it does not add
a batch/flush implementation (the setting that implied one was removed), and it
does not make the store itself survive a database outage — a required-decision
write failure still fails closed, which is the documented behaviour.

