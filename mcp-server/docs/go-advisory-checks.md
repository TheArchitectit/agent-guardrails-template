# Go advisory checks

The recurring advisory job scans this Go module on pull requests and `main`/`master` pushes. It takes the Go toolchain from `go.mod`, pins govulncheck to `v1.10.0`, and uses `GOTOOLCHAIN=local` so the scan cannot silently select another toolchain. A nonzero scanner exit or invalid/incomplete JSON fails the job. Findings with a source call path fail; imported-package and module-only findings are reported separately.

Run the same scan locally from the repository root:

```sh
cd mcp-server
export GOTOOLCHAIN=local
go install golang.org/x/vuln/cmd/govulncheck@v1.10.0
govulncheck -json ./... > /tmp/govulncheck.json
python3 scripts/govulncheck_gate.py --input /tmp/govulncheck.json --report /tmp/govulncheck-summary.json
```

The checker writes a redacted summary containing advisory IDs, reachability classes, and package names. It omits function traces and source locations. Exit status `1` means a source-call-path finding was reported; status `2` means the scan output is invalid or lacks evidence that packages were analyzed. The scanner command itself must also exit successfully before classification runs.

Run the sanitized classifier fixtures from the repository root with:

```sh
python3 -m pytest mcp-server/tests/test_govulncheck_gate.py -q
```

The historic QA-05 evidence is a dated point-in-time result, not a current status promise. Module-only findings remain visible in scan summaries and do not imply source-call reachability or standardized severity.
