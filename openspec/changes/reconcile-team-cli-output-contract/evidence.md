# Output contract inventory and evidence

## Command matrix (source review)

The Go persistent `-o/--output` flag advertises `text` and `json`. In `cmd/team-cli/main.go`, JSON mode forwards `--format json` for the commands below unless the wrapper performs a separate conversion. Python backend parser support is shown as of this change.

| Go command | Go behavior in JSON mode | Python backend support before this change | Effect |
| --- | --- | --- | --- |
| `init` | forwards `--format json` | rejects unknown option | creates project state |
| `list` | forwards `--format json` | rejects unknown option | read |
| `assign` | forwards `--format json` | rejects unknown option | mutates assignment |
| `unassign` | forwards `--format json` | rejects unknown option | mutates assignment |
| `start` | forwards `--format json` | rejects unknown option | mutates team status |
| `complete` | forwards `--format json` | rejects unknown option | mutates team status |
| `status` | forwards `--format json` | did not accept option; now emits JSON | read |
| `validate` | maps to `validate-size --format json` | rejects unknown option | read/validation |
| `phase-gate` | forwards `--format json` | backend has no matching command parser | read/validation |
| `query` | forwards `--format json` | supports `--format table|json`; its handler was omitted from backend dispatch and is now reachable | read; serializer exists |
| `reassign` | forwards `--format json` | rejects unknown option | mutates assignment |
| `audit` | forwards `--format json` | accepts filters but rejects unknown option | read |
| `history` | forwards `--format json` to `team-history` | supports `--format table|json`; its handler was omitted from backend dispatch and is now reachable | read; serializer exists |
| `backup` | maps to `list-backups --format json` | rejects unknown option | reads backup metadata |

`export` and `import` choose backend subcommands from their own Go format flags; they do not forward a generic `--format`. `template` chooses `template-csv` or `template-json`. `timeline` has a backend `--format table|json` option but no Go wrapper in the current command tree.

The CLI now permits global JSON only for `status`, `query`, and `history`, whose backend serializers are reachable. Other commands reject global JSON in Cobra's persistent pre-run, before their handler can invoke the backend. Unknown output strings are rejected there as well. Command-specific formats such as `export --format json` remain available through their own flags. The wider output matrix and serialization for additional commands remain open.

## Status behavior and regression checks

`status --phase NAME --format json` emits one object containing `phase`, `total_teams`, `completed`, `active`, `not_started`, and `progress_pct`. `status --format json` emits an ordered array of the same objects, sorted by phase number. Text mode retains the existing selected-phase summary and all-phase progress lines.

The source-only subset uses Go package tests and `scripts/tests/test_status_output.py` against a disposable copy of `.teams/fixtures/full-project.json`. The runnable installed-prefix entry `tests/regression/test_team_cli_output_contract.sh` covers selected/all-phase status JSON, text, query/history JSON, and byte-identical synthetic project data after unsupported JSON and unknown-format mutation requests. The broader command matrix remains open for future format support; commands outside the three supported global JSON modes now fail early.
