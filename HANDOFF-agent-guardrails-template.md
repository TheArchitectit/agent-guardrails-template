# HANDOFF — agent-guardrails-template (DevGate)

> **Project:** `agent-guardrails-template`  ·  **Default branch:** `main`
> **Scope:** DevGate consumer guardrails: gates, workflow templates, runner standard, monitor hub
> **This file:** `HANDOFF-agent-guardrails-template.md` — a `HANDOFF-<project>` file. If the project name in this
> filename does not match the work you are doing, you have the wrong handoff.
> **Spec index:** [`openspec/INDEX.md`](openspec/INDEX.md)

> **Spec status lives in the index: [`openspec/INDEX.md`](openspec/INDEX.md).**
> Generated from the tree; it is the single answer to "which specs are closed
> and which are not". Do not keep a second list here.
>
> Current census (2026-10-20): **4 change packages, 0 archived, 0 capability
> specs.** All four (`aiggp-03`..`aiggp-06`) are PROPOSED — complete packages
> with no closure evidence, because `tasks.md` holds phase outlines rather than
> checked-off work.

## Identity

| Field | Value |
|---|---|
| Remote | `git@github.com:TheArchitectit/agent-guardrails-template.git` |
| Default branch | `main` |
| Language | Python 3 (stdlib-only gates) + Node scanners |
| Tests | `python3 -m pytest tests/ -q` |

## What this repo is

DevGate: the language-agnostic quality-engineering framework consumers embed as
a `.devgate/` submodule. It ships the static/CI gates (pattern, semantic,
regression, file-size, charset/encoding/corruption, manifest-join), the gated
deploy pipeline, the self-hosted runner standard, and the runner-monitor hub.

## Where things live

| Path | Purpose |
|---|---|
| `openspec/INDEX.md` | **generated** spec census — start here |
| `openspec/changes/<change>/` | change packages (proposal, design, adrs, specs, tasks, acceptance) |
| `scripts/` | the gates themselves (`check_*.py`, `guardrails-scan.mjs`) |
| `templates/` | consumer-facing copies (workflows, runner, skills) |
| `tests/` | gate self-tests; floors in `tests/expected-counts.json` |
| `docs/` | standards, runbooks, advisor personas |

## Open work

Four change packages, all PROPOSED and unarchived. None has been declared
shipped. Promoting one means checking its tasks, merging its `specs/` deltas
into a capability spec under `openspec/specs/<capability>/spec.md`, and moving
the package under `openspec/changes/archive/`.

**Structural gap worth naming:** this repo has no `openspec/specs/` capability
directory at all, so there is nothing to report per-requirement CLOSED/PARTIAL
against. Creating those capability specs is the blocker to a richer index.

## Gates

```bash
python3 -m pytest tests/ -q                    # gate self-tests
python3 scripts/check_charset.py --repo .      # fw-charset-01
python3 scripts/check_encoding.py --repo .     # fw-encoding-01
python3 scripts/check_corruption.py --repo .   # fw-corruption-01
python3 scripts/check_manifest_join.py --repo .# fw-manifest-join-01
python3 tests/gen_floors.py                    # fw-floor-01
python3 scripts/guardrails-scan.mjs            # pattern rules
```

Never weaken a gate to reach green. A `.guardrails/` overlay narrows policy; it
never disables it.
