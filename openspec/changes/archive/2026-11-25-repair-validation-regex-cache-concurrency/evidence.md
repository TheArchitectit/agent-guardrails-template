# Evidence

## Baseline

The observed legacy Jenkins Go 1.26.11 race run reported concurrent writes to `cachedRegex.lastAccess` in `mcp-server/internal/validation/safe_regex.go:26`, through `CompilePattern`, called by `SafeRegex` and `MatchPattern`. The existing corpus audit also failed to match destructive commands in that contended run. Two existing 100ms pattern validation checks timed out in the same run; those timeout symptoms are recorded separately and do not establish that the race caused them.

The affected validation source and corpus test are byte-identical between the original template base and the released #13 head. This is a pre-existing source defect.

## Verification record

The repository regression entry is `tests/regression/test_validation_regex_cache_regression_FAIL-01925f50.sh`. It runs the native Go race test and requires its named PASS line, so an unmatched `-run` expression cannot succeed silently. A disposable fake-`go` no-match control emitted successful-looking `ok ... 0.001s` output without the named PASS line; the entry rejected it with exit 1.

All Go runs below used the isolated Linux container `guardrail-audit-go:2d71e714ae06`, Go `1.26.11`, and the cached module set. The container has no host mount of the operator pilot and used `--network none` for test execution.

- Pre-fix control: `go test -race -count=1 -run '^TestCompilePatternConcurrentClear$' ./internal/validation` exited 1 and emitted `WARNING: DATA RACE`, including conflicts through `ClearRegexCache`. Sanitized local receipt: `tpl51-prefx-race.log`.
- Fixed regression via the repository entry: `bash tests/regression/test_validation_regex_cache_regression_FAIL-01925f50.sh` passed; `TestCompilePatternConcurrentClear` reported PASS in 0.01s and the package invocation exited 0.
- Focused controls repeated five times: `go test -race -count=5 -run '^(TestCompilePatternConcurrentClear|TestDangerousCommandCorpus|TestShippedPatternsCompile|TestSafeRegex_ReDoSProtection|TestMatchPattern_DefaultTimeout)$' ./internal/validation` passed. Existing timeout assertions were unchanged.
- Full validation package: `go test -race -count=1 ./internal/validation` passed in 1.074s, including the dangerous-command corpus and shipped-pattern checks.

These are focused package results; no complete MCP module suite or hosted runner acceptance is claimed by this evidence.

## Limitations

The change does not add eviction or memory bounds; it preserves the cache's existing retention behavior. Legacy hosted runner acceptance remains separate from local test evidence.
