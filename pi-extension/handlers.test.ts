import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { createBashPermissionHandler, type HandlerDeps } from "./handlers.js";
import { SessionStore } from "./standalone/session-store.js";
import { FileReadStore } from "./standalone/file-read-store.js";
import { StrikeCounter } from "./standalone/strike-counter.js";
import { ScopeValidator } from "./standalone/scope-validator.js";
import { HaltChecker } from "./standalone/halt-checker.js";
import { ViolationLog } from "./standalone/violation-log.js";
import { PermissionManager } from "./permissions/permissions.js";
import { DangerAllowList } from "./permissions/danger-allow-list.js";
import { DEFAULT_CONFIG, type GuardrailsConfig } from "./types.js";

function makeDeps(dir: string, overrides?: Partial<GuardrailsConfig>): HandlerDeps {
  const config: GuardrailsConfig = { ...DEFAULT_CONFIG, ...overrides };
  return {
    sessionStore: new SessionStore(config.maxStrikes),
    fileReadStore: new FileReadStore(),
    scopeValidator: new ScopeValidator(),
    strikeCounter: new StrikeCounter(config.maxStrikes),
    haltChecker: new HaltChecker(),
    violationLog: new ViolationLog(path.join(dir, "violations.jsonl")),
    mcpClient: { isConnected: () => false, close: async () => {} } as unknown as HandlerDeps["mcpClient"],
    config,
    permissionManager: new PermissionManager(config.toolPermissions, path.join(dir, "config.json")),
    dangerAllowList: new DangerAllowList(path.join(dir, "allowlist.json")),
  };
}

function makeCtx(overrides?: { select?: (title: string, options: string[]) => Promise<string | undefined>; input?: (title: string, placeholder?: string) => Promise<string | undefined>; hasUI?: boolean }) {
  return {
    hasUI: overrides?.hasUI ?? true,
    ui: {
      select: vi.fn(overrides?.select ?? (async () => "Allow once")),
      input: vi.fn(overrides?.input ?? (async (_t: string, p?: string) => p)),
      setStatus: vi.fn(),
    },
  };
}

const bashEvent = (command: string) => ({ toolName: "bash", input: { command } });

function readViolations(dir: string): Array<{ severity?: string; details?: string; operation?: string }> {
  const file = path.join(dir, "violations.jsonl");
  if (!fs.existsSync(file)) return [];
  return fs.readFileSync(file, "utf-8").split("\n").filter(Boolean).map((l) => JSON.parse(l));
}

