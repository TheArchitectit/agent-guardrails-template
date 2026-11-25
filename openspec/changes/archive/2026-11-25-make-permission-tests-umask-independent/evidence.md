# QA-02 verification

Four tests explicitly set and verify mode `0644` for invalid permission
fixtures. In a disposable Linux container, exact Go `1.26.11` tests passed
for `internal/auth`, `internal/config`, `internal/secretfile`, and
`internal/mutationkill` under umasks `0022` and `0077`. Both runs used
`-count=1` to bypass Go's test-result cache. Mutation suite durations were
285.181 s (`0022`) and 135.944 s (`0077`). Production owner-only file mode
assertions passed in both runs.
