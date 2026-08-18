import { describe, it, expect, beforeEach, afterEach } from "vitest";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
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

  describe("session danger allows", () => {
    it("records and checks session-scoped bash allowances", () => {
      const pm = new PermissionManager();
      expect(pm.isSessionDangerAllowed("git push --force")).toBe(false);
      pm.allowSessionDanger("git push --force");
      expect(pm.isSessionDangerAllowed("git push --force")).toBe(true);
    });

    it("normalizes commands for session allowances", () => {
      const pm = new PermissionManager();
      pm.allowSessionDanger("git   push   --force");
      expect(pm.isSessionDangerAllowed("git push --force")).toBe(true);
    });
  });

  describe("setPermission persist", () => {
    let dir: string;
    let configPath: string;

    beforeEach(() => {
      dir = fs.mkdtempSync(path.join(os.tmpdir(), "pm-config-"));
      configPath = path.join(dir, "config.json");
    });

    afterEach(() => {
      fs.rmSync(dir, { recursive: true, force: true });
    });

    it("persists tool permission changes to config.json when persist is true", () => {
      const pm = new PermissionManager({ tools: { bash: "ask" } }, configPath);
      pm.setPermission("bash", "auto", { persist: true });
      const parsed = JSON.parse(fs.readFileSync(configPath, "utf-8"));
      expect(parsed.toolPermissions.tools.bash).toBe("auto");
    });

    it("merges into an existing config.json", () => {
      fs.writeFileSync(configPath, JSON.stringify({ statusBarEnabled: false }));
      const pm = new PermissionManager({ tools: { bash: "ask" } }, configPath);
      pm.setPermission("bash", "blocked", { persist: true });
      const parsed = JSON.parse(fs.readFileSync(configPath, "utf-8"));
      expect(parsed.statusBarEnabled).toBe(false);
      expect(parsed.toolPermissions.tools.bash).toBe("blocked");
    });

    it("does not persist when persist is false", () => {
      const pm = new PermissionManager({ tools: { bash: "ask" } }, configPath);
      pm.setPermission("bash", "auto");
      expect(fs.existsSync(configPath)).toBe(false);
    });
  });
});
