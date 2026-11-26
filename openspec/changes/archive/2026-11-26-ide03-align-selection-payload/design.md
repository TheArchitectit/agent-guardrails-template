# Design

## Context

`validateSelection` currently decodes `selection` and rejects an empty value, while VS Code, JetBrains, Vim, and Neovim send selected text as `code`. A client-only rename would break already-installed extensions. Selecting silently between two different supplied values would make the server validate ambiguous text.

## Goals / Non-Goals

**Goals:**

- Make `selection` the canonical field for new clients.
- Keep existing `code` clients working during migration.
- Reject conflicting dual-field payloads and preserve source text exactly.

**Non-Goals:**

- Remove support for installed clients that send only `code`.
- Normalize or transform selected code in the request handler.

## Decisions

- Accept legacy `code` only when `selection` is absent; if both are present, accept equal values and reject different values with HTTP 400.
- Update all four maintained IDE request builders to send `selection` while retaining handler compatibility for older clients.
- Test the real Go handler for canonical, legacy, matching dual, conflicting dual, and empty/missing cases; verify each IDE builder uses an accepted payload shape.

## Risks / Trade-offs

- [Older clients stop working during migration] → keep the `code`-only compatibility path until a separately reviewed removal.
- [Ambiguous requests validate the wrong snippet] → reject conflicts instead of picking either field.
- [Client contract test becomes vacuous] → assert all four named client request builders are exercised.

## Migration Plan

No immediate client upgrade is required. New releases send `selection`; older releases continue to work through `code`-only compatibility. Any future removal of `code` requires its own migration decision and evidence.
