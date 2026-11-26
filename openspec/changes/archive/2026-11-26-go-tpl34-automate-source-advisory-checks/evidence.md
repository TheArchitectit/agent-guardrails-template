# TPL-34 Go advisory gate evidence

**Local verification date:** 2026-11-25

**Go toolchain:** 1.26.11 from `mcp-server/go.mod`

**govulncheck:** v1.10.0

## Pinned end-to-end scan

From `mcp-server/`, the following command completed with `GOTOOLCHAIN=local`:

```sh
govulncheck -json ./... > /tmp/govulncheck.json
python3 scripts/govulncheck_gate.py --input /tmp/govulncheck.json --report /tmp/govulncheck-summary.json
```

The actual v1.10.0 stream declared protocol `v1.0.0`, `scan_mode: source`,
`scan_level: symbol`, and a nonempty SBOM `roots` list. The parser accepted the
complete scan, required the SBOM roots as analyzed-package evidence, and
verified each finding ID resolves to an emitted OSV record. The pinned source
emits OSV records before findings; unreferenced OSV records are valid because
raw records can apply to a module without matching its selected version.
Result:

- Source-call-path findings: **0**
- Imported-package findings: **0**
- Module-only findings: **4** — `GO-2026-5932`, `GO-2026-6860`, `GO-2026-6866`, and `GO-2026-6911`

The four module-only records remain visible and do not fail the source-call
policy. No severity or exploitability claim is made. The saved report contains
only advisory IDs, package names, and reachability; scanner traces and source
locations are omitted. The temporary raw JSON was not uploaded or retained as
an artifact.

## Fixture and specification checks

- `python3 -m pytest mcp-server/tests/test_govulncheck_gate.py -q` — **PASS**, 13 cases covering source-call-path denial, imported-only and module-only classification, clean analyzed scan, malformed/duplicate/empty output, scanner failure, missing OSV references, valid unreferenced OSV records, and whitespace-separated pretty JSON.
- `npx --yes --package @fission-ai/openspec@2.2.0 openspec validate go-tpl34-automate-source-advisory-checks --strict` — **PASS**.
- `python3 scripts/check_python_tests.py --repo .` — **PASS**, four discovered suites covered by the existing floor inventory. The repository scope intentionally excludes `mcp-server/`; the advisory workflow therefore runs this classifier suite explicitly before scanning.

These are local results. No hosted workflow execution or branch-protection setting was verified.
