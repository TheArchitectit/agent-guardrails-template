# Governance Playbook

## Purpose
- Define who does what, how decisions get made, and how changes are audited.

## Roles
- Executive Steering Board: strategic oversight, budget approvals, risk governance.
- Architecture Review Board (ARB): ADRs, interface contracts, data contracts, architectural decisions.
- Security & Compliance Panel: SBOM, risk posture, privacy, audit readiness.
- Docs Governance Council: document quality, segmentation, owner assignments, gatekeeping.
- Release Gate Review: cross-functional sign-off for each major release.

## Ceremonies and cadence
- Executive Steering Board: monthly review meetings.
- ARB: quarterly or ad-hoc as needed; ADR review cycle.
- Security & Compliance Panel: monthly checks; incident drills.
- Docs Governance Council: monthly doc health review; change-control gating.
- Release Gate Review: per release, led by Release Manager.

## Change-control and gates
- All governance changes require ARB sign-off.
- All doc changes require owner sign-off and doc-health gates.
- Release gating includes architecture, security, accessibility, observability, and migration readiness.

## Documentation lifecycle
- Draft → Review → Approve → Publish → Archive
- Archive keeps historical reference with deprecation notes.

## Evidence and traceability
- Each governance decision captured as ADRs (templates included in docs/templates/adr-template.md).
- Gating evidence stored with the release package.
