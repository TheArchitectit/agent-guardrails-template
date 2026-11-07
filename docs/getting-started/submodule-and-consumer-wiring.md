# Submodule Embedding & Consumer Wiring

> How to consume Agent Guardrails Template as a vendored checkout rather than
> copying advisory files into a target repo — including coexistence with a
> second framework gate submodule (DevGate).

---

## Purpose

This repository can be consumed two ways. The **copy-in** mode (advisory files
copied into a consumer repo, step-by-step in
[how to apply the project](how-to-apply.md)) installs guardrail instructions
and supporting material as plain files in the target repository. The
**embed** mode consumes the repository itself as a vendored checkout — a git
submodule — so the framework's own gates, CI, and self-update path stay
available under version control alongside the consumer.

This document is the embedding contract for the embed mode.

---

## Choose the consumption mode

| | Copy-in | Submodule / vendored embed |
|---|---|---|
| What lands in the consumer repo | Advisory files (CLAUDE.md, `.claudeignore`, docs, templates) | A git submodule checkout at a fixed path |
| Enforcement | Advisory only; does not block an agent by itself | Framework gates/CI run against the checkout via the consumer's own gate tooling |
| Version control | Consumer owns the copies | Submodule pin tracks upstream `main` |
| When to choose | Quick adoption; no desire for framework CI history | Want framework gates, self-update path, and upstream provenance available |

---

## Embed as a submodule

From the consumer repository root:

```sh
git submodule add https://github.com/TheArchitectit/agent-guardrails-template.git .agent-guardrails
git submodule update --init --recursive
```

- Suggested checkout path: `.agent-guardrails` (any fixed path may be chosen, but once chosen it should stay fixed).
- Add the submodule and the pin to git, then commit the parent repo:

```sh
git add .agent-guardrails
git commit -m "chore: embed agent-guardrails-template v3.8.2"
```

- The branch to track is `main`; the pin records the upstream commit SHA.

---

## The scope lesson

When embedded, any quality-gate scanner the consumer runs over their own
repository — e.g. DevGate from the
[AIGGP Agentic Framework](https://github.com/TheArchitectit/AIGGP-Agentic-Framework)
at `.devgate/` — must be told to skip the embedded checkout's directory.

Node walkers prune by `skip_dirs` name lists, not by dot-prefix. An unlisted
hidden directory such as `.agent-guardrails` is scanned as first-party source.
A consumer measured this: 162 pattern violations and 107 warnings against this
repo's own vendored code and zero against their own code, until the directory
was named in `skip_dirs`.

**Rule:** after embedding, add the checkout directory name to the consumer's
gate scope before the first gate run.

---

## Coexistence with DevGate

| Framework | Role | Location |
|---|---|---|
| DevGate | Enforcement gates: pattern, semantic, regression, file-size, charset, manifest-join | `.devgate/` (submodule) |
| Agent Guardrails Template | Advisory agent wiring: instructions, skills, hooks, docs, policy inputs | `.agent-guardrails/` (submodule) |

- The two directories are distinct; do not merge them.
- DevGate's repository and its documentation, including
  [`SUBMODULE_WIRING.md`](https://github.com/TheArchitectit/DevGate-Agentic-Framework/blob/main/docs/SUBMODULE_WIRING.md),
  cover its side of the contract.

---

## Living with the embed

- Never edit files inside the checkout; upstream owns them.
- Never commit or push inside a submodule checkout of this repository.
- Keep submodule URLs as the upstream HTTPS ones, so CI checks out submodules
  anonymously over the recorded URL.
- Run parent-repo git commands from the parent repository root, not from inside
  the checkout.

---

## Verify the wiring

Consumer-side sanity checks:

- `git submodule status` shows both pins.
- The gate scope names both directories.
- The advisory doctor reports its honest `UNKNOWN` / `PASS` / `FAIL` states:

```sh
bash scripts/guardrails-cli.sh doctor /path/to/project
```
