# DIST-01 Team CLI backend distribution evidence

**Source:** detached GitHub `main` at `7585c2cf4b2bfa85e20829b36188bd534c2d3326`

**Local verification:** Go 1.23.2, Python 3.11.2, Linux/amd64 container

## Install behavior

`bash cmd/team-cli/test-install.sh` passed with a disposable prefix and an
unrelated temporary working directory. The harness verified:

- `make install PREFIX=...` installs `team`, the `team-cli` symlink, and the
  backend modules plus migrations under
  `share/agent-guardrails-template/team-cli/scripts`.
- The installed Makefile binary invokes `team_manager.py` and lists the 12
  teams from a copied fixture outside the checkout.
- A separate user `TEAM_MANAGER_PATH` script runs before the packaged default;
  its marker and file remain after uninstall.
- `install.sh` runs while the caller is in the unrelated directory, installs
  the same byte-identical Python payload, and its installed binary runs the
  same backend command.
- Makefile uninstall removes the package binaries, alias, and backend payload
  from each prefix while preserving a sibling user-data file.
- The Makefile rejects an empty `PREFIX` before an uninstall can target a
  root-level default path.

The Go package tests passed with:

```sh
GOTOOLCHAIN=local go test ./... -count=1
```

Result: `ok github.com/thearchitectit/guardrail-mcp/cmd/team-cli`.

## Scope note

During setup, `team status -o json` failed because the Go CLI passes
`--format json`, which the Python status parser rejects. The packaging test
uses the working text `team list` path to prove the installed backend runs;
this separate output-format behavior was not changed by DIST-01.

## Independent verification

Against template main 7c52f2091b9b0fa508500daa1d93acd3f091a396, the staged
implementation passed the complete Go 1.23.2 module tests and disposable
install harness on Linux with Python 3.11.2. The harness exercised relative
prefix installation, named migration files, blank prefixes, missing backend
files, a substituted symlink path, and preservation of user files inside the
migration directory. Both installed commands invoked the real packaged Python
backend from a separate working directory.

The [DIST-02 output contract](../../reconcile-team-cli-output-contract/proposal.md)
tracks the observed JSON status error and verifies advertised formats.
