# Design

## Context

`activate()` registers handlers and runs the first connection check before `GuardrailClient.updateConfiguration(context)`. The client config is only assigned by that method; `fetch()` reads its URL and API key. Existing configuration-change events refresh settings, and explicit workspace URL settings must remain supported.

## Goals / Non-Goals

**Goals:**

- Initialize the client before request paths can read config.
- Preserve SecretStorage access and existing configuration-change behavior.
- Test activation with stubbed VS Code APIs and fetch, without a live MCP server.

**Non-Goals:**

- Change endpoint defaults, commands, or diagnostic behavior.
- Treat this plan as evidence that activation has been fixed.

## Decisions

- Call `updateConfiguration(context)` during activation before any connection check or command can issue a request.
- Preserve SecretStorage access by passing the extension context on initialization; retain later configuration-change updates.
- Use a mocked activation harness and assert the configured request URL, bearer key, and connected/disconnected states.

## Risks / Trade-offs

- [Configuration loads after a command starts] → register and initialize in a defined sequence and assert the first fetch in the activation test.
- [A refactor drops user configuration] → cover a non-default URL and SecretStorage key in the same test.

## Migration Plan

No settings or API migration is required. Run the focused activation test and the VS Code compile/lint checks in `tasks.md`.
