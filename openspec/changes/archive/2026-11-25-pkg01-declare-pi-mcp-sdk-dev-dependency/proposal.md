# Proposal

## Why

The Pi MCP bridge imports and types `@modelcontextprotocol/sdk`, but the manifest and lockfile do not declare it. Development tests therefore cannot reliably exercise SDK-present bridge imports, while the SDK is intentionally optional for standalone runtime users.

## What Changes

- Declare and lock the compatible SDK as a development dependency, not a required production dependency.
- Test SDK-present bridge loading and an isolated production install without the SDK.

## Capabilities

### New Capabilities

- `pi-mcp-sdk-development`: Keep the SDK available to development checks while preserving optional runtime loading.

### Modified Capabilities

None.

## Impact

Affects `pi-extension/package.json`, its lockfile, bridge tests, and package smoke tests. Standalone users remain able to install and run without the MCP bridge SDK. No implementation is included in this planning package.
