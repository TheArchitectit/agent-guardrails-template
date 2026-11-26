# JavaScript dependency advisory audit — 2026-11-25 (America/Chicago)

This registry-time snapshot is from template `main` at `7c52f2091b9b0fa508500daa1d93acd3f091a396`. Scan time: `2026-11-26T04:07:06.691Z` (November 25, Central). The procedure extracted only each committed `package.json` and `package-lock.json` blob into fresh temporary directories, ran `npm ci --ignore-scripts --no-audit --no-fund`, then captured full and `--omit=dev` audits. Lifecycle scripts were disabled. The temporary npm cache and empty user/global npm configs were isolated; npm used only `https://registry.npmjs.org/`.

The audit ran as an unprivileged user in local image `qa04-node22-git` (`sha256:d4ff93e50d23720b6d53118553b51a2da89e9bf2ffc97d3551b400ca34f19cbe`), built from Node base `node@sha256:b21fe589dfbe5cc39365d0544b9be3f1f33f55f3c86c87a76ff65a02f8f5848e` and Git `2.39.5`. Runtime versions: Node `v22.20.0`, npm `10.9.3`. No manifests or lockfiles were changed.

| Package tree | Raw Git lockfile SHA-256 | Full audit findings | `--omit=dev` findings | npm ci / full / omit exit |
|---|---|---:|---:|---|
| `pi-extension` | `42100c30908fb0a36993ae5400876a80301e94633a8b3efe2253f1b27480dace` | 10 (3 moderate, 5 high, 2 critical) | 0 | 0 / 1 / 0 |
| `ide/vscode-extension` | `edfa88305337b827cf5f19246ebea47f4de24c5593d4e45ee5127bfc836f371c` | 14 (4 moderate, 10 high) | 0 | 0 / 1 / 0 |

The full audit reports findings in the complete locked dependency graph. The production-only audit omits development dependencies and returned no advisory entries for these lockfiles. This does not establish runtime reachability, exploitability, or the absence of all production risk. The zero result is limited to `npm audit --omit=dev` at this date and lock identity. The raw npm JSON and metadata are adjacent. SHA-256 values above were computed from raw Git blob bytes.

The guarded procedure's negative controls used the same Node runtime and Git-pinned source in a disposable container. A simulated `npm ci` exit 7 and a malformed audit JSON response with exit 1 each returned nonzero, created no output directory, and emitted no captured diagnostic sentinel. These controls verify failure handling; they are not package audit results. Raw report files contain public npm audit data, with public advisory URLs, and no checkout paths or credential configuration.

## Package-entry review queue

The rows below preserve npm's reported package, severity, and directness. Every entry is currently **pending maintainer disposition**. Proposed routing is to the relevant extension maintainer to review the advisory path and any package update in a separate change; this is a review suggestion, not approval, acceptance, or a runtime reachability finding. A `fixAvailable` value in the raw npm report is an upstream npm suggestion, not an approved fix.

| Tree | Reported package | Severity | Direct? | Proposed owner role | Disposition |
|---|---|---|---|---|---|
| Pi | `@vitest/mocker` | moderate | no | Pi extension maintainer | Pending review |
| Pi | `esbuild` | moderate | no | Pi extension maintainer | Pending review |
| Pi | `nanoid` | high | no | Pi extension maintainer | Pending review |
| Pi | `postcss` | high | no | Pi extension maintainer | Pending review |
| Pi | `source-map-js` | high | no | Pi extension maintainer | Pending review |
| Pi | `tinypool` | critical | no | Pi extension maintainer | Pending review |
| Pi | `vite` | high | no | Pi extension maintainer | Pending review |
| Pi | `vite-node` | moderate | no | Pi extension maintainer | Pending review |
| Pi | `vitest` | critical | yes | Pi extension maintainer | Pending review |
| Pi | `zod` | high | no | Pi extension maintainer | Pending review |
| VS Code | `ajv` | moderate | no | VS Code extension maintainer | Pending review |
| VS Code | `brace-expansion` | high | no | VS Code extension maintainer | Pending review |
| VS Code | `flatted` | high | no | VS Code extension maintainer | Pending review |
| VS Code | `js-yaml` | high | no | VS Code extension maintainer | Pending review |
| VS Code | `linkify-it` | high | no | VS Code extension maintainer | Pending review |
| VS Code | `markdown-it` | high | no | VS Code extension maintainer | Pending review |
| VS Code | `minimatch` | high | no | VS Code extension maintainer | Pending review |
| VS Code | `picomatch` | high | no | VS Code extension maintainer | Pending review |
| VS Code | `qs` | moderate | no | VS Code extension maintainer | Pending review |
| VS Code | `tmp` | high | no | VS Code extension maintainer | Pending review |
| VS Code | `underscore` | high | no | VS Code extension maintainer | Pending review |
| VS Code | `undici` | high | no | VS Code extension maintainer | Pending review |
| VS Code | `vsce` | moderate | yes | VS Code extension maintainer | Pending review |
| VS Code | `xml2js` | moderate | no | VS Code extension maintainer | Pending review |

No severity threshold or automatic release block is proposed here. Maintainers should replace each Pending value with a dated decision before citing this record as release evidence; a later scan must retain its own date and lockfile identity.
