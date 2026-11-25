# Proposal

## Why

The legacy GitLab and Jenkins templates advertise Go 1.21, while the MCP module declares Go 1.26.11 and Jenkins also builds the CLI module, which declares Go 1.23.2. The templates need explicit toolchain selection for the modules their existing jobs build, so runner defaults and implicit downloads do not determine compatibility.

## What Changes

- Align the legacy MCP test jobs to the MCP module's declared Go toolchain and install/select it explicitly.
- Align the existing Jenkins CLI build with its module directive; require future jobs for other modules to use that module's own directive without adding an example job.
- Preserve existing runner stages, tests, race flags, and artifact behavior.

## Capabilities

### New Capabilities
- `legacy-ci-go-toolchain-selection`: explicit module-aligned Go toolchains for legacy CI templates.

### Modified Capabilities
- None.

## Impact

`ci/gitlab-ci.yml`, `ci/Jenkinsfile`, and Go CI setup. GitHub Actions toolchain alignment is addressed separately in the completed QA-05 work.
