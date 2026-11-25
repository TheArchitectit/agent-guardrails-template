# Design

## Context

The 2026-11-25 trace distinguishes library and consumer integration. Pi is a separate consumer with its own handler lifecycle, classifier hooks, permission state, and output validator. Existing helper tests can prove local logic but cannot alone prove event registration, ordering, or that a deny prevents the corresponding write/process effect.

## Goals / Non-Goals

**Goals:** a reproducible consumer-level test spanning the real event handlers, Four Laws scope checks, classifier outcomes, permission states, and the output validator's actual effect.

**Non-Goals:** claim all agent hosts are covered; change runtime policy; claim the output validator blocks tool results where code only warns/logs; collect real user data or credentials.

## Decisions

- Use a synthetic temporary workspace and mocked host event API with the production handler registration path; assert event order and actual effect counters.
- Exercise safe and unsafe scope, classifier unavailable/ambiguous, and `auto`/`ask`/`blocked` permission outcomes. A failed or unavailable check may not result in an effect or pass claim.
- Feed synthetic secret/PII output through the real validator. Record whether returned content was altered; if the hook is side-effect-only, classify the result as warning/telemetry and explicitly keep redaction enforcement unverified.
- Bind records to extension revision, configuration digest, classifier identity/version (or `not_configured`), and fixture ID. Omit payload values and secret-like output.

## Risks / Trade-offs

- [A unit helper test is overclaimed as runtime evidence] -> require the real event registration and handler dispatch path.
- [A deny decision is recorded after an effect] -> test the effect counter and order for every denied case.
- [Warning-only output validation is called redaction] -> compare returned bytes and permit only the accurately labeled outcome.

## Migration Plan

Build a local synthetic host harness, run focused and adversarial cases, then have a reviewer inspect event wiring and redacted evidence. Do not change the original user's Pi installation or settings.
