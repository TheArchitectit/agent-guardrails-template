# Qualify supported MCP session topologies

## Context

Session state is process-local and the existing Kubernetes example requests two replicas. Adopters require a route and restart contract for each supported profile.

## Decisions

With process-local state, replacement requires fresh initialization. A profile promising resumption must verify state availability, repeated authentication, principal binding, expiry, and revocation across every serving instance. Require fresh authorization on a new connection. Coordinate rendered delivery with DEP-01 and drills with OPS-03 and OPS-04.

## Risk and verification

Sticky routing can mask process loss or mismatched identities. Exercise pod replacement, a new connection and disallowed identity, along with any supported multi-replica movement.

## Rollout

Review the scenario matrix, exercise it with an isolated configuration, and record exact source and artifact identities before enabling the supported profile or recording acceptance.
