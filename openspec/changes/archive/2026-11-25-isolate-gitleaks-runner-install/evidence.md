# TPL-38 verification

## Earlier licensed Action approach

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

## Direct CLI baseline on the laptop

The maintainer selected direct CLI execution and local verification on
2026-11-25. This replaces the licensed GitHub Action contract in the plan above.

The official Windows x64 archive for Gitleaks 8.24.3 matched SHA-256
`3f1a35578631dbfe633cc5b49e6c906e55ff14a4bfd7336a10fb27fe33b6dcd2`.
The extracted executable reported 8.24.3. On the laptop it scanned the 12
commits from the PR #13 merge base through
`109c95213a1a64518707dcfd27d3b81a2433021d` with full redaction. It exited 0
with zero findings. Scanner output was captured locally.

On the same laptop's Docker Desktop Linux engine, the official Linux x64
archive matched the pinned digest and the executable reported 8.24.3.
Separate disposable Git repositories exercised a clean result with exit 0
and an isolated synthetic detection with exit 2. The harness emitted status
only. These results precede the shared workflow command implementation.

## Verified command acceptance

The repository command `.github/scripts/gitleaks-verified.sh` was run in
Docker Desktop's Linux x64 engine against a disposable full-history clone of
this draft. It downloaded the official 8.24.3 release archive, verified the
Linux x64 SHA-256 pin
`9991e0b2903da4c8f6122b5c3186448b927a5da4deef1fe45271c3793f4ee29c` before
extraction, and confirmed the executable reports `8.24.3`. The pinned range
from PR #13 base `943bcf65055128e338b696dc40d74e79ac33449e` through draft head
`109c95213a1a64518707dcfd27d3b81a2433021d` contains 12 commits; the wrapper
completed with exit 0 and zero findings.

A separate disposable Linux Git repository with one synthetic detector fixture
completed through the same verified wrapper with exit 2 (finding detected).
The canary value was absent from captured stdout and stderr. The scanner's raw
output remained inside its private temporary directory, which the wrapper
removed after completion.

`tests/test_gitleaks_temp_isolation.py` passed all 9 focused tests in a
Python 3.12 Alpine container. The tests exercise concurrent private scratch,
preservation of a pre-existing sentinel, real digest rejection of corrupt
bytes, a wrong-version executable, path-scanner non-use, clean/detected exit
handling, output redaction, local equal and malformed ranges, shallow and
unrelated history refusal, ordinary push ranges, new-ref and rewritten-push
head-history fallback, and PR merge-history coverage. OpenSpec CLI 1.14.3
strict validation passed all 62 items.

## Independent laptop review

The independent replay passed all nine wrapper tests and Bash syntax checking.
The exact event entry point verified the official archive and returned exit 0
with zero findings on the local candidate. The published 12-commit range also
returned exit 0 with zero findings from the Windows CLI and the Linux wrapper.
The workflow retains the required `Run Gitleaks` step name. The mandatory-row
gate and all 18 of its controls passed. OpenSpec 2.2.0 strict validation
passed on the combined draft candidate.

The five GitHub Actions workflows for the published draft head had no assigned
runner at the 19:03 UTC check. The recorded scanner results come from this
laptop.
