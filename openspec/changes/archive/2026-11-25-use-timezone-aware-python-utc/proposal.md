# Proposal

## Why

The Python 3.12 QA run emitted 363 deprecation warnings, primarily from naive
`datetime.utcnow()` calls in `scripts/team_manager.py` and
`scripts/log_aggregator.py`. Replacing those calls with timezone-aware UTC
values will remove the deprecated API use while preserving the established
timestamp strings consumed by scripts and serialized reports.

## What Changes

- Use timezone-aware UTC values for timestamps and UTC date arithmetic.
- Preserve the existing serialized UTC wire shape, including the `Z` suffix
  where currently emitted.
- Add compatibility tests for timestamp parsing and serialized output.

## Capabilities

### New Capabilities

- `python-utc-timestamps`: timezone-aware UTC timestamp production with
  compatible serialized values.

### Modified Capabilities

None. No existing OpenSpec capability defines Python script timestamp output.

## Impact

Affected code includes timestamp and date-window construction in
`scripts/team_manager.py` and `scripts/log_aggregator.py`, plus their tests and
serialized artifacts. Audit finding `QA-03` is recorded in
`openspec/audits/2026-11-25-qa-and-gap-review.md`.
