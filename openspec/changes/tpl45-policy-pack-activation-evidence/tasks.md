# Tasks

## 1. Verify manager-level candidate atomicity

- [ ] 1.1 Build disposable pack/lock fixtures and ephemeral test signing keys; record source revision and test configuration.
- [ ] 1.2 At manager level, activate a valid candidate then submit a tampered/invalid candidate; verify the prior active set and digest remain unchanged. Label this as manager-level behavior only.

## 2. Verify production startup and diagnostics

- [ ] 2.1 Start an isolated server through the production configuration path with a valid locked/signed pack and out-of-band ephemeral trust key; record signer ID, lock/content digest, effective digest and rule attribution.
- [ ] 2.2 Run production-startup controls for unusable trust configuration, changed pack bytes, unknown key and invalid signature; capture actual diagnostics/status and verify no bad candidate is reported active.
- [ ] 2.3 Independently compare runtime observations with the exact lock, trust set and configuration. Do not claim hot reload or semantic approval.
- [ ] 2.4 Produce redacted machine-readable evidence excluding private keys, raw credentials and policy secrets.

## Workflow follow-up

Keep manager atomicity and server startup evidence distinct. Do not use production signing keys or label signature validity as policy approval. Keep the package open until integrated evidence and independent review are complete.
