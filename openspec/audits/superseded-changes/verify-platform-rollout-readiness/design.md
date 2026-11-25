# Design

## Context

Release checks exercise repository controls on named runners. A rollout also
needs evidence for the actual client, identity provider, ingress, storage,
and replica topology selected by the operators.

## Decisions

- Use a rollout inventory that maps each supported client and enforcement
  boundary to a deployment profile, policy version, service owner, and
  verification record. Keep operational identities, network details, and
  approval records in the operator's approved evidence store.
- Render and validate the deployment manifest for the selected profile before
  applying it. Verify probe and client reachability through the intended
  ingress and verify that direct access is constrained by that profile.
  Update deployment documentation from this verified path.
- Use the supported authentication and authorization paths for a real
  consumer smoke test. Bind its result to the image, policy, consumer version,
  and deployment profile. If the deployment has multiple replicas, verify
  credential revocation on each within the documented bound and during the
  selected failover procedure. Evidence verification records the outcome of a
  decision. Authorization remains with the deployed policy path.
- Rehearse the documented migration, readiness checks, recovery from backup,
  and rollback in an isolated environment matching the selected topology.
  Have the responsible operators record the observed results and whether
  those results meet the recovery objectives they set for the deployment.

## Rollout

Validate one supported profile and consumer first. Record failed and unrun
checks alongside successful runs. Extend the evidence to each additional
profile or client before relying on it for broader adoption.
