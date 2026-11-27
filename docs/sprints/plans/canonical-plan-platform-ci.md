# Canonical sprint plan: Platform CI

> Planning baseline only. CLOSED means the canonical spec records exercised closure; this plan schedules regression/evidence review, not reimplementation. UNKNOWN means no status or closure table is recorded: first inspect implementation and evidence, then propose a narrowly scoped change only if a gap is confirmed. Do not infer status from archived task checkboxes.

## Operating model

- **Models:** use Luna for bounded implementation, testing, and evidence preparation; the lead owns design decisions, security-sensitive review, and final sign-off. These are recommendations, not execution evidence.
- **Time:** per-spec ranges below are person-hours for the listed scope. Estimates exclude external approvals, credentials, unavailable services, and unrelated repair.
- **Platform applicability:** names identify where a requirement applies. Applicability is not a claim of current runtime compatibility. Source-only, CI, and server requirements are editor-neutral unless named.
- **Acceptance:** retain the canonical SHALL clauses and named scenarios as the acceptance source. Record exact commit, command/tool versions, output hashes where useful, and failures/skips.

## Planned work

<a id="commit-attribution-records"></a>
### [commit-attribution-records](../../../openspec/specs/commit-attribution-records/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Attribution checks use complete event-scoped commit messages.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** a commit body spans several lines — a selected commit contains a keyword and the expected attribution on different lines; both lines belong to the same inspected commit and the attribution is recognized; a commit body resembles a delimiter — a selected body contains text that resembles a record separator; the workflow preserves its whole body and separately checks the next commit; the range is unavailable — the shared helper cannot resolve the event range or Git cannot read a selected commit; the job fails and reports that the inspection could not complete; the range is empty — the shared helper successfully returns an empty range; the workflow explicitly reports zero commits available for inspection.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); QA (acceptance). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (2–4h); lead decision and sign-off: 0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="gitlab-python-job-dependencies"></a>
### [gitlab-python-job-dependencies](../../../openspec/specs/gitlab-python-job-dependencies/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** GitLab Python jobs use declared dependency inputs; GitLab test dependencies are explicit.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** execute a Python validation job — GitLab runs Python validation, test, or export jobs from a clean checkout; dependency installation succeeds using a tracked manifest and the jobs preserve their configured scripts and artifacts; run Python tests in the pipeline — the GitLab Python test job installs dependencies and runs its test command; the test and coverage tools are available without relying on undeclared runner state.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); QA (acceptance). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: 0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="gitleaks-runner-installation"></a>
### [gitleaks-runner-installation](../../../openspec/specs/gitleaks-runner-installation/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Scanner scratch belongs to one invocation; Scanner integrity is verified before execution; The scan preserves event and commit coverage; Scanner use is available locally and reports safe outcomes.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** concurrent installers use one parent — two invocations share the same temporary parent or the old fixed path already exists; the installers use different directories and leave the pre-existing path intact; incorrect scanner bytes or version — the archive digest or reported scanner version disagrees with the repository pin; the command fails before scanning and does not report a clean result; a pull request advances its head — the target and head are available as pinned commits; the scanner checks the history between their merge base and the event's head, including introduced merge history; a ref is newly created or rewritten — the event has a valid head but an all-zero or unavailable ancestor boundary; the scanner checks available full history at that head and a shallow or unresolved checkout fails closed; local clean and detected fixtures — the verified command runs against a clean repository and a separate approved synthetic-detection repository; the clean run succeeds, the detection run fails with the defined finding status, and the fixture value is absent from emitted output.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); Security (threat-boundary review); QA (acceptance). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (2–4h); lead decision and sign-off: 0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="go-advisory-automation"></a>
### [go-advisory-automation](../../../openspec/specs/go-advisory-automation/spec.md) — CLOSED

- **Disposition:** regression and evidence review; keep closed unless new contradictory evidence appears.
- **Scope:** Go source-reachable findings are checked in CI.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** source-reachable advisory appears — structured scan output contains a vulnerability with a source call path; the CI advisory gate classifies the result; the job fails and reports the affected package and advisory identifier; dependency-only advisory appears — output contains only imported-package or module-only findings without a source call path; the CI advisory gate classifies the result; it reports those findings in a separate reachability category and does not label them source-reachable; scanner cannot complete — the scanner, database, toolchain, output parser, or package analysis fails or analyzes no packages; the workflow evaluates the scan result; the job fails rather than claiming a clean advisory scan.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); QA (acceptance). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (0.5–1.5h); lead decision and sign-off: 0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="go-advisory-evidence"></a>
### [go-advisory-evidence](../../../openspec/specs/go-advisory-evidence/spec.md) — CLOSED

