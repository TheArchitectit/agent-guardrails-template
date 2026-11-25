# QA-02 verification

Four tests explicitly set and verify mode `0644` for invalid permission
fixtures. Fresh Go 1.26.11 tests passed for `internal/auth`,
`internal/config`, and `internal/secretfile` under umasks `0022` and
`0077`. The complete acceptance tasks also name the mutation suite.
That suite was not run, so both combined tasks remain open.
