# Tasks

- [ ] [RUN-01] Add supported-runtime capability checks with a benign permitted operation that succeeds and a benign denied operation that is blocked for each claimed boundary.
- [ ] [RUN-01] Record runtime identity, version, host profile, configured isolation level, operation, and observed result; distinguish static plan checks.
- [ ] [RUN-01] Modify `isolation in evidence` so signed workload envelopes include declared/actual isolation, enforcing runtime identity/version, measured host capability, and whether containment held through workload completion.
- [ ] [RUN-01] Add hermetic verifier tests with a local generated verification key: verify signed field binding offline, reject omitted or altered fields, and ensure interruption/fault evidence cannot claim successful completion.
- [ ] [RUN-01] Reconcile the sandbox closure evidence to distinguish safe capability checks, signed run-specific evidence, and full escape-fixture conformance; do not retain an exercised claim based only on static helper tests.
- [ ] [RUN-01] Mark only the safe capability checks exercised; leave escape-fixture conformance and unevaluated profiles explicitly unverified.
