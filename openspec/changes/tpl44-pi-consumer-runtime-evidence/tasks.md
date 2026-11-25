# Tasks

## 1. Build and exercise the Pi consumer fixture

- [ ] 1.1 Construct a synthetic host lifecycle fixture that calls production handler registration and dispatch, with effect counters for file edits and shell/process calls.
- [ ] 1.2 Run safe and unsafe Four Laws scope, classifier pass/block/unavailable, and permission `auto`/`ask`/`blocked` cases; verify decision order and that denials precede effects.
- [ ] 1.3 Run a synthetic secret/PII fixture through the real output-validation lifecycle; compare returned content and classify enforcement versus warning-only behavior.
- [ ] 1.4 Record extension revision, configuration and classifier identity/version, fixture IDs, redacted outcomes, and any unverified behaviors without storing fixture payload values.

## 2. Review consumer claims

- [ ] 2.1 Independently inspect the registered Pi handlers and confirm each result comes from the production event path, not a direct helper call.
- [ ] 2.2 Map the verified evidence separately to Four Laws, classifier, permissions, and output redaction; leave unsupported claims open.

## Workflow follow-up

Do not alter a user Pi installation. Keep all tasks open until the isolated integration run and independent review are recorded.
