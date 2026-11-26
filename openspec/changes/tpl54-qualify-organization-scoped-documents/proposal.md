# Qualify shared document access across organization scopes

## Why

The generic document store supports ID, slug, category and full-text paths. Those paths do not establish the current source permission and trusted organization isolation required for a shared document service.

## What changes

Define the shared-deployment scope and enforce trusted isolation and current source permissions across supported reads, list/search, writes, caches and audit. Require a documented interface for adopters using an external retrieval service.

## Capabilities

Add the `organization-scoped-document-access` capability.

## Impact

The supported change applies to the repository-owned deployment or verification paths identified in the design. Authoritative adopter acceptance remains with the relevant service or release owner.
