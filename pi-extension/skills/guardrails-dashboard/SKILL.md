---
id: guardrails-dashboard
name: Pi Guardrails Dashboard
description: How to use and interpret the guardrails panel and status bar in the pi TUI
version: 1.0.0
tags: [safety, pi, tui]
---

# Pi Guardrails Dashboard

## Slash Command

Type `/guardrails` to open the guardrails overlay panel in the pi TUI.

## Status Bar

The guardrails extension shows a compact status in the pi status bar:

| Status | Meaning |
|--------|---------|
| `g: ok` | No issues — all laws satisfied, no violations |
| `g: !!2/3` | 2 out of 3 max strikes active |
| `g: src/` | Scope set to `src/` directory |
| `g: !3v` | 3 violations logged |
| `g: mcp:*` | MCP server connected |
| `g: mcp:.` | MCP server disconnected (standalone mode) |

## Panel Sections

The `/guardrails` overlay displays:

1. **Safety Score** — SAFE / CAUTION / AT RISK based on violation count and severity
2. **Four Laws Status** — Checkmark or X for each law based on current session state
3. **Strike Tracker** — Per-task strike counts, color-coded (green/yellow/red)
4. **Scope** — List of authorized file paths, or "Unscoped" if no restrictions
5. **Files Read** — Count of files tracked as read
6. **MCP Status** — Connected or disconnected indicator

## Closing the Panel

Press `Esc` or `q` to close the panel. Use `j`/`k` to scroll.
