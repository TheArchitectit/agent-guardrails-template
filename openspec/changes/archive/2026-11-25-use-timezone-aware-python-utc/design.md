# Design

## Context

`team_manager.py` emits UTC timestamps for logs, metrics, audit records, and
reports. `log_aggregator.py` constructs relative time boundaries and parses
timestamp filters. Both use naive `datetime.utcnow()` values; existing JSON
fields commonly serialize the ISO value with a terminal `Z`.

## Goals / Non-Goals

**Goals:**

- Use aware UTC values consistently for time-window arithmetic and comparisons.
- Keep existing timestamp keys and the serialized ISO-plus-`Z` shape.

**Non-Goals:**

- Change persisted JSON/CSV field names or migrate existing records.
- Redesign timestamp precision or accept non-UTC output.

## Decisions

- Create current instants with `datetime.now(timezone.utc)` and keep them aware
  through arithmetic and filtering. A naive replacement of `utcnow()` is
  rejected because it would retain the ambiguity and mixed-awareness risk.
- At serialization boundaries, render aware UTC values to the existing ISO
  string shape with `Z` instead of exposing a new `+00:00` suffix. This keeps
  consumers of current logs and reports compatible.
- Parse existing `Z` timestamps into aware UTC values and normalize any
  supported explicit offset before comparison. Relative CLI windows use the
  same aware representation.
- Keep timestamp construction and serialization tests next to each script's
  behavior. A shared helper may be used where it does not create a new
  import-path dependency for standalone scripts.

## Risks / Trade-offs

- Mixing historic timestamps with and without an explicit timezone can cause
  comparison errors. Continue accepting the current stored forms and normalize
  parsed values to UTC before filtering.
- Adding `+00:00` to output would be valid ISO but can break string consumers.
  Assert the established terminal `Z` format in tests.

## Migration Plan

No storage migration is required. Update production timestamp creation and
filter boundaries, add/adjust compatibility tests, then run `pytest scripts/`
on Python 3.12 and inspect output-format and deprecation results. Roll back
only the code and test changes if compatibility checks fail; existing records
remain readable.
