# Proposal

## Why

The single-skill registry entry for Windsurf uses `.windsurfrules` as both its source file and destination subdirectory. The installer appends the source basename to that destination, producing a nested path where Windsurf expects one rules file.

## What Changes

- Represent file destinations distinctly from directory destinations.
- Install the Windsurf rules file directly at the target root as `.windsurfrules`.
- Verify file and directory platform targets with temporary install roots.

## Capabilities

### New Capabilities
- `platform-skill-file-destinations`: correct target resolution for platform rules that are single files.

### Modified Capabilities
- None.

## Impact

`scripts/setup_agents.py` registry and target resolution. Bulk platform installation remains supported and should retain its current behavior.
