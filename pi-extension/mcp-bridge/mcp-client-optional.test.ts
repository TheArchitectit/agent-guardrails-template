import { describe, expect, it } from "vitest";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";
import ts from "typescript";

describe("optional MCP SDK", () => {
  it("loads the bridge in isolation and reports the SDK unavailable", async () => {
    const tempDir = mkdtempSync(join(tmpdir(), "pi-guardrails-sdk-absent-"));
    try {
      const source = readFileSync(new URL("./mcp-client.ts", import.meta.url), "utf8");
      const { outputText } = ts.transpileModule(source, {
        compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
      });
      const modulePath = join(tempDir, "mcp-client.mjs");
      writeFileSync(modulePath, outputText);

      const { MCPClient, getMcpSdkLoadError } = await import(
        `${pathToFileURL(modulePath).href}?isolated=${Date.now()}`
      );
      const client = new MCPClient();

      await expect(client.tryConnect("http://127.0.0.1:1")).resolves.toBe(false);
      expect(client.isConnected()).toBe(false);
      expect(client.getTools()).toEqual([]);
      expect(getMcpSdkLoadError()).toBeNull();
    } finally {
      rmSync(tempDir, { recursive: true, force: true });
    }
  });
});
