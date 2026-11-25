# Tasks

- [ ] [QA-01] Make strict OpenSpec validation cover all 15 spec files and
  proposed changes; resolve or explicitly classify legacy scenario-format
  failures and retain the results.
- [ ] [QA-02] Make file-permission-sensitive auth/config tests independent of
  the caller's default umask, and preserve private fixture permissions.
- [ ] [QA-03] Replace deprecated naive UTC timestamp calls in the Python suite
  with timezone-aware UTC serialization and retain JSON compatibility.
- [ ] [QA-04] Review npm advisory results against the locked development
  dependencies and add an automated advisory check with date-stamped results.
- [ ] [CI-01] Correct GitLab dependency installation inputs so configured jobs
  reference a dependency manifest present at the declared path.
- [ ] [CI-02] Align GitLab and Jenkins Go toolchain configuration with the MCP
  module requirement, and verify behavior without relying on implicit
  toolchain download.
- [ ] [DIST-01] Make the team CLI installation provide or locate its Python
  backend outside a repository checkout, with documented and verified path
  behavior.
- [ ] [IDE-01] Initialize VS Code extension configuration before its first
  connection check and verify activation against the configured server.
- [ ] [IDE-02] Align IDE client defaults and setup documentation with the
  actual REST server port, keeping it distinct from the MCP protocol port.
- [ ] [IDE-03] Reconcile the IDE selection-validation request field across the
  REST handler and VS Code, Neovim, and JetBrains clients.
- [ ] [INSTALL-01] Preserve leading dots when resolving documented hidden
  configuration destinations in `setup_agents.py`.
- [ ] [INSTALL-02] Correct the Windsurf rules target in the single-skill
  installer and verify the resulting file path.
- [ ] [PKG-01] Declare or otherwise supply the MCP SDK required by the Pi
  extension bridge test and make clean-install test results reproducible.
- [ ] [PKG-02] Align the published Pi extension package name with README and
  CLI installation commands.
- [ ] [LEGACY-01] Reconcile dated design and implementation status statements
  against current code and named tests; retain explicit dates and residuals.
