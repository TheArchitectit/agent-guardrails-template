# Guardrails lab handoff | 27 November 2026

Resume from the published Git refs below. A clean lab clone is sufficient for source and specifications. The Windows pilot used machine-local storage. A fresh lab launch creates a separate synthetic pilot. The private operational audit and the retained IntelliJ test package travel separately with their owner.

## Pick up the branches

```sh
git clone git@github.com:TheArchitectit/agent-guardrails-template.git
git clone git@github.com:TheArchitectit/AIGGP-Agentic-Framework.git
git -C agent-guardrails-template fetch origin --tags
git -C AIGGP-Agentic-Framework fetch origin --tags
git -C agent-guardrails-template worktree add --detach ../template-pi origin/review/qa04-pi-dev-dependencies
git -C AIGGP-Agentic-Framework worktree add --detach ../aiggp-ledger origin/review/dg02-seal-evidence-batch-003
```

Use separate worktrees for the other branch names below. To make changes, create a local branch tracking the corresponding remote branch. Fetch the latest base and check the PR head again before merging. Each row describes the published checkpoint. GitHub records subsequent checks and merges.

| Order | Pull request and branch | Head | Evidence at transfer and next gate |
| --- | --- | --- | --- |
| A1 | [AIGGP #58](https://github.com/TheArchitectit/AIGGP-Agentic-Framework/pull/58) · `review/monitor-recovery-state-inventory` | `b4fd08bd5ad84915fda564273bb335505a29cc11` | Merged as `ab8ac7dd4de811f3b6c0c8cbf0df655dba49b881`. Inventory and synthetic fixture passed. The existing runner registry can contain unspent enrollment material. Start with manifest and receipt schemas, reject ineligible backups before writing bytes, then implement and rehearse controlled single-writer recovery. |
| A2 | [AIGGP #59](https://github.com/TheArchitectit/AIGGP-Agentic-Framework/pull/59) · `review/dg02-seal-evidence-batch-003` | `e281ce3af802755e2d799c0e41b10422b78aa1b8` | 518 tests passed, seven Podman cases skipped. Strict validation, traceability, index, spec-state and corruption checks passed for the proposed merge over #58. [CI](https://github.com/TheArchitectit/AIGGP-Agentic-Framework/actions/runs/46020693235) and the push gate passed. Merged on main as `af14e28cc872e6d7c98ae74f95e58468fba911e9`. Repeat skipped tests with Podman where applicable. Start the next batch at `DG02-U-037`. |
| T1 | [Template #34](https://github.com/TheArchitectit/agent-guardrails-template/pull/34) · `review/qa04-pi-dev-dependencies` | `a5085359a8623d1749017cf2ca86b5f2eaa704a7` | Pi's 200 tests in 24 files passed. Full and production-only npm audits each returned zero findings. Strict OpenSpec passed. Five PR workflows passed before transfer. Inspect all 33 jobs in [Team Validation](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/46020489075) and refresh the base. |
| T2 | [Template #35](https://github.com/TheArchitectit/agent-guardrails-template/pull/35) · `review/ide05-runtime-matrix` | `681bedeb84b8535ab39eb7c82f916a7467739210` | IntelliJ Community 2023.3.2 and its bundled JBR loaded Guardrail 1.0.0 under Xvfb. The timed run ended after the load marker. Verify the registered selection action through an editor and loopback capture with synthetic text, then qualify the declared build range. The two archive hashes and Linux-shell launch command are in the owning change. Verify both SHA-256 values before launch and bind-mount a dedicated log directory to retain `idea.log` after the disposable container exits. |
| T3 | [Template #36](https://github.com/TheArchitectit/agent-guardrails-template/pull/36) · `review/qa04-vscode-package-tools` | `19643aead55636798a8fc19990e6111cbe26a4f9` | Clean install, compilation, lint, three regressions and VSIX packaging passed. The VSIX contained the extension entry point. Full audit reported 3 moderate and 6 high findings. In a complete clone, use Node 22.20.0 and npm 10.9.3 under `ide/vscode-extension`. Run both audit modes and strict OpenSpec, then record package dispositions. Integrate after #34 and retain both QA-04 task entries. |
| T4 | [Template #37](https://github.com/TheArchitectit/agent-guardrails-template/pull/37) · `review/tpl53-process-local-preflight` | `0707208b86284ab96b6464a5c43b1c273d34107b` | The intermediate renderer and file validator passed. Final type, namespace and routing changes are pending verification. Install PyYAML in the lab test environment, run the render and `--validate` commands in the change evidence, then `python -m unittest discover -s tests -p test_render_k8s_profile.py -v`. Test malformed artifacts and safe handling of an existing output on failure. Run strict OpenSpec and the changed MCP module's gates. Keep tasks 1.2, 1.3 and 2.1 open until their respective evidence exists. |

The template baseline at transfer was `ec06a4c88fb2fb838bbbb6419cf7653cbc6c2ebe`. The full local Go replay used that base because #34 has identical Go and workflow inputs. The replay passed 29 jobs (the CLI build and 28 controls) before the mutation test returned exit 1 at Go's ten-minute test timeout. The later `test-go` job did not run in that replay. The exact workflow command from `mcp-server` is `go test ./internal/mutationkill -count=1 -v`. Reproduce on the lab runner with the pinned toolchain. If a diagnostic run needs a longer timeout, record that separately from the workflow result. Consult the job log before changing tests or declaring a pass.

## OpenSpec and remaining work

- Follow each repository's current instructions and pinned CLI. The recorded local template gate used OpenSpec 2.2.0 and AIGGP used 1.14.3. Run `openspec validate --all --strict` from the repository root with the appropriate CLI. Verify each owning change, the final merge candidate and relevant repository gates. Keep incomplete scenarios unchecked. Use separate Luna execution lanes and independent review of receipts and scope. Merge only green, reviewed changes in dependency order.
- Resume the template [verification queue](2026-11-25-template-verification-queue.md) and [gap register](2026-11-25-template-gap-register.md). Continue the optional-surface authorization, session topology and organization-scope evidence in TPL-52, TPL-53 and TPL-54. Also retain the cold-start, consuming client and external operations gates. Reconcile remaining items with each change's tasks and archive history before quoting an open total.
- Resume AIGGP's [verification queue](https://github.com/TheArchitectit/AIGGP-Agentic-Framework/blob/main/openspec/audits/2026-11-25-aiggp-verification-queue.md) and [gap register](https://github.com/TheArchitectit/AIGGP-Agentic-Framework/blob/main/openspec/audits/2026-11-25-aiggp-gap-register.md). Batch 003 covers rows 25–36. On that branch 36 of 182 have detailed reviews, 146 await detailed review and all 182 remain UNPROVEN. Preserve the original roster, recompute ordered citations and hashes from committed bytes, execute owner-named tests and verify seals before considering a promotion. Follow the existing DG-04, DG-06, DG-10 and enrollment lanes. GitLab support is in scope; live operational parity requires an adopted instance.
- TPL-35 and DG-22 are local release rehearsals. Publishing, tagging and marketplace uploads are outside the handoff. Use available GitHub checks within the free tier and local lab runners as needed.

## Windows Docker pilot

The laptop used Docker Desktop's local Linux engine. Immediately before transfer PostgreSQL, Redis and MCP reported healthy. TCP 8080 and 8081 were both published on 127.0.0.1, and the readiness endpoint on 8081 returned HTTP 200. TPL-39's original [accepted evidence](../changes/archive/2026-11-25-repair-laptop-docker-compose/evidence.md) remains the initial live pilot record.

On a Windows lab host with Docker Desktop and Linux containers, start from a clean shell in the template clone. The checked-in launcher enforces its source pin, generates host-local synthetic credentials, and checks Compose and readiness. On the original laptop it reuses its existing local setup and named volumes. Run these checks on the host that actually runs the pilot.

```powershell
docker --context desktop-linux version --format '{{.Server.Os}}'
.\scripts\start-laptop-pilot.ps1
docker --context desktop-linux ps --filter label=com.docker.compose.project=guardrails-laptop-pilot --format '{{.Names}} {{.Status}} {{.Ports}}'
docker --context desktop-linux port guardrail-mcp-server 8080/tcp
docker --context desktop-linux port guardrail-mcp-server 8081/tcp
(Invoke-WebRequest -UseBasicParsing http://127.0.0.1:8081/health/ready).StatusCode
```

Expect `linux`, three healthy services, `127.0.0.1:8080`, `127.0.0.1:8081` and `200`. The migration job can exit successfully. The laptop state uses `guardrails-laptop-pilot_pg_data` and `guardrails-laptop-pilot_redis_data`. The named volumes stay on that Docker engine. The launcher keeps its generated credential file on the Windows host under `%LOCALAPPDATA%\GuardrailsLaptopPilot`. Use a fresh synthetic fixture in the lab; transferring database contents would be a separate operation.


### Shutdown receipt

At 15:11:49 UTC on 27 November 2026, the project-scoped Compose shutdown had removed the three pilot services, the completed migration container and the pilot network. Inspection returned zero containers for the pilot project and zero local TCP listeners on 8080 or 8081. Both named volumes listed above remained present with the local driver. Twelve earlier template and AIGGP verification containers were also stopped, as was the VS Code packaging worker. Docker Desktop continues to serve other projects. The launcher and the machine-local pilot data are available for a later restart.
