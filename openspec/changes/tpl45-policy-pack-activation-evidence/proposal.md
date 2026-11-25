# Proposal: Verify active policy-pack trust and startup activation

## Why

The repository has resolver, lock, digest, activation and signature code with focused tests. Source presence and isolated unit tests do not prove that production server startup uses the expected trusted key set, activates the candidate pack, and attributes an enforcement decision to its effective digest. The server constructs a fresh manager at startup; no hot-reload caller was found. TPL-45 therefore separates manager-level atomic candidate behavior from production startup evidence without implying reload support.

## What Changes

- Require a startup acceptance record showing configured packs, lock/content digest, signer identity and verification result, active effective digest, and an enforcement rule ID.
- Exercise tampered bytes, unknown signer, invalid signature and production startup diagnostics.
- Verify manager-level failed-candidate atomicity separately; do not describe it as production hot reload.
- Bind one observed enforcement decision to the active policy digest and rule ID, with secret-free evidence.

## Scope mapping

- Primary normative source: `docs/specs/07-versioned-policy-pack-governance.md`, sections 4.2–4.4 and acceptance criteria 1–8.
- Related standard requirement: `openspec/specs/policy-and-conformance/spec.md` (`class-aware policy`, `ancestry reconstruction`) supplies downstream trust/action claims; TPL-45 covers pack activation, trust and digest attribution.
- Related packages: `bind-assembly-to-runtime-evidence` (TPL-36) proves selected assembly inputs match a runtime test; `bind-decision-evidence-context` (RUN-04) binds decision envelopes. Neither is a policy-pack production-startup acceptance run.

## Capabilities

### New Capabilities
- `policy-pack-runtime-activation`: prove configured pack trust, startup activation and use at runtime.

## Impact

Policy-pack deployment and a redacted runtime acceptance record. No private signing key or operator secret is requested or included.
