# Evidence and verification limits

## Source inspection

- `pi-extension/index.ts`, `pi-extension/handlers.ts`, and `pi-extension/standalone/` contain the Pi hook/handler surfaces identified in the audit.
- `pi-extension/permissions/` implements consumer permission decisions.
- `pi-extension/output-validator/` is the output-validation surface; `docs/integrations/pi.md` describes its supported lifecycle.
- Existing test symbols are documented in the audit trace, but no Pi host lifecycle test was run for this package.

## Not run / not established

No consumer runtime harness or live Pi session was run. No real project, user content, API key, or Pi settings were accessed. The claims and tasks remain open until the actual event path and returned-output behavior are tested.
