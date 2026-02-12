# Guardrail IDE Extensions

> Native IDE integrations for the Guardrail MCP Server

## Overview

This directory contains IDE extensions that provide real-time guardrail validation directly within your development environment.

## Status

**Branch:** `ide`  
**Phase:** Planning & Scaffolding  
**Target Release:** v1.13.0

## Supported IDEs

| IDE | Status | Lead | Priority |
|-----|--------|------|----------|
| VS Code | 🚧 Planning | TBD | P0 |
| JetBrains | 📋 Planned | TBD | P1 |
| Neovim | 📋 Planned | TBD | P2 |

Legend:
- ✅ Released
- 🚧 In Development  
- 📋 Planned
- ⏸️ On Hold

## Quick Start

### VS Code (Coming Soon)

```bash
# Install from VS Code Marketplace
ext install TheArchitectit.guardrail
```

### JetBrains (Coming Soon)

Install from JetBrains Marketplace.

### Neovim (Coming Soon)

```lua
-- Using lazy.nvim
{ 'TheArchitectit/guardrail.nvim' }
```

## Directory Structure

```
ide/
├── IDE_EXTENSIONS_PLAN.md    # Master plan document
├── TEAM_STRUCTURE.md          # Team organization
├── README.md                  # This file
├── vscode-extension/          # VS Code extension
│   ├── package.json
│   ├── tsconfig.json
│   └── src/
│       ├── extension.ts
│       ├── commands/
│       ├── providers/
│       └── utils/
├── jetbrains-plugin/          # IntelliJ/PyCharm plugin
│   └── build.gradle.kts
├── neovim-plugin/             # Neovim Lua plugin
│   └── lua/
└── shared/                    # Shared components
    ├── api-client/
    ├── icons/
    └── schemas/
```

## Features

All IDE extensions provide:

- ✅ Real-time validation (on save and on type)
- ✅ Inline diagnostics with severity levels
- ✅ Status bar connection indicator
- ✅ Command palette integration
- ✅ Quick fixes for common violations
- ✅ Configuration UI
- ✅ Output channel for logs

## Architecture

```
IDE Extensions
├── VS Code (TypeScript)
│   └── VS Code API
├── JetBrains (Kotlin)
│   └── IntelliJ Platform SDK
├── Neovim (Lua)
│   └── Neovim API
└── Shared
    └── HTTP Client → MCP Server (Port 8095)
```

## Development

### Prerequisites

- Node.js 16+ (VS Code)
- JDK 17+ (JetBrains)
- Neovim 0.9+ (Neovim)

### Setup

```bash
# VS Code Extension
cd ide/vscode-extension
npm install
npm run compile
```

## Contributing

See [TEAM_STRUCTURE.md](./TEAM_STRUCTURE.md) for team organization and [IDE_EXTENSIONS_PLAN.md](./IDE_EXTENSIONS_PLAN.md) for roadmap.

## Resources

- **Plan:** [IDE_EXTENSIONS_PLAN.md](./IDE_EXTENSIONS_PLAN.md)
- **Team:** [TEAM_STRUCTURE.md](./TEAM_STRUCTURE.md)
- **MCP Server:** `/mcp-server/`

## License

BSD-3-Clause
