# Design

## Context

The supported consumers include template users, CI operators, and clients
installed outside this checkout. See the [proposal](proposal.md) for the
QA and distribution findings.

## Decisions

- Run a pinned OpenSpec CLI in CI with strict validation. During inventory
  reconciliation, record the known baseline by spec slug so any additional
  failures remain visible. Remove the baseline when reconciliation passes.
- Derive CI dependency paths and toolchain versions from supported module
  inputs. Validate each advertised workflow and dependency path from a clean
  checkout, including the Go module version used by each template.
- Smoke-test documented install paths in an isolated target and invoke the
  installed components outside the source tree. Test the dependency and
  configuration defaults each consumer needs at startup. Include the relevant
  IDE API listener, package identity, and optional dependencies in the
  installation evidence.

## Rollout

Use the same verification steps on the pinned baseline and on the proposed
changes. Publish separate results for each supported CI or install path so
operators can identify the configuration covered by a successful check.
