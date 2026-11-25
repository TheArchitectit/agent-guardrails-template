# Design

## Context

`docs/specs/guardrail-gaps-2026/06-regulatory-compliance-mapping.md` records that the measured path uses `GenerateMeasuredReport`/`CheckMeasured` and the three compliance tools, while the older collection path simulated query results. Current code includes engine probes and a measured reporter; the audit finding is the missing integrated live probe/report artifact bound to an actual configured runtime. Some requirements or external sources may remain unknown.

## Goals / Non-Goals

**Goals:** prove a report's measured rows derive from concrete live probes with known sources; demonstrate unmeasured requirements remain unknown; bind artifact to configuration, runtime and probe time; retain redacted output.

**Non-Goals:** claim legal compliance, treat self-declarations as probes, report full coverage from partial probes, simulate a query as an observation, or duplicate semantic/injection conformance and release-promotion work.

## Decisions

- Use an isolated server with explicitly activated framework configuration and named probe adapters. Record each probe's source identifier, collection window, runtime/build identity, status and evidence reference.
- Require a positive control where a known live control is observed and a negative control where an absent/unconfigured control remains unknown; require an empty probe set to be refused.
- Compare the exported report to the actual probe result set and verify every measured row has a supporting observation. Declarations remain separate metadata and never promote an unknown probe to observed.
- Redact credentials, personal data, raw query results and private endpoints; store the signed or hashed artifact in approved local test storage.

## Risks / Trade-offs

- [Simulated evidence is reported as live] -> trace each report row to the concrete probe call/result and include probe-source IDs.
- [Unknown is converted to covered] -> include absent-control fixture and assert its report status stays unknown/unmeasured.
- [Sensitive data leaks through an artifact] -> use synthetic fixtures, redact payloads, and inspect the final report before retention.

## Migration Plan

Select an authorized isolated runtime and approved evidence sources, execute the probe matrix, verify artifact provenance and unknown handling, and obtain independent review. Keep any unobservable controls explicitly open.
