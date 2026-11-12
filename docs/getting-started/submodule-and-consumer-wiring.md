# Submodule Embedding & Consumer Wiring

> How to consume Agent Guardrails Template as a tooling checkout rather than
> copying advisory files into a target repo — and why the checkout must stay
> **local** since `PREVENT-SUB-001`.

---

## Purpose

This repository can be consumed two ways. The **copy-in** mode (advisory files
copied into a consumer repo, step-by-step in
[how to apply the project](how-to-apply.md)) installs guardrail instructions
and supporting material as plain files in the target repository. A tooling
checkout keeps the framework's own gates, CI, and self-update path available
next to the consumer.

> **`PREVENT-SUB-001` blocks the committed embed.** A change to `.gitmodules`
> naming `agent-guardrails-template` (or `DevGate-Agentic-Framework`) is a
> `critical` pattern violation. v3.8.3 of this document described committing
> the checkout as the embed contract; that instruction is withdrawn. The
> checkout is **local tooling only** — keep it out of the consumer repo and
> push only the consumer's own code fixes.

---

## Choose the consumption mode

| | Copy-in | Local tooling checkout |
|---|---|---|
| What lands in the consumer repo | Advisory files (CLAUDE.md, `.claudeignore`, docs, templates) | Nothing — the checkout is gitignored |
| Enforcement | Advisory only; does not block an agent by itself | Framework gates/CI run against the checkout |
| Version control | Consumer owns the copies | Re-pull the checkout to advance; record the SHA outside the repo if you need a rollback point |
| When to choose | Quick adoption; no desire for framework CI history | Want the framework's gates available without committing them |
| Committed submodule | n/a | **Blocked by `PREVENT-SUB-001`** |

---

## Keep a local tooling checkout

From the consumer repository root:

```sh
git clone https://github.com/TheArchitectit/agent-guardrails-template.git .agent-guardrails
echo '.agent-guardrails/' >> .gitignore
```

- Suggested checkout path: `.agent-guardrails` (once chosen it should stay
  fixed, because the consumer's gate scope names it).
- Do **not** `git submodule add`, and do **not** `git add .agent-guardrails`.
  There is no pin in the consumer repo and no `.gitmodules` entry to review.
- Advance by pulling (`git -C .agent-guardrails pull`); CI must clone the
  checkout itself as a workflow step, pinned to a SHA if you want reproducible
  gate results.

---

## The scope lesson

With a checkout in place, any quality-gate scanner the consumer runs over their
own repository — e.g. DevGate from the
[AIGGP Agentic Framework](https://github.com/TheArchitectit/AIGGP-Agentic-Framework)
at `.devgate/` — must be told to skip the checkout's directory.

Node walkers prune by `skip_dirs` name lists, not by dot-prefix. An unlisted
hidden directory such as `.agent-guardrails` is scanned as first-party source.
A consumer measured this: 162 pattern violations and 107 warnings against this
repo's own vendored code and zero against their own code, until the directory
was named in `skip_dirs`.

**Rule:** after adding the checkout, add the directory name to the consumer's
gate scope before the first gate run.

---

## Coexistence with DevGate

| Framework | Role | Location |
|---|---|---|
| DevGate | Enforcement gates: pattern, semantic, regression, file-size, charset, manifest-join | `.devgate/` (local checkout) |
| Agent Guardrails Template | Advisory agent wiring: instructions, skills, hooks, docs, policy inputs | `.agent-guardrails/` (local checkout) |

- The two directories are distinct; do not merge them.
- Neither is committed. `PREVENT-SUB-001` blocks a `.gitmodules` entry naming
  either repository, so both are gitignored local checkouts.
- DevGate's repository and its documentation, including
  [`SUBMODULE_WIRING.md`](https://github.com/TheArchitectit/DevGate-Agentic-Framework/blob/main/docs/SUBMODULE_WIRING.md),
  cover its side of the contract.

---

## Living with the checkout

- Never edit files inside the checkout; upstream owns them.
- Never commit or push the checkout, and never `git add` it — keep it gitignored.
- Run parent-repo git commands from the parent repository root, not from inside
  the checkout, so a stray `git add -A` cannot sweep the tooling into a commit.
- Advance by pulling (`git -C .agent-guardrails pull`); CI clones the checkout
  itself as a workflow step.

---

## Verify the wiring

Consumer-side sanity checks:

- `ls .agent-guardrails/scripts` (and `.devgate/scripts`) is non-empty — the
  checkouts are present, not empty directories.
- Neither directory is tracked: `git check-ignore -q .agent-guardrails` and
  `git check-ignore -q .devgate` both exit 0.
- `git ls-files .gitmodules` is empty — no tooling submodule registration.
- The gate scope names both directories.
- The advisory doctor reports its honest `UNKNOWN` / `PASS` / `FAIL` states:

```sh
bash scripts/guardrails-cli.sh doctor /path/to/project
```
