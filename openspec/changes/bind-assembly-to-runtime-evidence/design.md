# Design

## Context

Spec 08 requires an evaluation record to bind manifest and policy digests, evaluator identity, and check outcomes. Assembly selection identifies candidate components, while runtime evidence must show what actually ran. The evidence join must be explicit and exact.

## Decisions

- Carry the immutable assembly identifier, selected MCP image/build digest, policy digest, runtime profile, and runtime check identifiers into the evidence record.
- Require runtime-observed build and policy identities to equal the selected assembly values before reporting a passing binding.
- Treat absent, unparseable, or mismatching identities as incomplete or failed evidence; never infer a match from version labels alone.
- Exercise one authorized matching fixture and one controlled mismatch fixture to verify both the positive binding and rejection behavior.
- Keep external consumer transport, deployed topology, and consumer acceptance outside this capability.

## Rollout

Add the evidence join to the existing local runtime-check flow and retain bounded fixture results. Do not change any pin until the exact selected build and policy binding is demonstrated.
