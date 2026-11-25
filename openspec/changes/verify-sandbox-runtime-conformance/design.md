# Design

## Context

The canonical sandbox requirement says levels are tested against escape fixtures and actual isolation is recorded. Current helpers check static plan properties. This change sets the minimum evidence needed before calling a runtime containment claim exercised.

## Decisions

- Identify runtime, version, host profile, configured level, and workload result in each conformance record.
- Bind the declared and observed isolation level, enforcing runtime identity/version, measured host capability, and run-specific containment-through-completion state into the signed envelope. Verify that signature offline before accepting that run-specific claim; do not infer broader certification.
- Represent an interrupted workload or containment fault as failed/incomplete evidence; never record successful containment through completion for a workload that did not complete under containment.
- Exercise both a harmless permitted operation that must succeed and a harmless denied operation that must be refused for each claimed capability through the supported runtime interface.
- Label these outcomes as safe capability-check evidence; they do not establish resistance to escape attempts or satisfy the separate escape-fixture requirement.
- Label static plan inspection as configuration evidence and keep it distinct from runtime capability results.
- Treat unsupported profiles or unavailable capabilities as not exercised; do not infer containment or upgrade a safe capability check into escape-fixture certification.
- Keep sandbox runtime-envelope fields separate from RUN-04 injection decision context and TPL-36 assembly/build/policy binding. Neither other item substitutes for signing and verifying the sandbox execution claim.

## Rollout

Collect runtime evidence for each supported profile before marking its benign capability checks exercised. Keep escape-fixture conformance unverified until its own separate suite is executed.
