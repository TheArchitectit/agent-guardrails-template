# Qualify shared document access across organization scopes

## Context

The generic document store supports ID, slug, category and full-text paths. Those paths do not establish the current source permission and trusted organization isolation required for a shared document service.

## Decisions

Use trusted request context for organization scope, bound to the authenticated caller. Inventory direct and derived document access, include aliases and metadata, and keep permissions consistent on cache hits. Each supported profile declares which document surfaces it exposes and their authorized clients.

## Risk and verification

Identifiers or cached results can cross organizations when source entitlements change. Exercise two organizations sharing identical logical document identifiers and verify removal of access after revocation.

## Rollout

Review the scenario matrix, exercise it with an isolated configuration, and record exact source and artifact identities before enabling the supported profile or recording acceptance.
