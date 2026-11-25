# Design

## Context

The `windsurf-rules` registry pair is `(".windsurfrules", ".windsurfrules")`. `resolve_target` derives the source basename and joins it to the configured destination, so a file destination is treated as a directory.

## Decisions

- Make target resolution distinguish a destination directory from an explicit destination file, preferably through a small typed registry shape or an unambiguous file-target convention.
- For Windsurf single-skill installation, write directly to `<target>/.windsurfrules`.
- Keep directory-based skill destinations and bulk platform install paths unchanged.
- Use temporary roots to verify the returned path and actual file content.

## Compatibility and Rollback

The corrected path replaces the malformed nested output for this single-skill command. Do not delete or migrate pre-existing files automatically. Revert the registry/resolver change together if tests show a directory-based platform target changes.

## Verification

Run the single-skill Windsurf installer against a temporary directory and assert `.windsurfrules` is a regular file at the root; exercise representative directory-based entries and bulk install tests.
