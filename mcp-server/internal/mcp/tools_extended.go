package mcp

// This file intentionally left as a package anchor. The handler functions
// that previously lived here have been split by concern into:
//   - tools_extended_validation.go    (scope/commit/push validation, regression checks)
//   - tools_extended_fileread.go       (file-read tracking, attempts, three-strikes)
//   - tools_extended_halt.go           (halt conditions, halt record/acknowledge)
//   - tools_extended_production.go     (production-first, feature-creep detection)
//   - tools_extended_replacement.go    (fix verification, exact-replacement validation)
//   - tools_extended_commit.go         (pre-commit tests, commit scan, merge conflicts)
