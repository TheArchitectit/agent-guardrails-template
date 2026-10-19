# Spec 19 implementation notes

Companion to
[`19-phase0-ci-security-gates.md`](19-phase0-ci-security-gates.md).
The dated implementation notes are split out so that no Markdown file exceeds
the 500-line limit — Spec 19 R19-11. Section numbers are kept from the spec so
existing references stay valid.

## 6. Implementation note (2026-10-15) — R19-15 action pinning

R19-15 requires required workflow actions to use reviewed full commit SHA
references, with a test that validates them and a mutation fixture that changes
a pin to a tag and fails. That half is now implemented and enforced:

- Every third-party action in all six workflows under `.github/workflows/` is
  pinned by a 40-hex commit SHA with the human tag in a trailing comment
  (`actions/checkout` `11d5960a…` v4.4.0, `actions/setup-python` `a26af69b…`
  v5, `actions/setup-go` `40f1582b…` v5, `actions/github-script` `f28e40c7…`
  v7, `actions/upload-artifact` `ea165f8d…` v4.6.2, `codecov/codecov-action`
  `ab904c41…` v3.1.6, `gitleaks/gitleaks-action` `ff98106e…` v2). Each SHA was
  resolved live via the GitHub API and re-verified to resolve to that commit.
- `scripts/check_action_pins.py` enforces the rule on every push: fail-closed
  (an unreadable workflow is a violation, not a skip) and non-vacuous (zero
  workflows, or zero `uses:` refs, FAILS). Local actions (`./…`) and
  digest-pinned container actions are allowed; a mutable tag or a bare
  `owner/repo` is a violation. The allowlist is empty by design.
- `scripts/test_check_action_pins.py` is the mutation fixture the requirement
  names: it asserts the gate changes PASS → FAIL on a mutable tag, a missing
  ref, and an unpinned container, and PASS on a full-SHA pin and a local action.
- The gate + self-test run in `.github/workflows/regression-guard.yml` on every
  push and PR (the step "Action pinning (Spec 19 R19-15 / Spec 17 R17-11)").

Remaining R19 rows (external required-check configuration, negative-control
matrix, UCS03 isolation attestation) stay **proposed** — this note closes only
the action-pinning control, and does not assert that the full Phase 0 matrix is
enforced. No gate was weakened.

## 7. Implementation note (2026-10-15) — R19-15 image-digest half

R19-15 also requires that release/base **images** use immutable digests. That
half is now implemented and enforced (see Spec 17 R17-11 Sec. 5 for the full
record): every external image ref in `docker-compose.yml`,
`mcp-server/deploy/{podman-compose,docker-compose.example}.yml` and
`mcp-server/deploy/Dockerfile` is pinned by an immutable `@sha256` digest, and
`image-pins.json` is the reviewed `name:tag` → digest record each manifest must
agree with.

- `scripts/check_image_pins.py` enforces the rule on every push (fail-closed,
  non-vacuous, empty-of-third-parties allowlist with a reason per entry).
- `scripts/test_check_image_pins.py` is the mutation fixture: a tag-only ref, a
  mismatched digest, and an unreviewed pin must each change PASS → FAIL.
- Both run in `.github/workflows/regression-guard.yml` on every push and PR (the
  step "Image-digest pinning (Spec 19 R19-15 / Spec 17 R17-11)").

With this, R19-15's two named controls (workflow actions by full SHA; images by
digest) are both enforced. The external required-check configuration, the
negative-control matrix and the UCS03 isolation attestation remain **proposed**;
this note does not assert the full Phase 0 matrix is enforced. No gate was
weakened.

A third fail-closed gate now runs in the same `regression-guard.yml` job —
`scripts/check_secret_argv.py` (Spec 17 R17-08): a credential in a container
manifest's argv (`redis-cli -a`, `--requirepass <pw>`) fails the build. It is
the Phase 0 "no secret in process arguments" control; see Spec 17 Sec. 6.

## 9. Implementation note (2026-10-19) — R19-13 runner trust and repository isolation

R19-13 ends with a sentence that decides how the whole row has to be built:

> **"A runner label alone is not isolation evidence. Release acceptance MUST
> include an isolation probe or recorded host attestation."**

Before this note, the requirement was satisfied by prose. `.github/workflows/team-validation-fleet.yml`
carried a comment above its `runs-on:` line explaining why the self-hosted pool
is safe, and nothing else — and **a comment cannot fail a build**. The label
says where a job was *routed*; it says nothing about whether the job shared a
workspace, a cache, a token, or a tree with an earlier job on the same host.

