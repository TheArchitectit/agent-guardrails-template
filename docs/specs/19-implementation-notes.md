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
