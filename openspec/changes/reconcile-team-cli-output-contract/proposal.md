# Reconcile Team CLI output formats

## Why

The Team CLI advertises a persistent `--output` flag with text and JSON formats. During DIST-01 installed-prefix verification, `team status -o json` failed because the Go command forwarded `--format json` to a Python subcommand that does not accept it. Several other command wrappers forward the same argument and need a command-by-command check.

## What changes

Document the machine-readable contract for each command exposed by the Go CLI. Make supported JSON mode emit one structured result on stdout and preserve the established text output. Report unsupported formats before invoking operations with side effects.

## Capabilities

Add `team-cli-output-contract`.

## Impact

The contract covers the Go Team CLI, the Python backend argument and serialization paths, and the installed command verified under DIST-01. Commands that already produce structured output keep their existing supported result fields.
