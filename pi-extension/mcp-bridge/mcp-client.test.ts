import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { resolveMcpEndpoint, MCP_STREAM_PATH, getMcpSdkLoadError } from "./mcp-client.js";

const here = dirname(fileURLToPath(import.meta.url));

describe("resolveMcpEndpoint", () => {
  it("targets the server's Streamable HTTP route for a bare base URL", () => {
    expect(resolveMcpEndpoint("http://127.0.0.1:8095").pathname).toBe(MCP_STREAM_PATH);
    expect(resolveMcpEndpoint("http://127.0.0.1:8095/").pathname).toBe(MCP_STREAM_PATH);
  });

  it("targets the Streamable HTTP route for a legacy SSE path", () => {
    // Spec 24: the server serves no /mcp/v1/sse; that path must not survive.
    const resolved = resolveMcpEndpoint("http://127.0.0.1:8095/mcp/v1/sse");
    expect(resolved.pathname).toBe(MCP_STREAM_PATH);
    expect(resolved.pathname).not.toContain("/v1/");
    expect(resolveMcpEndpoint("http://127.0.0.1:8095/sse").pathname).toBe(MCP_STREAM_PATH);
  });

  it("preserves an already-correct explicit route", () => {
    expect(resolveMcpEndpoint("http://127.0.0.1:8095/mcp/stream").pathname).toBe(MCP_STREAM_PATH);
  });

  it("preserves the host and port", () => {
    const resolved = resolveMcpEndpoint("https://guardrails.internal:8443/mcp/v1/sse");
    expect(resolved.origin).toBe("https://guardrails.internal:8443");
  });
});

describe("mcp-client transport contract (spec 24)", () => {
  const source = readFileSync(join(here, "mcp-client.ts"), "utf8");
  // Strip comments so a doc reference to the old path is not mistaken for code.
  const code = source
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .split("\n")
    .map((line) => line.replace(/\/\/.*$/, ""))
    .join("\n");

  it("uses the SDK Streamable HTTP transport, not SSE", () => {
    expect(code).toContain("StreamableHTTPClientTransport");
    expect(code).not.toContain("SSEClientTransport");
  });

  it("never targets the nonexistent /mcp/v1/sse prefix in code", () => {
    expect(code).not.toContain("/mcp/v1/sse");
  });
});
describe("mcp-client SDK entrypoints (spec 27)", () => {
  const source = readFileSync(join(here, "mcp-client.ts"), "utf8");
  const code = source
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .split("\n")
    .map((line) => line.replace(/\/\/.*$/, ""))
    .join("\n");

  it("never imports the package root the published SDK does not expose", () => {
    // Spec 27: `import("@modelcontextprotocol/sdk")` resolves to a
    // dist/esm/index.js that no published version ships.
    expect(code).not.toMatch(/import\(\s*["']@modelcontextprotocol\/sdk["']\s*\)/);
  });

  it("imports the SDK's real subpath entrypoints", () => {
    expect(code).toContain("@modelcontextprotocol/sdk/client/index.js");
    expect(code).toContain("@modelcontextprotocol/sdk/client/streamableHttp.js");
    expect(code).toContain("@modelcontextprotocol/sdk/client/stdio.js");
  });

  it("loads the SDK on this install (no unexpected resolution error)", () => {
    // R27.3: with the SDK installed, resolution must succeed and the bridge
    // must not record a load error. A non-null value means an installed
    // package's subpath failed to resolve (the spec 27 defect class).
    expect(getMcpSdkLoadError()).toBeNull();
  });

  it("resolves the transport to the server's Streamable HTTP /mcp/stream route", () => {
    // The HTTP path used by connectHttp() for a bare URL endpoint.
    expect(resolveMcpEndpoint("http://127.0.0.1:8081").pathname).toBe("/mcp/stream");
  });
});
