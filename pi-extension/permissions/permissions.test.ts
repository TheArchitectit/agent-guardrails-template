import { describe, it, expect } from "vitest";
import { PermissionManager } from "./permissions.js";

describe("PermissionManager", () => {
  it("defaults to auto for unknown tools", () => {
    const pm = new PermissionManager();
    expect(pm.getPermission("unknown-tool")).toBe("auto");
  });

  it("respects configured tool permissions", () => {
    const pm = new PermissionManager({ tools: { bash: "ask", write: "blocked" } });
    expect(pm.getPermission("bash")).toBe("ask");
    expect(pm.getPermission("write")).toBe("blocked");
  });

  it("allows auto tools", () => {
    const pm = new PermissionManager();
    const result = pm.checkTool("read");
    expect(result.allowed).toBe(true);
    expect(result.needsConfirmation).toBe(false);
  });

  it("blocks blocked tools", () => {
    const pm = new PermissionManager({ tools: { bash: "blocked" } });
    const result = pm.checkTool("bash");
    expect(result.allowed).toBe(false);
    expect(result.needsConfirmation).toBe(false);
  });

  it("requires confirmation for ask tools", () => {
    const pm = new PermissionManager({ tools: { bash: "ask" } });
    const result = pm.checkTool("bash");
    expect(result.allowed).toBe(false);
    expect(result.needsConfirmation).toBe(true);
  });

  it("session overrides take priority", () => {
    const pm = new PermissionManager({ tools: { bash: "ask" } });
    pm.setPermission("bash", "auto");
    expect(pm.getPermission("bash")).toBe("auto");
    const result = pm.checkTool("bash");
    expect(result.allowed).toBe(true);
  });

  it("returns the full permission matrix", () => {
    const pm = new PermissionManager({ tools: { bash: "ask" } });
    pm.setPermission("write", "blocked");
    const matrix = pm.getPermissionMatrix();
    expect(matrix.bash).toBe("ask");
    expect(matrix.write).toBe("blocked");
  });
});
