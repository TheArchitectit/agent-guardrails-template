#!/usr/bin/env bash
# Regression test for FAILURE-ID: FAIL-52a11c0f
# Bug: optional Vision MCP and REST operations bypassed shared authorization
# and inherited a deployment example that enabled the optional surface.
# Fix: route calls through policy/audit checks and default the profile off.
# The existing Go tests own the synthetic fixtures; this script only runs them.

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/../.." && pwd)"
cd "$repo_root/mcp-server"

go test ./internal/auth ./internal/web ./internal/mcp \
  -run '^(TestClassifyOptionalVisionRESTSurface|TestClassifyVisionMCPReportResource|TestAPIKeyAuth_OptionalVision.*|TestOptionalVisionRegisteredRoutesAuthorizeEffectsAndRedactAudit|TestOptionalVisionDefaultProfileDoesNotMountRoutes|TestOptionalVisionToolsUseCommonAuthorization)$' \
  -count=1