**Two defects were live in the tree when this pass measured it.**

1. **The trusted job had no post-job cleanup step.** The UCS03 pool is a
   persistent host, so a failed or cancelled run left `.teams/example.json` and
   `teams-export.json` in the workspace for the next run — which may be a
   different commit. R19-13 names post-job cleanup explicitly. Added as the last
   step, `if: always()`.
2. **Nothing measured the host at all.** Clause 5 was unimplemented: no probe,
   no attestation, no recorded evidence of any kind.

**Fixed with one gate and one probe, split along the line between what can be
read and what must be measured.**

`scripts/check_runner_isolation.py` reads the definitions and enforces six
properties, fail-closed and non-vacuous:

- **Fork code never reaches the private pool.** A workflow declaring
  `pull_request` may only have GitHub-hosted jobs. An expression `runs-on:`
  (e.g. `${{ inputs.runner }}`) is a violation rather than an assumption,
  because its value is not decidable at review time and an input can resolve it
  to the self-hosted label.
- **The label appears in exactly one file**, in a job guarded by
  `refs/heads/main`.
- **Least-privilege token** — an explicit `permissions` block with no write
  scope. An omitted block is refused, because the omission resolves to the
  repository default rather than to the read-only token the job should hold.
- **Bounded concurrency** — a `concurrency.group` with `cancel-in-progress`
  declared, not implied.
- **No cross-repository reuse** — no `actions/cache` (a cache on a persistent
  host is restored by key across runs and repositories), no checkout `path:`,
  no checkout `repository:`.
- **Cleanup last** — an `if: always()` cleanup step, and it must be the final
  step, because cleanup that runs before other work does not clean up after it.

`scripts/runner_isolation_probe.py` runs **inside the trusted job** and measures
the host: the workspace is a single git top-level at the run's commit; its
parent holds no other repository tree; the runner work root holds no other
repository's checkout; the workspace is not group/world writable; and no
`.env`-shaped file in the checkout root carries a non-empty assignment. It
writes `runner-isolation/v1` to `$RUNNER_TEMP` (outside the checkout, so the
evidence does not become the residue), and a violated property fails the job.
`check_runner_isolation.py --attestation` then validates that record with the
same code that reads the definitions, so the two cannot disagree.

**The cross-repository check is written against the real Actions layout, and the
first version was wrong.** A runner lays checkouts out as
`<runner>/_work/<repo>/<repo>`, so every *other* repository sits one level
*above* the workspace parent. Scanning only the parent reported a clean scope on
a runner that hosts the whole fleet. The probe now also examines the work root,
and only in that recognised `_work/<repo>/<repo>` shape — so an ordinary `/tmp`
full of scratch clones is not mistaken for a runner workspace. Both directions
are pinned in the self-test.

**Measured, not asserted.** On this host the probe correctly **refused** a
scratch clone at `/tmp/gr-w` (shared `/tmp`, 157 sibling repository trees,
mode `0o775`) and **passed** a genuinely isolated workspace
(`_work`-shaped, `0o700`, one repo, matching HEAD). Both records were then fed
to `--attestation` and produced the corresponding verdict from the gate.

**Evidence.** `scripts/test_check_runner_isolation.py` — 27 cases (13 trusted-lane
and hosted-runner rules, 7 attestation shapes, 3 non-vacuity cases, plus the
main-tree run). `scripts/test_runner_isolation_probe.py` — 14 cases over **real
git trees** built by the test, covering each property in both directions and all
three exit codes. **Red-on-drift proven four ways against the real workflows**,
each measured `EXIT=0 → EXIT=1` then restored: cleanup step removed; `contents:
read` → `write`; `pull_request` added to the trusted workflow; the `main` guard
removed. No gate was weakened and no check was skipped.

**Wired.** `regression-guard.yml` runs the gate + self-test on every push and PR
(step "Runner trust and repository isolation (Spec 19 R19-13)").
`team-validation-fleet.yml` runs the probe then the attestation validation
before its cleanup step.

**Named limits, recorded rather than hidden.** The credential scan is a
root-level `.env` heuristic, not a secret sweep (`secret-validation.yml` owns
that). The probe measures the workspace, not the host kernel, network namespace
or container runtime. And **the first trusted-lane run on `main` is the real
test of clause 5** — a probe that passes locally has still not proved the UCS03
host; this note does not claim that run has happened.

