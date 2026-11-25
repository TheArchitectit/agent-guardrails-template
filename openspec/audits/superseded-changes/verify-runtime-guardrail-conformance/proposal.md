# Proposal: Verify runtime guardrail conformance

## Why

The current closure evidence for several guardrail specs demonstrates helper
logic, test harnesses, and explicit MCP tool behavior, but does not show that
the named runtime boundaries and representative production configurations
were exercised. In particular, the sandbox conformance evaluator builds and
checks a policy plan without executing a workload in the supported runtime;
semantic conformance relies on marker-recognition reference fixtures; and the
repository does not evidence automatic interception at every mediation and
ingestion boundary named by the specs. Injection evidence also does not carry
all of the active classifier, category, policy, and verification information
described by its spec.

These are evidence and scope-alignment gaps. The review has not established
behavior outside this repository or an exploitable weakness. The corresponding
findings are `RUN-01` through `RUN-04` in
`openspec/audits/2026-11-25-qa-and-gap-review.md`.

## Change

Specify the evidence needed for runtime conformance claims: supported-runtime
execution, representative versioned semantic corpora evaluated with configured
classifiers, demonstrated action and ingestion boundary wiring, and evidence
that binds claims to active policy and classifier identity. Until such
evidence exists, closure reporting must state the narrower behavior actually
verified.

## Impact

This proposes verification and evidence requirements. It does not change
production code or reclassify existing specs in this review.
