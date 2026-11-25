# TPL-50 local assignment evidence

The maintainer requested local job assignment when GitHub cannot schedule a
runner. Before the latest installer update, GitHub reported five queued
workflows and 45 unassigned jobs at draft head
`547c2dec6a730d68df4e1cac3cef9da0dffdeea4`. They target Linux x64 with the
`guardrails-ucs03` label. The connected account's repository permission is
`write`. Local verification will use Docker Desktop's Linux engine.

The installer update advanced the draft to
`1ff334b04a7e5b70eb3ae1fc2519ac06d2e26e1a` on base
`943bcf65055128e338b696dc40d74e79ac33449e`. The local job matrix will be
keyed to these revisions and identify any later revision explicitly.

At 2026-11-25 19:51 UTC the updated draft had all five workflows queued,
with 45 jobs targeting `self-hosted`, `Linux`, `X64`, and
`guardrails-ucs03`. GitHub assigned none. The existing connected account
reported `write` permission. Docker Desktop still reported PostgreSQL,
Redis, and MCP healthy and published MCP ports 8080 and 8081 on
`127.0.0.1`. The readiness URL returned HTTP 200.

## Local shell jobs at the exact published draft revision

| Workflow | Local outcome | Runtime |
| --- | --- | --- |
| Regression Guard | The workflow's action pin, resource, image, secret, charset, test discovery, runner trust, mandatory row, tool count, and policy-pack steps passed. Committed-range analysis, bug-fix output, required regression test, failure registry, and final enforcement passed. | Python 3.10.22, Bash, jq 1.6, exact DevGate pin `8ccb4467c70614c37f4a533428d2aa1e09a80517` |
| Documentation Standards Check | All four workflow jobs passed their shell checks, including length and link controls. | Python 3.10.22 and Bash |
| Guardrails Compliance Check | Scope, forbidden files, commit messages, and full-message attribution passed. The dependent summary shell also ran. | Python 3.10.22 and Bash |
| Secret Validation | Verified Gitleaks, environment and credential filenames, and the hardcoded pattern checks passed. | Gitleaks 8.24.3, Python 3.10.22, and Bash |
| Team Validation | Fixture staging, team size and status, Python inventory enforcement, each enumerated test command, and floors passed. The Go job map has 29 named security rows, the enumerated module test/build, and the CLI build. The Go receipts are recorded below. | Python 3.11.18, jq 1.6, and Bash. The Go jobs declare Go through their setup and module inputs. |

The local replay staged a clean, full-history worktree per job with the
declared draft base and head. It evaluated PR-dependent conditions and used
temporary event, output, and summary files. The GitHub Actions checkout, cache, and setup actions were supplied locally.
PR event routing, Codecov upload, the Regression Guard PR comment, and CLI
artifact upload require GitHub. Local coverage XML was verified.
The Team export and upload are main-branch-only and were skipped for the PR.

## Go runner qualification

The clean Linux replay verified Go 1.26.11 and the exact LF source bytes
from Git. All five control-plane component digests matched committed bytes.
The same verifier reproduced mismatches when a Windows checkout supplied
CRLF worktree bytes. The manifest itself required no change. The focused
Go manifest test passed from the canonical archive.

The local test container includes Bash, Python exposed as `python`,
`prlimit`, jq, and curl. The isolated sandbox test container can create its
own Linux network namespace. The full guardrails package passed after
provisioning those prerequisites. The verification uses disposable
containers, with the existing pilot volumes and engine socket outside the
container's test workspace.

All 29 named Team Validation security-matrix jobs passed on the canonical
LF workspace under Go 1.26.11 using the published Bash step bodies. Independent
review matched their job IDs, 32 Bash blocks, 40 Go test calls, 46 package PASS
summaries, and all declared test selectors. The mutation job used
a local 30-minute Go test bound. This bound applies to the laptop test
environment. The enumerated Go module and CLI results follow.

## Integrated Python and shell replay

The same 13 shell jobs and both Team Python jobs passed at local candidate
`cfb14ea5d4cf3337652b316e74051be334536b1b`, including the QA-03 UTC fixes.
The targeted Go JSON parsing and manifest checks passed for the QA-03
candidate. The Team Python receipt included the existing inventory and
per-suite floors, with local coverage XML present.

## Standalone CLI build

The separate Team Validation CLI job passed under Go 1.26.11 in a clean
Linux Git checkout of the published draft. It fetched the pinned modules
through the Go module proxy and verified the module cache. The 7,606,460-byte
binary reported tag `v3.8.11-16-g1ff334b` and commit `1ff334b`.
Artifact SHA-256 was
`280334bfaf37dc170257d301826331e7cd1d054a43a942afd9d1da45dd7a728d`.
GitHub artifact upload remains a hosted integration step.

All 13 static workflow jobs passed again at candidate
`9ffd6c88dee5f40599d406a195fcd6815a6a1bf6`, including the catalog and
archive-reference changes. The only changes from the verified Python
candidate were under OpenSpec. Go module, workflow, and floor paths
match the earlier published draft.

## Full Go module replay

Go 1.26.11 found all three modules in the tracked floor registry and passed
the ten module-discovery controls. Tests, builds, and discovery completed
for each module.

| Module | Local result | Test floor |
| --- | --- | --- |
| `mcp-server` | Full fresh race test, build, and test listing passed. Mutation controls took 1,157.841 seconds. | 896 found, minimum 250 |
| `cmd/team-cli` | Fresh test and listing passed; the exact Git checkout CLI build is documented above. | 9 found, minimum 8 |
| `examples/go` | Fresh test, build, and listing passed. | 11 found, minimum 10 |

The initial all-module loop passed MCP and then stopped at uncached CLI
dependencies in the network-isolated container. The CLI and examples ran
their tracked commands after the exact public Go dependencies were fetched
and verified. Local MCP testing used a 30-minute timeout. The hosted default
timeout and GitHub action orchestration were outside this replay.

Independent review matched the workflow steps, module receipts, test floors,
matrix commands, and CLI build artifact. All 45 jobs visible in GitHub and
the dependent compliance summary passed their applicable laptop checks.
Hosted setup, event routing, and artifact or reporting operations are
documented above. The 20:51 UTC independent Windows recheck confirmed the
local Linux engine, three healthy pilot services, both IPv4 loopback ports,
HTTP 200, and both original named volumes created at 15:45:35 UTC.
