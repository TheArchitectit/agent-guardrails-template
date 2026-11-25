# Tasks

## 1. Prepare isolated cold-start acceptance

- [ ] 1.1 Select a separate authorized Windows host or VM and unique user-local state root. Verify the launcher's fixed project name has no existing pilot containers, env file, or volumes; do not override `COMPOSE_PROJECT_NAME` or inspect the original laptop stack.
- [ ] 1.2 On that host, verify Docker Desktop is stopped and the configured `desktop-linux` context targets the local Windows named pipe while the Linux engine is unavailable. Define secret-safe continuity comparison, stop conditions, and cleanup before launcher invocation.

## 2. Run and review acceptance

- [ ] 2.1 Invoke the exact pinned launcher without parameters from the cold state; record Desktop process start, context/engine readiness within five minutes, source revision, synthetic credential-file properties, Compose policy, and application readiness without recording credential values.
- [ ] 2.2 Run the documented volume-preserving teardown without `--volumes`, then invoke the same parameterless launcher; verify credential continuity and named-volume persistence without printing/storing credentials.
- [ ] 2.3 Independently review the redacted evidence, confirm no original pilot resource changed, and link the evidence to both TPL-43 requirement rows.

## Workflow follow-up

Do not close this change based on TPL-39 startup/restart evidence alone. Keep it open until an authorized separate-host cold process-start and persisted-restart run is reviewed.