describe("createBashPermissionHandler", () => {
  let dir: string;
  beforeEach(() => {
    dir = fs.mkdtempSync(path.join(os.tmpdir(), "handlers-"));
  });
  afterEach(() => {
    fs.rmSync(dir, { recursive: true, force: true });
  });

  it("ignores non-bash tools and malformed input", async () => {
    const deps = makeDeps(dir);
    const handler = createBashPermissionHandler(deps);
    expect(await handler({ toolName: "read", input: {} } as any, makeCtx() as any)).toBeUndefined();
    expect(await handler({ toolName: "bash", input: {} } as any, makeCtx() as any)).toBeUndefined();
  });

  it("allows allow-listed commands without prompting", async () => {
    const deps = makeDeps(dir);
    deps.dangerAllowList.addExact("git push --force origin main", "tool", "workflow");
    const ctx = makeCtx();
    const handler = createBashPermissionHandler(deps);
    expect(await handler(bashEvent("git push --force origin main") as any, ctx as any)).toBeUndefined();
    expect(ctx.ui.select).not.toHaveBeenCalled();
    deps.violationLog.flush();
    const entries = readViolations(dir);
    expect(entries.some((e) => e.severity === "info" && e.details?.includes("git push --force origin main") && e.details?.includes("allow-list"))).toBe(true);
  });

  it("blocks dangerous commands at level blocked without prompting", async () => {
    const deps = makeDeps(dir, { toolPermissions: { tools: { bash: "blocked" } } });
    const ctx = makeCtx();
    const handler = createBashPermissionHandler(deps);
    const result = await handler(bashEvent("git push --force origin main") as any, ctx as any);
    expect(result?.block).toBe(true);
    expect(ctx.ui.select).not.toHaveBeenCalled();
    deps.violationLog.flush();
    const entries = readViolations(dir);
    expect(entries.some((e) => e.severity === "warning" && e.details?.includes("git push --force origin main"))).toBe(true);
  });

  it("treats a partial allowDanger config as enabled", async () => {
    const deps = makeDeps(dir, {
      allowDanger: { requireTypebackForCatastrophic: false } as any,
      toolPermissions: { tools: { bash: "ask" } },
    });
    const ctx = makeCtx();
    const handler = createBashPermissionHandler(deps);
    expect(await handler(bashEvent("ls") as any, ctx as any)).toBeUndefined();
    expect(ctx.ui.select).toHaveBeenCalledTimes(1);
  });

  it("allows session-danger-allowed commands without prompting", async () => {
    const deps = makeDeps(dir);
    deps.permissionManager.allowSessionDanger("sudo apt update");
    const ctx = makeCtx();
    const handler = createBashPermissionHandler(deps);
    expect(await handler(bashEvent("sudo apt update") as any, ctx as any)).toBeUndefined();
    expect(ctx.ui.select).not.toHaveBeenCalled();
  });

  it("auto-allows safe commands at level auto", async () => {
    const deps = makeDeps(dir, { toolPermissions: { tools: { bash: "auto" } } });
    const ctx = makeCtx();
    const handler = createBashPermissionHandler(deps);
    expect(await handler(bashEvent("ls -la") as any, ctx as any)).toBeUndefined();
    expect(ctx.ui.select).not.toHaveBeenCalled();
  });

  it("blocks safe commands at level blocked", async () => {
    const deps = makeDeps(dir, { toolPermissions: { tools: { bash: "blocked" } } });
    const handler = createBashPermissionHandler(deps);
    const result = await handler(bashEvent("ls -la") as any, makeCtx() as any);
    expect(result?.block).toBe(true);
  });

  it("prompts for safe commands at level ask and honors scope choices", async () => {
    const deps = makeDeps(dir, { toolPermissions: { tools: { bash: "ask" } } });

    // Allow once: no state recorded
    let ctx = makeCtx({ select: async () => "Allow once" });
    let handler = createBashPermissionHandler(deps);
    expect(await handler(bashEvent("npm test") as any, ctx as any)).toBeUndefined();
    expect(deps.permissionManager.isSessionDangerAllowed("npm test")).toBe(false);

    // Allow for session: recorded
    ctx = makeCtx({ select: async () => "Allow for session" });
    handler = createBashPermissionHandler(deps);
    expect(await handler(bashEvent("npm run build") as any, ctx as any)).toBeUndefined();
    expect(deps.permissionManager.isSessionDangerAllowed("npm run build")).toBe(true);

    // Always allow: persisted to allow-list
    ctx = makeCtx({ select: async () => "Always allow" });
    handler = createBashPermissionHandler(deps);
    expect(await handler(bashEvent("npm run lint") as any, ctx as any)).toBeUndefined();
    expect(deps.dangerAllowList.matches("npm run lint")?.type).toBe("exact");

    // Deny: blocked
    ctx = makeCtx({ select: async () => "Deny" });
    handler = createBashPermissionHandler(deps);
    const denied = await handler(bashEvent("npm whoami") as any, ctx as any);
    expect(denied?.block).toBe(true);
    deps.violationLog.flush();
    const deniedEntries = readViolations(dir);
    expect(deniedEntries.some((e) => e.severity === "warning" && e.details?.includes("npm whoami"))).toBe(true);

    // Dismissed (undefined): blocked
    ctx = makeCtx({ select: async () => undefined });
    handler = createBashPermissionHandler(deps);
    const dismissed = await handler(bashEvent("npm ping") as any, ctx as any);
    expect(dismissed?.block).toBe(true);
  });

  it("prompts for dangerous commands with the category in the title", async () => {
    const deps = makeDeps(dir, { toolPermissions: { tools: { bash: "auto" } } });
    const ctx = makeCtx({ select: async () => "Allow once" });
    const handler = createBashPermissionHandler(deps);
    expect(await handler(bashEvent("git push --force origin feature") as any, ctx as any)).toBeUndefined();
    expect(ctx.ui.select).toHaveBeenCalledTimes(1);
    const title = (ctx.ui.select as any).mock.calls[0][0] as string;
    expect(title).toContain("destructive");
    expect(title).toContain("git push --force origin feature");
  });

  it("requires type-back for catastrophic commands", async () => {
    const deps = makeDeps(dir, { toolPermissions: { tools: { bash: "auto" } } });
    const cmd = "rm -rf /";

    // Typed text mismatch -> blocked, no scope prompt
    let ctx = makeCtx({ input: async () => "something else", select: async () => "Allow once" });
    let handler = createBashPermissionHandler(deps);
    const mismatch = await handler(bashEvent(cmd) as any, ctx as any);
    expect(mismatch?.block).toBe(true);
    expect(ctx.ui.select).not.toHaveBeenCalled();

    // Typed text matches -> proceeds to scope prompt
    ctx = makeCtx({ input: async () => cmd, select: async () => "Allow once" });
    handler = createBashPermissionHandler(deps);
    expect(await handler(bashEvent(cmd) as any, ctx as any)).toBeUndefined();
    expect(ctx.ui.select).toHaveBeenCalledTimes(1);
  });

  it("skips type-back when requireTypebackForCatastrophic is false", async () => {
    const deps = makeDeps(dir, {
      allowDanger: { enabled: true, requireTypebackForCatastrophic: false },
      toolPermissions: { tools: { bash: "auto" } },
    });
    const ctx = makeCtx({ select: async () => "Allow once" });
    const handler = createBashPermissionHandler(deps);
    expect(await handler(bashEvent("rm -rf /") as any, ctx as any)).toBeUndefined();
    expect(ctx.ui.input).not.toHaveBeenCalled();
  });

  it("denies when no UI is available", async () => {
    const deps = makeDeps(dir, { toolPermissions: { tools: { bash: "ask" } } });
    const ctx = makeCtx({ hasUI: false });
    const handler = createBashPermissionHandler(deps);
    const result = await handler(bashEvent("npm test") as any, ctx as any);
    expect(result?.block).toBe(true);
    expect(result?.reason).toContain("no UI");
  });

  it("reverts to legacy hard-block behavior when allowDanger is disabled", async () => {
    const deps = makeDeps(dir, {
      allowDanger: { enabled: false, requireTypebackForCatastrophic: true },
      toolPermissions: { tools: { bash: "ask" } },
    });
    const ctx = makeCtx({ select: async () => "Always allow" });
    const handler = createBashPermissionHandler(deps);

    // Dangerous command blocked without any prompt
    const dangerous = await handler(bashEvent("git push --force origin main") as any, ctx as any);
    expect(dangerous?.block).toBe(true);
    expect(ctx.ui.select).not.toHaveBeenCalled();

    // Safe command blocked by ask level with the legacy chat-instruction reason
    const safe = await handler(bashEvent("ls") as any, ctx as any);
    expect(safe?.block).toBe(true);
    expect(safe?.reason).toContain("confirmation");
  });
});