- **Disposition:** regression and evidence review; keep closed unless new contradictory evidence appears.
- **Scope:** QA-05 evidence identifies its toolchain and scan scope.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** Evidence record preserves the tested remediation boundary — a reviewed Go advisory remediation is recorded; its evidence identifies the matching toolchain and build pins, dated before-and-after source-reachability counts, residual module-only findings, and build verification scope.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); QA (acceptance). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (0.5–1.5h); lead decision and sign-off: 0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="legacy-ci-go-toolchain-selection"></a>
### [legacy-ci-go-toolchain-selection](../../../openspec/specs/legacy-ci-go-toolchain-selection/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Legacy CI selects each module's declared Go toolchain; Legacy toolchain setup does not depend on implicit downloads.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** run MCP module tests — a legacy CI job runs Go tests from `mcp-server`; it installs or selects the toolchain declared by `mcp-server/go.mod` before invoking the existing test command; build the existing CLI module — Jenkins runs its existing team CLI build; it selects a toolchain compatible with `cmd/team-cli/go.mod` and does not assume that an example module job exists; add a future module job — a future legacy CI job builds a different Go module; its setup selects a toolchain compatible with that module's own `go.mod` directive; runner starts with an older default Go version — the runner's preinstalled Go version differs from the module's declared version; the job installs or selects the required toolchain explicitly and reports a clear setup failure if it is unavailable.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); QA (acceptance). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (2–4h); lead decision and sign-off: 0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="local-draft-ci-evidence"></a>
### [local-draft-ci-evidence](../../../openspec/specs/local-draft-ci-evidence/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Local acceptance identifies the tested Git revision and job.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** A draft workflow is queued without an assigned runner — the maintainer assigns the job to local verification; its verifiable checks run against the draft revision in a disposable local workspace; evidence lists observed results and any GitHub-only steps.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); QA (acceptance). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: 0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="openspec-catalog-identity"></a>
### [openspec-catalog-identity](../../../openspec/specs/openspec-catalog-identity/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Catalog lists spec files by unique path; Template audit work item identifiers are unique.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** Catalog covers all specification paths — the repository specification inventory is published; each current specification file appears once by full path and overlapping short names remain distinguishable; New audit proposals coexist with archived audit work — an architecture proposal is added to the template audit register; its identifier is distinct from every existing register row, resolves to the same proposal across the integration audit and verification queue, and leaves prior archive identifiers attached to their original evidence.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); QA (acceptance). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: 0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="openspec-strict-validation"></a>
### [openspec-strict-validation](../../../openspec/specs/openspec-strict-validation/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Strict validation covers every specification.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** Strict validation covers all specs — the repository strict-validation procedure runs; every current specification is checked and each scenario-format failure is reported separately from implementation status.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); QA (acceptance). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: 0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="python-test-suite-inventory"></a>
### [python-test-suite-inventory](../../../openspec/specs/python-test-suite-inventory/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Every discovered suite has an executable inventory row.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** Root tests suite is covered by the inventory — the repository contains collectable tests directly under `tests/`; the Python suite discovery gate checks the inventory; it finds a row for `tests/` with a runnable pytest command and positive floor no higher than the current collection; Unregistered discovered suite fails coverage — a collectable Python test directory has no inventory row; the Python suite discovery gate checks coverage; it exits nonzero and identifies the unregistered suite; Collection floor tracks a nonempty suite — the root `tests/` suite has a current collection of 11 cases, including any conditionally skipped Docker integration test; CI collects and runs that suite; the inventory floor is 11 and the ordinary pytest command runs the collected suite, allowing only the Docker case's declared skip condition.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); QA (acceptance). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (2–4h); lead decision and sign-off: 0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="regression-guard-committed-diff"></a>
### [regression-guard-committed-diff](../../../openspec/specs/regression-guard-committed-diff/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** CI evaluates the committed change range; Local mode evaluates each selected patch.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** committed match in a pull request — a temporary repository has a committed base and a head commit that adds a benign string matching a registered regression rule; CI invokes the guard with that base and head; the guard reports the changed file and matching rule; clean committed range — the base-to-head range contains no changed file matching a registry or pattern rule; CI invokes the guard with the resolved range; it reports no potential regression after evaluating the non-empty range; divergent PR uses the unique merge base — the event base advanced after the PR head forked, and a benign matching sentinel exists only in a base-side commit; CI invokes the guard in PR merge-base mode with the event's exact base and head revisions; it evaluates the unique common ancestor to PR-head patch and does not report the base-only sentinel; ambiguous or shallow PR history — the PR revisions have no unique merge base or the checkout is shallow; CI invokes the guard in PR merge-base mode; it exits nonzero with a history error; push base is not an ancestor — the push before revision does not precede the pushed head; CI invokes the guard in push range mode; it exits nonzero instead of scanning a misleading range; missing base revision — the CI event base does not resolve or checkout history is incomplete; the guard is invoked; it exits nonzero with the unavailable-range cause instead of reporting a green empty scan; local staged and unstaged review — a developer has staged or unstaged edits in a working tree; the developer invokes the existing local mode; the guard continues to evaluate those edits without requiring event SHAs; combined local mode sees unstaged pattern content — a benign registered pattern is added only in an unstaged file; a developer invokes `--all`; the guard evaluates the unstaged patch and reports the matching rule; forbidden context is patch-local — the staged patch adds a matching sentinel and the unstaged patch adds only its forbidden-context marker; a developer invokes `--all`; the staged match is still reported because each patch is evaluated independently.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); Security (threat-boundary review); QA (acceptance). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (2–4h); lead decision and sign-off: 0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="regression-guard-step-output"></a>
### [regression-guard-step-output](../../../openspec/specs/regression-guard-step-output/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** PR commit analysis emits valid consumed outputs.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** mixed commit subjects — a PR subject list contains ordinary and conventional fix subjects; the analysis step writes its GitHub Actions outputs; `bug_fix_count` equals the number of matching fix subjects and each output record is a valid single line; no unused subject-list output is emitted; regression guard test is included in Python suite execution — the focused regression is stored under `tests/regression/`; CI discovers and runs its declared Python test suites; `tests/regression` has an inventory row with a positive test floor and a command that executes that suite.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); Security (threat-boundary review); QA (acceptance). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: 0.5–1h). Estimate excludes unavailable approvals or infrastructure.