---

## 10. Implementation note (2026-10-19) — R19-14 path-filter safeguards and mandatory-row reachability

**What was measured first.** R19-14 was the one Phase 0 row with no automated
half. Nothing in the tree read a `paths:` filter, and nothing asserted that the
six classes it names with MUST were reachable from a protected push. Three
defects were live in `guardrails-lint.yml`, and the third is the one that
matters:

1. It ran **only** on `pull_request`, branch-filtered to `main`/`develop`. A
   direct push to `main` never executed any of its four jobs. (`gh run list
   --workflow guardrails-lint.yml` shows five runs, every one a
   `pull_request`; the last was 2026-08-23.)
2. It computed its change set as `git diff --name-only origin/main...HEAD` while
   also triggering on `develop`, so a `develop` PR was compared against the wrong
   branch.
3. `|| git diff --name-only HEAD~1` is a **narrowing** fallback. When the real
   diff fails — shallow checkout, missing base ref, filter error — it scans the
   tip commit only and the job still reports green. R19-14 says a filter error or
   missing diff is `NOT_RUN`/FAIL, not an automatic skip.

**Also fixed in the same file.** The `check-forbidden-files` job passed **glob**
patterns to **`grep -E`**. `*.pem` and `*.key` begin with `*`, which GNU grep
warns about and handles as undefined behaviour; and `.env` as an ERE means "any
character then `env`", which matches `docs/.env.example`, `src/config.env.md`
and `.envrc`. The check therefore refused `.env.example` — the exact file its
own failure message tells contributors to use. Replaced with
`scripts/check_forbidden_files.py`: basename-exact, permits only `.env.example` /
`.env.sample`, and refuses a new `.env.*` variant by default rather than
silently permitting it.

**The gate.** `scripts/check_mandatory_rows.py` asserts three independent
properties, and each is separately killable:

- **Filter discipline.** No workflow may declare `paths:` / `paths-ignore:`.
  Narrowing is legal only behind a dependency manifest mapping every changed path
  to every affected gate; a filter in `on:` cannot express that mapping, so it is
  refused rather than assumed sound. `.guardrails/mandatory-ci-rows.json`
  declares the full-matrix escalation list, and a missing or empty required path
  class is a violation.
- **Reachability.** Each mandatory class must resolve to an existing step inside
  a workflow triggered by a protected push. A step that exists only in a
  `pull_request`-only workflow is a VIOLATION — a reader sees enforcement and
  `main` never runs it.
- **Event-correct diffing.** Executable `run:` blocks are scanned (the YAML
  parser keeps comments out of the document, so a note quoting the old broken
  command cannot fail the gate and train the next maintainer to delete the
  explanation): no hardcoded base its triggers contradict, no single-tip-commit
  fallback.

`scripts/changed_files.sh` and `scripts/changed_commits.sh` replace the one-line
diff. They derive the range from the event (`pull_request` → three-dot against
the PR base; `push` → the pushed range via `github.event.before`), deepen a
shallow clone before deciding, and **exit non-zero** when an honest set cannot be
computed. A push with no usable before-SHA and no default branch is refused
rather than reported as "no changes".

**Evidence.** `scripts/test_check_mandatory_rows.py` — 18 cases, one mutation per
case against a tree that otherwise passes, plus the real-tree run.
`scripts/test_check_forbidden_files.py` — 16 cases; the permission cases are the
paths the old matcher refused, so a regression to the old behaviour turns the
suite red. **Red-on-drift proven on the unfixed tree:** the gate returns
`EXIT=1` naming the hardcoded base and the tip-only fallback, and `EXIT=0` after
the fix. `regression_check --all --pre-commit` exit 0; all repo gates green
(action pins 88, tool count 50/22/72 ×10, resources 13, doc links 968/458, doc
length 458/10, inventory 23/17, image pins 8, secret argv 16, pack lock OK, Go
modules 3/3, Python suites 2/2).

**Named limits.** The gate reads workflow YAML; it does not query GitHub for the
repository's actual branch-protection or required-check configuration, so
R19-13's external required-check list and R19-16 release blocking remain
owner-configured items outside code. The escalation manifest is a declaration:
it states which path classes must select the full matrix, and the gate verifies
the declaration is complete, not that a future `paths:` filter would honour it.
That is why a filter is refused outright instead of being validated against the
manifest.
