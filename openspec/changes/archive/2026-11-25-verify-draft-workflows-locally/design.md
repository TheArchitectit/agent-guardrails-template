# Design

## Context

The workflow runs target a Linux x64 self-hosted runner. The laptop has a
working Docker Desktop Linux engine. The connected GitHub account has write
permission on the repo. Runner registration requires access beyond that role.
[GitHub runner prerequisites](https://docs.github.com/en/actions/how-tos/manage-runners/self-hosted-runners/add-runners)

## Decisions

Run the checks from the five GitHub workflow definitions in disposable Linux
workspaces. Pin the actual draft base and head, retain Git history needed by
the PR checks, and use the reviewed DevGate revision and declared toolchains.
Give each job its own writable scratch and artifact files. Exercise the
workflow's own shell steps and recorded acceptance controls where the local
runtime supports them.

Use generated synthetic input for steps that require a fixture. Verify produced
artifacts locally before any GitHub-only artifact or reporting action. Record
each unavailable control and its reason. Read hosted status separately and
preserve the published runner routing.

## Risks / Trade-offs

A generic suite can miss a workflow-specific gate. Map verification results to
workflow jobs and capture each exceptional command or option. Inspect logs for
secrets before surfacing relevant failure details. Keep live pilot volumes,
ports, and running services outside the disposable workspaces.
