# Design

## Context

On POSIX systems, a file requested with mode `0644` is created with that mode
masked by the process umask. The permission-denial tests need group/world bits
to reach their rejection paths. Production secret writers and validators use
owner-only modes and have separate security assertions.

## Goals / Non-Goals

**Goals:**

- Make intentional negative-permission fixtures deterministic across common
  host umasks.
- Keep production file mode assertions independent from fixture setup.

**Non-Goals:**

- Change production credential or secret-file permissions.
- Change process-wide umask as part of the test suite.

## Decisions

- Set the intended permissive mode explicitly after creating each negative
  fixture. This produces the test input required to exercise the denial path;
  relying on `os.WriteFile`'s requested mode is insufficient under umask
  `0077`.
- Keep the fixtures local to each test. Changing the process umask globally
  was rejected because it can affect parallel tests and unrelated temporary
  files.
- Preserve independent assertions that files written by production helpers
  remain owner-only and that permissive files are rejected.
- Run affected package tests from `mcp-server/` under both `umask 0077` and
  `umask 0022`; Windows continues to skip POSIX mode assertions where already
  documented.

## Risks / Trade-offs

- An explicit `chmod` could be omitted from one negative fixture. The scoped
  umask runs and the targeted mode assertion in each fixture expose this.
- Broader test suites may contain additional umask-sensitive fixtures outside
  the four audited cases. The focused acceptance command establishes the
  identified regression surface; later failures should be assessed separately.

## Migration Plan

There is no production migration. Update the four direct negative fixtures and
the dependent mutation baseline, run affected Go package tests under both
umasks, then run the required wider Go test matrix. Revert only fixture setup
changes if the targeted tests regress; retain existing production mode checks.
