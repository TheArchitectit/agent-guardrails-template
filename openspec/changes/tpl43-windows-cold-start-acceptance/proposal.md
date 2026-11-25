# Proposal: Verify cold-start and persisted restart acceptance

## Why

The published `local-docker-compose-pilot` spec requires synthetic credential creation, create-new behavior, launcher-owned credential reuse, and named-volume preservation. Existing TPL-39 evidence reports an earlier startup/restart path, but the delivered-code audit identifies automatic cold start of Docker Desktop as an acceptance gap. Required evidence must start from a stopped Desktop process and unavailable local Linux engine on a separate authorized Windows host or VM, then prove first-run and state reuse. The original pilot must remain untouched.

## What Changes

- Make separate-host cold Desktop startup, first-run credential creation, and same-state restart acceptance explicit for the existing pilot contract.
- Use the launcher’s fixed project name because it rejects ambient `COMPOSE_PROJECT_NAME`; check its target resources are absent.
- Require evidence of local Linux engine readiness, credential reuse without secret values, and volume persistence after documented teardown without `--volumes`.

## Scope mapping

- Published scope: `openspec/specs/local-docker-compose-pilot/spec.md`, `Windows launcher targets only the local Docker Desktop Linux engine`, `Windows launcher is repeatable without replacing pilot data`, `Pilot credentials stay synthetic and outside the build context`, and `Pilot teardown preserves named data volumes`.
- Related completed change: TPL-39 archive proves its stated startup/restart run; this package asks for separate-host cold process-start and persistence acceptance, not a repeat of its implementation.
- Related operation scope: `ops04-rehearse-operational-recovery` remains topology-specific migration/restore/rollback and is not a substitute for launcher credential lifecycle evidence.

## Capabilities

### Modified Capabilities
- `local-docker-compose-pilot`: require isolated cold-start and persisted-restart acceptance on a separate authorized host.

## Impact

Pilot QA evidence and the local Docker Compose pilot acceptance contract. Do not inspect or modify the existing pilot stack or its secrets.
