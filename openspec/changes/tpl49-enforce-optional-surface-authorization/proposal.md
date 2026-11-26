# Authorize every enabled optional MCP and REST path

## Why

The optional Vision dispatch runs ahead of the common MCP authorization and decision audit. Vision REST has its own registration group. An adopter needs a verified decision boundary for every enabled path.

## What changes

Route optional operations through the registered caller, scope, role and resource decision before handlers acquire data or invoke providers. Apply equivalent authorization and redacted audit to REST operations. Retain disabled-by-default exposure.

## Capabilities

Add the `optional-mcp-surface-authorization` capability.

## Impact

The supported change applies to the repository-owned deployment or verification paths identified in the design. Authoritative adopter acceptance remains with the relevant service or release owner.
