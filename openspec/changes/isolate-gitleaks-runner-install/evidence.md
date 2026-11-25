# TPL-38 verification

Two focused pytest checks passed on Linux. The Windows `core.autocrlf=true`
checkout had translated the Bash helpers to CRLF, causing Bash to reject its
`pipefail` option; `.github/scripts/*.sh` now has an LF checkout rule.
Concurrent jobs receive distinct temporary directories and clean only their
own child, including when the runner temp parent is a symlink. The exact pinned
action reads its install location from Node's temporary directory, and its
existing license, token, event range, redaction, exit-code, and upload
configuration are preserved.

Separately, the official `zricethezav/gitleaks:v8.24.3` CLI image reported
`v8.24.3`; a mounted benign fixture exited 0 and a synthetic token-shaped
canary exited 2. Both runs used `--redact`, suppressed scanner output, and did
not print fixture contents. This validates the local CLI scan behavior; it
does not exercise the pinned GitHub Action installer or hosted action path.

Source review of the pinned action installer at
`ff98106e4c7b2bc287b24eaf42907196329070c7` confirms it restores a versioned
cache or downloads and extracts the release without checking its checksum or
running a post-install version command. Owner verification found the current
upstream v3.0.0 installer has the same gap. The pinned action's integrity
replacement/equivalence decision and hosted rerun remain open; the licensed
action semantics are preserved pending that review.
