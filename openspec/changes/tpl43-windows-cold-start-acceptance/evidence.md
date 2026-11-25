# Evidence and verification limits

## Source inspection

- `openspec/specs/local-docker-compose-pilot/spec.md` specifies synthetic credential creation, launcher-owned reuse, named-volume preservation, and starting Docker Desktop when stopped.
- `openspec/changes/archive/2026-11-25-repair-laptop-docker-compose/evidence.md` is the existing TPL-39 evidence reference. It is not substituted for the separate-host cold process-start acceptance now specified here.

## Not run / not established

No Windows cold-start or restart was run for this package. No Docker process, context, container, volume, env file, credential, or host setting was inspected or modified. The TPL-43 tasks remain unchecked; the existing live pilot is explicitly outside the test scope.
