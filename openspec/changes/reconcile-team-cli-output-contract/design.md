# Reconcile Team CLI output formats

## Source and observed behavior

The DIST-01 Linux check ran with Go 1.23.2 and Python 3.11.2. The Go `status` command in `cmd/team-cli/main.go` forwards `--format json` when the persistent output flag selects JSON. The Python `status` parser in `scripts/team_manager.py` accepts `--phase` and currently prints progress text. The installed text-mode `list` check succeeded in the same verification.

## Decisions

Inventory explicit and inherited output flags and distinguish actual serialization from flags forwarded to the backend. Define the result and supported modes per command. Introduce JSON at the layer that owns the result fields, and keep progress or error diagnostics separate from the JSON document. Validate format support before a mutating command starts. Retain compatible output for existing JSON callers.

## Verification

Exercise the Go entry point and Python backend with temporary team data. Verify the status result for a selected phase and for all phases. Cover existing text output, an already-supported JSON command, unsupported output, and the absence of a state change after an unsupported-format error. Repeat the exposed entry-point checks from a temporary DIST-01 installed prefix.
