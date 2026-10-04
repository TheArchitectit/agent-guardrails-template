# Optional OAP Guardrail Checker

**Status:** Proposed and conditional; not implemented. This package records the
outcome of the AIGGP-11 audit material staged in the private
`repo-brainstorming` repository; it is blocked until a second real consumer
justifies portable reuse and OAP native authority/effect gates are independently
accepted.

## Boundary

Guardrails owns agent/runtime safety checks: prompt-injection handling,
provenance, semantic content decisions, sandbox posture, and bounded evidence
about checks that actually ran. OAP owns caller identity, tenant membership,
roles/scopes, resource ownership, final action authorization, credentials,
revocation, queues, effects, and native audit.

This checker may inspect trusted typed context supplied by a host and return
Guardrails findings/evidence. It may not mint identity, elevate scope, select a
tenant, authorize an OAP effect, or treat a receipt/signature as permission.

## Activation gate

Do not implement this as an active module unless all gates below have a named
artifact, expected result, and failure condition:

1. **OAP readiness record:** an independently reviewed report references the
   OAP commit, native route/effect inventory, passing no-effect tests, tenant
   isolation tests, revocation/rollback tests, and pre-effect audit evidence.
   Missing inventory or any failed mandatory test is a blocker.
2. **Second-consumer record:** a named second real product consumer has a
   source/configuration path, owner, supported contract range, and an exercised
   conformance run. A hypothetical consumer does not satisfy this gate.
3. **Pilot record:** a bounded subject and actual Guardrails request path have a
   run artifact showing producer, adapter, verifier, consumer, policy digest,
   outcome, and observe-only enforcement. Library-only or unexercised paths
   fail this gate.
4. **Contract conformance record:** frozen fixtures pass for wrong context,
   tampering, replay, expiry, missing/error/skipped checks, credential scope,
   and host-denied no-effect behavior. Any status laundering or side effect
   fails this gate.
5. **Maintenance decision:** an owner, compatibility window, deprecation path,
   and deletion trigger are recorded. No owner means no activation.

## Related

- [`spec.md`](spec.md) — proposed checker requirements.
- [Spec 07](../07-versioned-policy-pack-governance.md) — proposed policy identity and trust.
- [Spec 08](../08-control-plane-composition-and-evidence.md) — proposed evidence/state model.
- [Spec 11](../11-authorization-scopes-and-roles.md) — Guardrails-local authorization boundary.
- [Spec 12](../12-web-exposure-boundary.md) — web exposure boundary.
- [Spec 13](../13-webhook-ssrf-hardening.md) — outbound webhook safety.
- AIGGP `add-oap-evidence-consumer` — product-owned result-preserving contract; it is maintained in the separate AIGGP repository and is not a shipped integration.
