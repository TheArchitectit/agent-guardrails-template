# Design

## Context

The checked-in pilot contract specifies synthetic secrets outside the build context, create-new semantics, launcher-owned credential reuse, and named-volume preservation. The audit requires more than a warm restart: on a separate authorized Windows host or VM, Docker Desktop must initially be stopped and its local Linux engine unavailable; the exact launcher must start Docker Desktop, confirm the local Linux context/readiness, perform a clean bootstrap, then prove credential and volume reuse after restart. TPL-39's prior startup/restart is not evidence for that cold process-start acceptance.

## Goals / Non-Goals

**Goals:** obtain repeatable, redacted evidence for a cold Windows Docker Desktop process start, first pilot run, and restart using the same launcher-owned state.

**Non-Goals:** change, stop, restart, inspect, or delete the user's original Docker pilot; inspect host secrets; authorize external connectivity; change the published source pin or implementation behavior.

## Decisions

- Run only on a separate authorized Windows host or VM. Require the standard Docker Desktop application to be stopped and the `desktop-linux` engine unavailable before the run; verify the configured context points to the local Docker Desktop Linux named pipe before starting.
- Use a temporary user-local state root. The launcher rejects ambient `COMPOSE_PROJECT_NAME`, so use its fixed project name and prove no matching container, env file, or volume exists before startup. Do not inspect the original pilot to establish this.
- Start with no launcher env file or named volumes in that isolated project. Record credential-file properties and a non-secret keyed digest, never raw credentials.
- Use the documented volume-preserving Compose teardown, then invoke the exact parameterless launcher again. Compare a protected local HMAC/digest and prove named volumes persist.
- Record exact launcher/source revision, Compose render policy, selected Docker context/engine identity, readiness result, and cleanup outcome. Any inability to prove isolation is a stop condition.

## Risks / Trade-offs

- [The test collides with the live pilot] -> use a separate authorized host/VM and the launcher's fixed project name; check ownership before Compose and stop on ambiguity.
- [Evidence leaks synthetic credentials] -> never capture raw env output or container environment; report only redacted digests and pass/fail.
- [A warm engine run is mistaken for cold startup] -> verify Desktop process is stopped and engine unavailable before invoking the launcher.

## Migration Plan

After the separate host/VM is authorized and the isolated test plan is independently reviewed, run against disposable data. Keep tasks open until the redacted evidence proves cold app startup, first-run readiness, credential reuse, and volume persistence.
