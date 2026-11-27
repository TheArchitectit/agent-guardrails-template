# Local Secret Validation and Guardrails Lint

Run the event-scoped scans and checks from a clean Linux x86-64 checkout:

```bash
scripts/run_local_event_gates.sh \
  --event push \
  --base-sha <event-before-sha> \
  --head-sha <event-after-sha>
```

For a pull request, pin both event commits and the base branch ref:

```bash
scripts/run_local_event_gates.sh \
  --event pull_request \
  --base-ref main \
  --base-sha <pull-request-base-sha> \
  --head-sha <checked-out-github-sha> \
  --pr-head-sha <pull-request-source-sha>
```

`--head-sha` must equal the checked-out `HEAD`. For PRs,
`origin/<base-ref>` must resolve to `--base-sha` after the existing range
helper has attempted its normal base fetch, and the PR source commit must be
present. This lets `scripts/changed_files.sh` and `scripts/changed_commits.sh`
compute their normal event ranges. If history is shallow, the existing
`changed_files.sh` attempts to unshallow it; the helper then verifies that the
explicit base commit is available and fails if it is not. A dirty checkout,
missing/mismatched refs, unavailable PR head, or unsupported event fails. It
does not guess a base or turn an uncomputed range into an empty pass. For a
first push, an all-zero before SHA is accepted only when `origin/main` or
`origin/master` is present.

## What it runs

- The workflow's pinned Gitleaks script at `.github/scripts/gitleaks-verified.sh`
  with a temporary event payload built from the supplied SHA values. It downloads
  Gitleaks 8.24.3, verifies its pinned SHA-256, and scans the same push or PR
  range rules as CI.
- Secret Validation's `.env` filename scan, credential-file patterns and
  exact-path synthetic-fixture allowlist, and hardcoded-secret regex patterns
  with the workflow's file-type, directory, and allowlist filters. Local
  hardcoded-pattern output lists filenames only, not matched lines.
- Guardrails Lint's changed-file scope summary, forbidden-file matcher and its
  self-test, commit-message format scan, and AI-attribution inspection. The
  commit-message scan is advisory in the workflow and remains advisory here.

The script requires Bash 4+, Git, Python 3, and standard Linux utilities
(`grep`, `find`, `cut`, `mktemp`, `curl`, `sha256sum`, `tar`). It needs network
access to fetch missing Git history/base refs and download the digest-pinned
Gitleaks release, but no GitHub token or repository secret. An ignored `.devgate`
checkout may be present: the workflow's hardcoded-secret grep excludes that
directory, while its `.env` and credential filename scans use whole-tree `find`
commands and therefore also inspect it. The local helper preserves those
current workflow filters.

This command does not create GitHub Actions artifacts or step summaries, post
PR comments, or test GitHub's actual event payload and runner assignment. For a
PR, using the source branch as both `--head-sha` and `--pr-head-sha` validates
that source commit; using the GitHub PR merge commit as `--head-sha` also checks
the merged tree while `--pr-head-sha` keeps Gitleaks scoped to the event's PR
source commit. A missing source commit or nonstandard merge arrangement fails
closed.
