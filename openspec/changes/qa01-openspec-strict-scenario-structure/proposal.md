# Proposal: Make strict OpenSpec scenario validation complete

## Why

At the reviewed baseline, strict validation passes the eight indexed specs and fails seven unindexed specs because their scenarios lack the required WHEN/GIVEN trigger and THEN outcome structure. The repository's index does not demonstrate repository-wide strict validation, so both inventory coverage and scenario-format remediation need explicit acceptance evidence.

## What Changes

- Repair scenario structure in the seven currently failing unindexed specs while preserving each requirement's meaning.
- Add a repeatable strict validation command covering all specs and report its result separately from implementation closure.
- Keep the indexed-versus-unindexed catalog reconciliation in `INV-01`.

## Capabilities

### New Capabilities

- `openspec-strict-validation`: Strict validation reports specification structure across the complete repository inventory.

### Modified Capabilities

None. This change adds no behavior contract to an existing capability.

## Impact

The seven unindexed OpenSpec files and the documented validation procedure/CI configuration if selected. The baseline is 12/19 strict validation items passing overall (all four active changes plus eight specs); the seven strict failures are scenario-format warnings elevated by strict mode.
