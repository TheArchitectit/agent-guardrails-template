# Template OpenSpec verification queue

Work sequence for November 25–26, 2026. The [gap register](2026-11-25-template-gap-register.md)
identifies the owning change and its acceptance tasks.

## Current batch

- [ ] PKG-02. Verify the packed name, new test exclusion, clean locked package
  suite, and temporary-home installer help. Archive after strict validation.
- [ ] IDE-03. Run the four IDE request builders against the selection contract,
  verify the real server route and legacy compatibility, and finish the locked
  VS Code install, compile, and test run.
- [ ] TPL-35. Rehearse the checked-in release coverage gate in disposable Git
  repositories. Record matching, missing, lightweight, and stale tag outcomes
  and the changelog control. Release metadata preparation remains pending.

## Architecture acceptance

- [ ] TPL-49. Authorize every enabled optional MCP and REST path.
- [ ] TPL-50. Qualify supported MCP session topologies.
- [ ] TPL-51. Qualify shared document access across organization scopes.

## Following batch

- [ ] LEGACY-01. Reconcile the five dated design statements against source
  and named tests, retaining open acceptance criteria.
- [ ] TPL-34. Add the recurring source advisory check and exercise the specified
  reachability and tool-failure controls.
- [ ] DIST-01. Install the Team CLI and its Python backend into temporary
  prefixes, then verify external-directory invocation and uninstall ownership.
- [ ] QA-04. Re-run the Pi and VS Code lockfile-backed audits and record the
  separate production and development results for review.

Run the task-specific local checks and the pinned strict OpenSpec validator.
Record exact commit and test evidence with the owning change. Use available
GitHub checks and document local replay for unavailable runner jobs. Merge each
verified PR against its checked base. The existing Windows pilot remains active
during isolated checks.
