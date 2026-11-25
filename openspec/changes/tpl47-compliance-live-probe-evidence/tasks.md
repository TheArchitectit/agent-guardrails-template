# Tasks

## 1. Run live compliance probes

- [ ] 1.1 Select an authorized isolated runtime and activated framework/configuration; enumerate concrete probe sources, collection intervals, responsible owner, and redaction requirements.
- [ ] 1.2 Run at least one positive observed-control probe, one absent/unavailable control probe, and the empty-probe refusal through the real MCP/runtime reporting path.
- [ ] 1.3 Export the measured report and compare every covered row to the actual probe observation, runtime identity/config, and collection interval; verify declared/simulated evidence is not counted.
- [ ] 1.4 Review report privacy, integrity/signature or digest metadata, and unknown/unmeasured rows; retain only a redacted artifact and reviewer reference.

## 2. Bound the claim

- [ ] 2.1 Independently verify that no report row is sourced only from a declaration, simulated query, or helper-only test.
- [ ] 2.2 Map remaining unobservable controls and external frameworks to open gaps; avoid a compliance certification claim.

## Workflow follow-up

Keep all tasks open until authorized live probes run and are independently reviewed. If a control has no observable probe, report it as unmeasured.
