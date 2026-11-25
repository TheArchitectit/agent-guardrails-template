# Design

## Installation and trust

Allocate a fresh child of the caller's temporary directory for each invocation. The installed scanner, downloaded archive, and scan output stay inside that child. Clean up only the child allocated by the current process, including failure paths.

Keep Gitleaks 8.24.3 to preserve detector behavior. For the supported Linux x64 runner and Linux Docker Desktop environment, pin the official release archive SHA-256 to `9991e0b2903da4c8f6122b5c3186448b927a5da4deef1fe45271c3793f4ee29c`. Compare the bytes against that reviewed digest before extracting the one scanner executable. Execute its absolute path and require the exact reported version, permitting the release's optional `v` prefix. Fail closed for an unsupported platform, a missing prerequisite, installation failure, or an unexpected version. The version check runs only after the digest check. The normal path does not use a runner cache or a scanner found on PATH.

The Windows x64 asset can be verified and run directly on the laptop for independent evidence. The reusable script is exercised locally through the Linux engine and uses the same Linux x64 release as CI.
[Official release](https://github.com/gitleaks/gitleaks/releases/tag/v8.24.3) · [Official checksum list](https://github.com/gitleaks/gitleaks/releases/download/v8.24.3/gitleaks_8.24.3_checksums.txt)

## Commit scope and results

Keep the existing push and same-repository pull-request triggers and full-history checkout. The only workflow authorization required is read-only checkout access. Read GitHub event objects as data and pass validated full commit IDs to the scan command. For a pull request, find the merge base of the target and head and scan all commits introduced at the head. This includes the former first-parent coverage and retains commits brought in through merges. For an ordinary push, scan from the before commit through the after commit. For a new ref with an all-zero before commit or a rewrite that lacks an available ancestor, scan the available head history. An invalid event, unresolved head, or unavailable history fails closed. If the proven base equals the head, scan the head commit. For a local invocation, accept an explicit pair of pinned commits and establish its merge base.

Use complete redaction and the scanner's detection exit value of 2. A finding or a scanner error fails the command. The job summary exposes the result only. The wrapper does not upload reports or emit scanner output containing file content. Local success records which history was scanned and cannot substitute for a GitHub job result.

## Verification and rollback

Use disposable Git repositories for healthy, detected, corrupt archive, wrong version, malformed range, concurrency, and cleanup controls. Ensure a matching scanner version on PATH cannot substitute for the verified download. Use safe synthetic fixtures and capture status without their values. Run the real pinned CLI on the laptop against the draft range. Inspect the actual workflow job or validate its parsed shape for trigger, range, and permissions. A rollback must preserve an isolated, fail-closed scan.
