# Qualify supported MCP session topologies

## Why

Session state is process-local and the existing Kubernetes example requests two replicas. Adopters require a route and restart contract for each supported profile.

## What changes

Bind the supported deployment profile to a declared session mode and validate replica, routing and state settings before apply. Establish restart, retry, and identity isolation behavior on a disposable topology.

## Capabilities

Add the `mcp-session-topology` capability.

## Impact

The supported change applies to the repository-owned deployment or verification paths identified in the design. Authoritative adopter acceptance remains with the relevant service or release owner.
