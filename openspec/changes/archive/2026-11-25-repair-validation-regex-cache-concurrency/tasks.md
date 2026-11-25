## 1. Specify and record

- [x] 1.1 Record the observed race, affected files, and preservation constraints in the failure registry and OpenSpec evidence.
- [x] 1.2 Add the validation regex cache requirement and concrete concurrency scenarios.

## 2. Reproduce and repair

- [x] 2.1 Add a deterministic native Go regression for concurrent same-pattern compilation/matching and cache clearing; demonstrate it detects the pre-fix race.
- [x] 2.2 Store immutable compiled regex pointers, safely publish concurrent misses, and clear the existing `sync.Map` in place.
- [x] 2.3 Add a repository regression entry that invokes the native race test without duplicating its logic.

## 3. Verify and report

- [x] 3.1 Run the focused race regression repeatedly, then focused validation/corpus/control tests and the package suite with fresh results.
- [x] 3.2 Run regression guard checks, strict OpenSpec validation, and review the exact diff and evidence.
- [x] 3.3 Update the template gap register with verified status and remaining limitations.
