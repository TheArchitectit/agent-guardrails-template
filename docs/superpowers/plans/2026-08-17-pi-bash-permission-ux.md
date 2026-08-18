# Pi Extension Bash Permission UX Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace hard-blocked bash enforcement with interactive permission prompts (once / session / always) backed by a persisted danger allow-list managed via a `guardrail_allow_danger` tool.

**Architecture:** A single `createBashPermissionHandler` owns all bash `tool_call` decisions: allow-list short-circuit → classify → prompt (simple, strong, or catastrophic type-back) → record approval at the chosen scope. Non-bash tools keep the existing permission handler. Spec: `docs/superpowers/specs/2026-08-17-pi-bash-permission-ux-design.md`.

**Tech Stack:** TypeScript (ESM, `.js` import suffixes), Vitest, `@earendil-works/pi-coding-agent` extension API (`ctx.ui.select` / `ctx.ui.input`), typebox for tool param schemas.

**Working directory for all commands:** `/mnt/data/git/agent-guardrails-template/pi-extension` (run tests via `npx vitest run <file>`).

---

### Task 1: Type extensions + config plumbing

**Files:**
- Modify: `pi-extension/types.ts`
- Modify: `pi-extension/config.ts`
- Create: `pi-extension/config.test.ts`

- [ ] **Step 1: Write the failing test**

Create `pi-extension/config.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { getAllowListPath } from "./config.js";
import { DEFAULT_CONFIG } from "./types.js";

describe("config", () => {
  it("exposes an allow-list storage path sibling to config.json", () => {
    expect(getAllowListPath().endsWith("allowlist.json")).toBe(true);
    expect(getAllowListPath()).not.toContain("config.json");
  });

  it("defaults allowDanger to enabled with type-back required", () => {
    expect(DEFAULT_CONFIG.allowDanger).toEqual({
      enabled: true,
      requireTypebackForCatastrophic: true,
    });
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run config.test.ts`
Expected: FAIL — `getAllowListPath is not exported` and `DEFAULT_CONFIG.allowDanger` is `undefined`.

- [ ] **Step 3: Implement types.ts changes**

In `pi-extension/types.ts`:

a) After the `Violation` interface, widen its severity:

```ts
export interface Violation {
  id: string;
  law: string;
  severity: "info" | "warning" | "critical";
  details: string;
  filePath?: string;
  operation?: string;
  timestamp: string;
}
```

b) Replace `CommandCheckResult` (widen category to include `"safe"`):

```ts
export interface CommandCheckResult {
  shouldHalt: boolean;
  reason?: string;
  category: "safe" | "destructive" | "elevated" | "network";
}
```

Note: `category` becomes required. `standalone/halt-checker.ts` is updated in Task 4 to always populate it.

c) Append after the `AcknowledgeHaltParams` block, before `SessionState`:

```ts
export type ApprovalScope = "once" | "session" | "always";

export type AllowListEntry =
  | { type: "exact"; command: string; addedAt: string; reason?: string; source: "prompt" | "tool" }
  | { type: "pattern"; regex: string; addedAt: string; reason: string; source: "tool" };

export interface AllowDangerConfig {
  enabled: boolean;
  requireTypebackForCatastrophic: boolean;
}

export const AllowDangerParams = Type.Object({
  action: StringEnum(["add", "remove", "list", "clear"] as const, { description: "Operation to perform" }),
  command: Type.Optional(Type.String({ description: "Exact command to add or remove (normalized before storage)" })),
  pattern: Type.Optional(Type.String({ description: "Regex pattern to add or remove" })),
  reason: Type.Optional(Type.String({ description: "Why this exception is needed (required for pattern entries)" })),
  sessionOnly: Type.Optional(Type.Boolean({ description: "Allow for this session only, do not persist (exact entries only)" })),
});
```

Leave `LogViolationParams` severity unchanged (`warning`/`critical`) — the tool is for agent-reported violations, not audit info entries.

d) In `GuardrailsConfig`, after `canary?:` block, add:

```ts
  allowDanger?: AllowDangerConfig;
```

e) In `DEFAULT_CONFIG`, add:

```ts
  allowDanger: { enabled: true, requireTypebackForCatastrophic: true },
```

- [ ] **Step 4: Implement config.ts change**

In `pi-extension/config.ts`, after `getConfigPath()`:

```ts
export function getAllowListPath(): string {
  return path.join(EXTENSION_DIR, "allowlist.json");
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `npx vitest run config.test.ts`
Expected: PASS (2 tests). Note: `halt-checker.test.ts` may now fail type-checking under vitest only if it asserts on the old optional `category` — it does not (it checks `result.category` on the `rm -rf /` case, which Task 4 keeps populated). Verify with `npx vitest run standalone/halt-checker.test.ts` still passing; if a type error appears, Task 4 resolves it.

- [ ] **Step 6: Commit**

```bash
git add pi-extension/types.ts pi-extension/config.ts pi-extension/config.test.ts
git commit -m "feat(pi): add allow-danger types and config plumbing"
```

---

### Task 2: DangerAllowList

**Files:**
- Create: `pi-extension/permissions/danger-allow-list.ts`
- Test: `pi-extension/permissions/danger-allow-list.test.ts`

- [ ] **Step 1: Write the failing test**

Create `pi-extension/permissions/danger-allow-list.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { DangerAllowList, normalizeCommand } from "./danger-allow-list.js";

describe("DangerAllowList", () => {
  let dir: string;
  let file: string;

  beforeEach(() => {
    dir = fs.mkdtempSync(path.join(os.tmpdir(), "allowlist-"));
    file = path.join(dir, "allowlist.json");
  });

  afterEach(() => {
    fs.rmSync(dir, { recursive: true, force: true });
  });

  it("normalizes commands (trim + collapse whitespace)", () => {
    expect(normalizeCommand("  git   push   --force ")).toBe("git push --force");
  });

  it("matches exact commands after normalization", () => {
    const list = new DangerAllowList(file);
    expect(list.addExact("git push --force origin main", "prompt")).toBe(true);
    expect(list.matches("git   push   --force   origin   main")?.type).toBe("exact");
    expect(list.matches("git status")).toBeUndefined();
  });

  it("rejects duplicate exact entries", () => {
    const list = new DangerAllowList(file);
    expect(list.addExact("sudo rm x", "tool", "cleanup")).toBe(true);
    expect(list.addExact("sudo   rm   x", "tool")).toBe(false);
  });

  it("matches pattern entries", () => {
    const list = new DangerAllowList(file);
    expect(list.addPattern("^git push --force", "repeated force-push workflow")).toBe(true);
    expect(list.matches("git push --force origin feature")?.type).toBe("pattern");
    expect(list.matches("git push origin feature")).toBeUndefined();
  });

  it("rejects invalid regex without throwing", () => {
    const list = new DangerAllowList(file);
    expect(list.addPattern("[unclosed", "bad")).toBe(false);
    expect(list.list()).toHaveLength(0);
  });

  it("persists entries across instances", () => {
    const list = new DangerAllowList(file);
    list.addExact("mkfs.ext4 /dev/sdb1", "tool", "disk prep");
    list.addPattern("^sudo ", "elevated ops");
    const reloaded = new DangerAllowList(file);
    expect(reloaded.matches("mkfs.ext4 /dev/sdb1")).toBeDefined();
    expect(reloaded.matches("sudo apt update")?.type).toBe("pattern");
  });

  it("starts empty when the file is corrupt", () => {
    fs.writeFileSync(file, "{not json");
    const list = new DangerAllowList(file);
    expect(list.list()).toHaveLength(0);
  });

  it("skips persisted pattern entries with invalid regex", () => {
    fs.writeFileSync(file, JSON.stringify([
      { type: "pattern", regex: "[bad", reason: "broken", addedAt: new Date().toISOString(), source: "tool" },
      { type: "exact", command: "rm -rf /", addedAt: new Date().toISOString(), source: "prompt" },
    ]));
    const list = new DangerAllowList(file);
    expect(list.list()).toHaveLength(1);
    expect(list.matches("rm -rf /")).toBeDefined();
  });

  it("removes entries by value", () => {
    const list = new DangerAllowList(file);
    list.addExact("git push --force", "prompt");
    list.addPattern("^sudo ", "elevated ops");
    expect(list.remove("git   push   --force")).toBe(true);
    expect(list.remove("^sudo ")).toBe(true);
    expect(list.remove("not there")).toBe(false);
    expect(list.list()).toHaveLength(0);
    const reloaded = new DangerAllowList(file);
    expect(reloaded.list()).toHaveLength(0);
  });

  it("clears all entries", () => {
    const list = new DangerAllowList(file);
    list.addExact("git push --force", "prompt");
    list.clear();
    expect(list.list()).toHaveLength(0);
    const reloaded = new DangerAllowList(file);
    expect(reloaded.list()).toHaveLength(0);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run permissions/danger-allow-list.test.ts`
Expected: FAIL — module `./danger-allow-list.js` not found.

- [ ] **Step 3: Implement**

Create `pi-extension/permissions/danger-allow-list.ts`:

```ts
import * as fs from "node:fs";
import { getAllowListPath } from "../config.js";
import type { AllowListEntry } from "../types.js";

/** Normalize a command for storage and comparison: trim + collapse whitespace. */
export function normalizeCommand(cmd: string): string {
  return cmd.trim().replace(/\s+/g, " ");
}

export class DangerAllowList {
  private entries: AllowListEntry[] = [];
  private compiledPatterns: Map<string, RegExp> = new Map();
  private storagePath: string;

  constructor(storagePath?: string) {
    this.storagePath = storagePath ?? getAllowListPath();
    this.load();
  }

  matches(cmd: string): AllowListEntry | undefined {
    const normalized = normalizeCommand(cmd);
    for (const entry of this.entries) {
      if (entry.type === "exact" && entry.command === normalized) return entry;
      if (entry.type === "pattern") {
        const re = this.compiledPatterns.get(entry.regex);
        if (re?.test(normalized)) return entry;
      }
    }
    return undefined;
  }

  /** Add an exact-command entry. Returns false if the normalized command is already allow-listed. */
  addExact(command: string, source: "prompt" | "tool", reason?: string): boolean {
    const normalized = normalizeCommand(command);
    if (this.entries.some((e) => e.type === "exact" && e.command === normalized)) return false;
    this.entries.push({ type: "exact", command: normalized, addedAt: new Date().toISOString(), reason, source });
    this.persist();
    return true;
  }

  /** Add a regex pattern entry. Returns false if the regex is invalid or already present. */
  addPattern(regex: string, reason: string): boolean {
    if (this.entries.some((e) => e.type === "pattern" && e.regex === regex)) return false;
    try {
      this.compiledPatterns.set(regex, new RegExp(regex));
    } catch {
      return false;
    }
    this.entries.push({ type: "pattern", regex, addedAt: new Date().toISOString(), reason, source: "tool" });
    this.persist();
    return true;
  }

  /** Remove an entry by exact command (normalized) or pattern regex string. */
  remove(commandOrRegex: string): boolean {
    const normalized = normalizeCommand(commandOrRegex);
    const before = this.entries.length;
    this.entries = this.entries.filter((e) => {
      const hit = (e.type === "exact" && e.command === normalized) || (e.type === "pattern" && e.regex === commandOrRegex);
      if (hit && e.type === "pattern") this.compiledPatterns.delete(e.regex);
      return !hit;
    });
    if (this.entries.length !== before) {
      this.persist();
      return true;
    }
    return false;
  }

  list(): AllowListEntry[] {
    return [...this.entries];
  }

  clear(): void {
    this.entries = [];
    this.compiledPatterns.clear();
    this.persist();
  }

  private load(): void {
    try {
      if (!fs.existsSync(this.storagePath)) return;
      const raw = JSON.parse(fs.readFileSync(this.storagePath, "utf-8"));
      if (!Array.isArray(raw)) return;
      for (const entry of raw) {
        if (entry?.type === "exact" && typeof entry.command === "string") {
          this.entries.push(entry as AllowListEntry);
        } else if (entry?.type === "pattern" && typeof entry.regex === "string") {
          try {
            this.compiledPatterns.set(entry.regex, new RegExp(entry.regex));
            this.entries.push(entry as AllowListEntry);
          } catch {
            console.warn(`[pi-guardrails] Skipping allow-list entry with invalid regex: ${entry.regex}`);
          }
        }
      }
    } catch {
      // Corrupt or unreadable file: start with an empty list; next persist() recreates it.
    }
  }

  private persist(): void {
    try {
      const dir = this.storagePath.substring(0, this.storagePath.lastIndexOf("/"));
      if (dir && !fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
      fs.writeFileSync(this.storagePath, JSON.stringify(this.entries, null, 2));
    } catch {
      // Best-effort persistence; the in-memory list still functions for this session.
    }
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run permissions/danger-allow-list.test.ts`
Expected: PASS (9 tests).

- [ ] **Step 5: Commit**

```bash
git add pi-extension/permissions/danger-allow-list.ts pi-extension/permissions/danger-allow-list.test.ts
git commit -m "feat(pi): add persisted danger allow-list for bash commands"
```

---

### Task 3: PermissionManager changes

**Files:**
- Modify: `pi-extension/permissions/permissions.ts`
- Modify: `pi-extension/permissions/permissions.test.ts`

- [ ] **Step 1: Write the failing tests**

Replace `pi-extension/permissions/permissions.test.ts` with:

```ts
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run permissions/permissions.test.ts`
Expected: FAIL — `isSessionDangerAllowed` / `allowSessionDanger` not defined, constructor signature mismatch.

- [ ] **Step 3: Implement**

Rewrite `pi-extension/permissions/permissions.ts`:

```ts
import * as fs from "node:fs";
import { getConfigPath } from "../config.js";
import { normalizeCommand } from "./danger-allow-list.js";

export type PermissionLevel = "auto" | "ask" | "blocked";

export interface ToolPermission {
  toolName: string;
  level: PermissionLevel;
  reason?: string;
}

export interface PermissionConfig {
  /** Default permission level for tools not in the matrix */
  defaultLevel: PermissionLevel;
  /** Per-tool permission overrides */
  tools: Record<string, PermissionLevel>;
}

const DEFAULT_PERMISSIONS: PermissionConfig = {
  defaultLevel: "auto",
  tools: {
    bash: "ask",
    write: "auto",
    edit: "auto",
    read: "auto",
    grep: "auto",
    glob: "auto",
    ls: "auto",
  },
};

export class PermissionManager {
  private config: PermissionConfig;
  private sessionOverrides: Map<string, PermissionLevel> = new Map();
  /** In-memory, non-persisted bash allowances granted with scope "session". */
  private sessionDangerAllows: Set<string> = new Set();
  private configPath: string;

  constructor(config?: Partial<PermissionConfig>, configPath?: string) {
    this.config = { ...DEFAULT_PERMISSIONS, ...config };
    this.configPath = configPath ?? getConfigPath();
    this.loadPersistedConfig();
  }

  getPermission(toolName: string): PermissionLevel {
    // Session overrides take highest priority
    const sessionOverride = this.sessionOverrides.get(toolName);
    if (sessionOverride) return sessionOverride;

    // Then check the configured matrix
    const configured = this.config.tools[toolName];
    if (configured) return configured;

    // Then default
    return this.config.defaultLevel;
  }

  setPermission(toolName: string, level: PermissionLevel, opts?: { persist?: boolean }): void {
    this.sessionOverrides.set(toolName, level);
    if (opts?.persist) this.persistToolPermission(toolName, level);
  }

  checkTool(
    toolName: string,
    args?: Record<string, unknown>,
  ): { allowed: boolean; needsConfirmation: boolean; reason?: string } {
    const level = this.getPermission(toolName);

    switch (level) {
      case "auto":
        return { allowed: true, needsConfirmation: false };
      case "blocked":
        return {
          allowed: false,
          needsConfirmation: false,
          reason: `Tool '${toolName}' is blocked by permission policy`,
        };
      case "ask":
        return {
          allowed: false,
          needsConfirmation: true,
          reason: `Tool '${toolName}' requires user confirmation before execution. Ask the user for approval first.`,
        };
    }
  }

  /** Record a session-scoped allowance for a dangerous bash command (normalized). */
  allowSessionDanger(cmd: string): void {
    this.sessionDangerAllows.add(normalizeCommand(cmd));
  }

  isSessionDangerAllowed(cmd: string): boolean {
    return this.sessionDangerAllows.has(normalizeCommand(cmd));
  }

  getPermissionMatrix(): Record<string, PermissionLevel> {
    const result: Record<string, PermissionLevel> = { ...this.config.tools };
    for (const [tool, level] of this.sessionOverrides) {
      result[tool] = level;
    }
    return result;
  }

  private persistToolPermission(toolName: string, level: PermissionLevel): void {
    try {
      let parsed: Record<string, unknown> = {};
      if (fs.existsSync(this.configPath)) {
        parsed = JSON.parse(fs.readFileSync(this.configPath, "utf-8"));
      }
      const toolPermissions = (parsed.toolPermissions ?? {}) as Record<string, unknown>;
      const tools = (toolPermissions.tools ?? {}) as Record<string, string>;
      tools[toolName] = level;
      toolPermissions.tools = tools;
      parsed.toolPermissions = toolPermissions;
      const dir = this.configPath.substring(0, this.configPath.lastIndexOf("/"));
      if (dir && !fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
      fs.writeFileSync(this.configPath, JSON.stringify(parsed, null, 2));
    } catch {
      // Best-effort persistence; the session override still applies for this session.
    }
  }

  private loadPersistedConfig(): void {
    try {
      if (!fs.existsSync(this.configPath)) return;
      const raw = fs.readFileSync(this.configPath, "utf-8");
      const parsed = JSON.parse(raw);
      if (parsed.toolPermissions) {
        this.config = { ...this.config, ...parsed.toolPermissions };
      }
    } catch {
      // Best-effort load
    }
  }
}
```

Note: `pendingConfirmations` and `confirmToolCall()` are deliberately removed (dead code — nothing populated or read them; the interactive prompt flow supersedes them).

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run permissions/permissions.test.ts`
Expected: PASS (10 tests).

- [ ] **Step 5: Commit**

```bash
git add pi-extension/permissions/permissions.ts pi-extension/permissions/permissions.test.ts
git commit -m "feat(pi): session danger allows + permission persistence; drop dead confirmation code"
```

---

### Task 4: HaltChecker — category on all results + catastrophic tier

**Files:**
- Modify: `pi-extension/standalone/halt-checker.ts`
- Modify: `pi-extension/standalone/halt-checker.test.ts`

- [ ] **Step 1: Write the failing tests**

Append inside the existing `describe("checkCommand", ...)` block in `pi-extension/standalone/halt-checker.test.ts`:

```ts
    it("reports category 'safe' for safe commands", () => {
      const checker = new HaltChecker();
      expect(checker.checkCommand("ls -la").category).toBe("safe");
    });

    it("maps classification engine categories onto the result", () => {
      const checker = new HaltChecker({ allowlist: ["rm -rf /tmp/*"], denylist: ["npm *"] });
      expect(checker.checkCommand("npm install evil").category).toBe("destructive");
      expect(checker.checkCommand("rm -rf /tmp/test").category).toBe("safe");
    });
```

Append a new top-level describe block:

```ts
  describe("isCatastrophic", () => {
    it("flags fork bombs, mkfs, rm -rf /, and dd of=/dev/", () => {
      const checker = new HaltChecker();
      expect(checker.isCatastrophic(":(){ :|:& };:")).toBe(true);
      expect(checker.isCatastrophic("mkfs.ext4 /dev/sdb1")).toBe(true);
      expect(checker.isCatastrophic("rm -rf /")).toBe(true);
      expect(checker.isCatastrophic("rm -rf /*")).toBe(true);
      expect(checker.isCatastrophic("dd if=/dev/zero of=/dev/sda")).toBe(true);
    });

    it("does not flag dangerous-but-standard commands", () => {
      const checker = new HaltChecker();
      expect(checker.isCatastrophic("git push --force origin main")).toBe(false);
      expect(checker.isCatastrophic("sudo apt install curl")).toBe(false);
      expect(checker.isCatastrophic("rm -rf node_modules")).toBe(false);
    });
  });
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run standalone/halt-checker.test.ts`
Expected: FAIL — `category` undefined for safe commands; `isCatastrophic` not a function.

- [ ] **Step 3: Implement**

Modify `pi-extension/standalone/halt-checker.ts`:

a) Add the catastrophic pattern list after the existing `DESTRUCTIVE_PATTERNS`:

```ts
const CATASTROPHIC_PATTERNS: RegExp[] = [
  /:\(\)\{\s*:\|:&\s*\}/,                          // fork bomb
  /\bmkfs\b/,                                       // filesystem format
  /\brm\s+-(rf|fr)\s+\/(\s|\*|$)/,                 // rm -rf / or /*
  /\bdd\s+\S+\s+.*of=\/dev\//,                     // dd writing to a device node
];
```

b) Replace `checkCommand` so every path populates `category`:

```ts
  checkCommand(cmd: string): CommandCheckResult {
    // Use classification engine when available
    if (this.classifyConfig) {
      const result = classifyCommand(cmd, this.classifyConfig);
      const block = shouldBlock(result, this.classifyConfig);
      const category = mapCategory(result.category);
      if (block.block) {
        return { shouldHalt: true, reason: block.reason ?? `Blocked ${result.category} command`, category };
      }
      return { shouldHalt: false, category };
    }

    // Fallback to hardcoded denylist (Sprint 0 behavior)
    const trimmed = cmd.trim().toLowerCase();

    for (const dangerous of DANGEROUS_COMMANDS) {
      if (trimmed.includes(dangerous.toLowerCase())) {
        return { shouldHalt: true, reason: `Dangerous command blocked: ${dangerous}`, category: "destructive" };
      }
    }

    for (const pattern of DESTRUCTIVE_PATTERNS) {
      if (pattern.test(trimmed)) {
        return { shouldHalt: true, reason: `Command matches dangerous pattern: ${pattern.source}`, category: "destructive" };
      }
    }

    if (/\bgit\s+push\s+--force\b/.test(trimmed)) {
      if (/\bmain\b|\bmaster\b/.test(trimmed)) {
        return { shouldHalt: true, reason: "Force-push to main/master branch is blocked", category: "destructive" };
      }
    }

    return { shouldHalt: false, category: "safe" };
  }

  /** True for commands in the catastrophic tier (type-back confirmation required). */
  isCatastrophic(cmd: string): boolean {
    const trimmed = cmd.trim().toLowerCase();
    return CATASTROPHIC_PATTERNS.some((p) => p.test(trimmed));
  }
```

c) Add the category-mapping helper below the class imports section (module level):

```ts
function mapCategory(category: string): CommandCheckResult["category"] {
  switch (category) {
    case "destructive":
      return "destructive";
    case "elevated":
      return "elevated";
    case "network":
      return "network";
    default:
      return "safe"; // read_only, constructive, unknown
  }
}
```

d) Update the `CommandCheckResult` import line at the top of the file to include it if not already:

```ts
import type { HaltResult, CommandCheckResult } from "../types.js";
```

(already present — no change needed).

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run standalone/halt-checker.test.ts`
Expected: PASS (all existing tests plus the 4 new ones).

- [ ] **Step 5: Commit**

```bash
git add pi-extension/standalone/halt-checker.ts pi-extension/standalone/halt-checker.test.ts
git commit -m "feat(pi): halt-checker reports category on all results + catastrophic tier"
```

---

### Task 5: createBashPermissionHandler

**Files:**
- Modify: `pi-extension/handlers.ts`
- Create: `pi-extension/handlers.test.ts`

- [ ] **Step 1: Write the failing test**

Create `pi-extension/handlers.test.ts`:

```ts
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run handlers.test.ts`
Expected: FAIL — `createBashPermissionHandler` not exported and `dangerAllowList` missing from `HandlerDeps`.

- [ ] **Step 3: Implement**

Modify `pi-extension/handlers.ts`:

a) Add imports at the top:

```ts
import { DangerAllowList, normalizeCommand } from "./permissions/danger-allow-list.js";
```

b) Add `dangerAllowList: DangerAllowList;` to the `HandlerDeps` interface.

c) Remove the entire `createBashSafetyHandler` function (lines ~112-134). Its dangerous-command logic moves into the new handler's `allowDanger.enabled === false` branch and into the prompt path.

d) Append the new handler and its helpers at the end of the file:

```ts
// ---- Bash permission handler: unified prompt + allow-list decision path ----

const SCOPE_OPTIONS = ["Allow once", "Allow for session", "Always allow", "Deny"] as const;

function auditBashDecision(deps: HandlerDeps, ctx: any, severity: "info" | "warning", details: string): void {
  deps.violationLog.log({ law: "halt-when-uncertain", severity, details, operation: "bash" });
  updateStatusBar(ctx, deps);
}

async function promptForBashScope(
  ctx: any,
  deps: HandlerDeps,
  cmd: string,
  category: string,
  catastrophic: boolean,
  reason?: string,
): Promise<{ block: true; reason: string } | void> {
  if (!ctx?.hasUI) {
    auditBashDecision(deps, ctx, "warning", `Bash denied (no UI available): ${cmd}`);
    return {
      block: true,
      reason: "Interactive confirmation required but no UI available; bash denied in non-interactive mode.",
    };
  }

  if (catastrophic) {
    const typed = await ctx.ui.input(`Type the command exactly to confirm: ${cmd}`, cmd);
    if (typed === undefined || normalizeCommand(typed) !== cmd) {
      auditBashDecision(deps, ctx, "warning", `Catastrophic command confirmation failed (type-back mismatch): ${cmd}`);
      return { block: true, reason: "Catastrophic command confirmation failed: typed text did not match the command." };
    }
  }

  const title = `Allow command? [${category}] ${cmd}${reason ? ` — ${reason}` : ""}`;
  const choice = await ctx.ui.select(title, [...SCOPE_OPTIONS]);

  switch (choice) {
    case "Allow once":
      auditBashDecision(deps, ctx, "info", `Bash allowed once via prompt: ${cmd}`);
      return;
    case "Allow for session":
      deps.permissionManager.allowSessionDanger(cmd);
      auditBashDecision(deps, ctx, "info", `Bash allowed for session via prompt: ${cmd}`);
      return;
    case "Always allow":
      deps.dangerAllowList.addExact(cmd, "prompt");
      auditBashDecision(deps, ctx, "info", `Bash always-allowed via prompt (persisted): ${cmd}`);
      return;
    default:
      // "Deny" or dialog dismissed (undefined)
      auditBashDecision(deps, ctx, "warning", `Bash denied via prompt: ${cmd}`);
      return { block: true, reason: `User denied command: ${cmd}` };
  }
}

export function createBashPermissionHandler(deps: HandlerDeps) {
  return async (_event: any, ctx: any): Promise<{ block: true; reason: string } | void> => {
    const event = _event as { toolName?: string; input?: Record<string, unknown> };
    if (event.toolName !== "bash") return;

    const input = event.input;
    if (!input || typeof input.command !== "string") return;
    const cmd = normalizeCommand(input.command as string);

    const allowDangerCfg = deps.config.allowDanger ?? { enabled: true, requireTypebackForCatastrophic: true };

    // Legacy mode: allowDanger disabled — reproduce the old hard-block behavior
    if (!allowDangerCfg.enabled) {
      const check = deps.haltChecker.checkCommand(cmd);
      if (check.shouldHalt) {
        deps.violationLog.log({
          law: "halt-when-uncertain",
          severity: "critical",
          details: `Blocked dangerous command: ${cmd}`,
          operation: "bash",
        });
        deps.sessionStore.recordHalt(`Dangerous command blocked: ${check.reason}`, "critical");
        updateStatusBar(ctx, deps);
        return { block: true, reason: `Command blocked: ${check.reason}` };
      }
      const perm = deps.permissionManager.checkTool("bash", event.input as Record<string, unknown>);
      if (!perm.allowed) {
        return { block: true, reason: perm.reason ?? "Tool 'bash' requires permission." };
      }
      return;
    }

    // 1. Persisted allow-list short-circuit
    const entry = deps.dangerAllowList.matches(cmd);
    if (entry) {
      auditBashDecision(deps, ctx, "info", `Bash allowed via allow-list (${entry.type}): ${cmd}`);
      return;
    }

    // 2. Session-scoped allowance
    if (deps.permissionManager.isSessionDangerAllowed(cmd)) {
      auditBashDecision(deps, ctx, "info", `Bash allowed via session allowance: ${cmd}`);
      return;
    }

    // 3. Classify
    const check = deps.haltChecker.checkCommand(cmd);

    if (!check.shouldHalt) {
      // Non-dangerous: the tool permission level decides
      const level = deps.permissionManager.getPermission("bash");
      if (level === "auto") return;
      if (level === "blocked") {
        auditBashDecision(deps, ctx, "warning", `Bash blocked by permission policy: ${cmd}`);
        return { block: true, reason: "Tool 'bash' is blocked by permission policy" };
      }
      // level === "ask"
      return promptForBashScope(ctx, deps, cmd, check.category, false);
    }

    // 4. Dangerous: strong prompt, type-back for the catastrophic tier
    const catastrophic = allowDangerCfg.requireTypebackForCatastrophic && deps.haltChecker.isCatastrophic(cmd);
    return promptForBashScope(ctx, deps, cmd, check.category, catastrophic, check.reason);
  };
}
```

e) In `createPermissionHandler` (the existing one for non-bash tools), add a bash early-return right after the `if (!event.toolName) return;` line:

```ts
    // Bash is owned by createBashPermissionHandler (prompt + allow-list flow)
    if (event.toolName === "bash") return;
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run handlers.test.ts`
Expected: PASS (10 test blocks).

- [ ] **Step 5: Commit**

```bash
git add pi-extension/handlers.ts pi-extension/handlers.test.ts
git commit -m "feat(pi): unified bash permission handler with prompts and allow-list"
```

---

### Task 6: Wire into index.ts + register guardrail_allow_danger tool

**Files:**
- Modify: `pi-extension/index.ts`

- [ ] **Step 1: Update imports**

In `pi-extension/index.ts`, change the handlers import block:

```ts
import {
  createSessionStartHandler,
  createSessionShutdownHandler,
  createReadTrackingHandler,
  createPreEditHandler,
  createBashPermissionHandler,
  createInjectionDefenseHandler,
  createOutputValidationHandler,
  createPermissionHandler,
  type HandlerDeps,
} from "./handlers.js";
```

(remove `createBashSafetyHandler`). Add:

```ts
import { DangerAllowList, normalizeCommand } from "./permissions/danger-allow-list.js";
```

and add `AllowDangerParams` to the types.ts import list.

- [ ] **Step 2: Instantiate and add to deps**

After `const permissionManager = new PermissionManager(config.toolPermissions);`, add:

```ts
const dangerAllowList = new DangerAllowList();
```

and add `dangerAllowList,` to the `deps` object.

- [ ] **Step 3: Register the guardrail_allow_danger tool**

Add after the `guardrail_status` tool registration (before the MCP bridge section):

```ts
  pi.registerTool({
    name: "guardrail_allow_danger",
    label: "Manage Dangerous Command Allow-List",
    description:
      "Add, remove, list, or clear persisted allow-list entries for dangerous bash commands. Adding a pattern entry requires a reason. Use sessionOnly for session-limited allowances of exact commands.",
    promptSnippet: "Manage the dangerous command allow-list",
    parameters: AllowDangerParams,
    execute(_id: string, params: any) {
      switch (params.action) {
        case "add": {
          if (params.command && params.pattern) {
            return { error: "Provide either 'command' or 'pattern', not both" };
          }
          if (!params.command && !params.pattern) {
            return { error: "Provide 'command' or 'pattern'" };
          }
          if (params.pattern) {
            if (!params.reason) {
              return { error: "Pattern entries require a reason" };
            }
            const ok = dangerAllowList.addPattern(params.pattern, params.reason);
            if (!ok) return { error: `Invalid regex or duplicate pattern: ${params.pattern}` };
            violationLog.log({
              law: "halt-when-uncertain",
              severity: "warning",
              details: `Danger allow-list pattern added: ${params.pattern} (${params.reason})`,
              operation: "guardrail_allow_danger",
            });
            return { added: true, type: "pattern", regex: params.pattern };
          }
          if (params.sessionOnly) {
            permissionManager.allowSessionDanger(params.command);
            violationLog.log({
              law: "halt-when-uncertain",
              severity: "info",
              details: `Session danger allowance granted: ${normalizeCommand(params.command)}`,
              operation: "guardrail_allow_danger",
            });
            return { added: true, scope: "session", command: normalizeCommand(params.command) };
          }
          const ok = dangerAllowList.addExact(params.command, "tool", params.reason);
          if (!ok) return { error: `Duplicate command already allow-listed: ${params.command}` };
          violationLog.log({
            law: "halt-when-uncertain",
            severity: "info",
            details: `Danger allow-list entry added: ${normalizeCommand(params.command)}`,
            operation: "guardrail_allow_danger",
          });
          return { added: true, type: "exact", command: normalizeCommand(params.command) };
        }
        case "remove": {
          if (!params.command && !params.pattern) {
            return { error: "Provide 'command' or 'pattern' to remove" };
          }
          const removed = dangerAllowList.remove((params.command ?? params.pattern) as string);
          if (!removed) return { error: "No matching allow-list entry found" };
          violationLog.log({
            law: "halt-when-uncertain",
            severity: "info",
            details: `Danger allow-list entry removed: ${params.command ?? params.pattern}`,
            operation: "guardrail_allow_danger",
          });
          return { removed: true };
        }
        case "list":
          return { entries: dangerAllowList.list() };
        case "clear":
          dangerAllowList.clear();
          violationLog.log({
            law: "halt-when-uncertain",
            severity: "critical",
            details: "Danger allow-list cleared",
            operation: "guardrail_allow_danger",
          });
          return { cleared: true };
        default:
          return { error: `Unknown action: ${params.action}` };
      }
    },
  });
```

- [ ] **Step 4: Swap the tool_call registrations**

Replace:

```ts
  pi.on("tool_call", createPermissionHandler(deps));
  pi.on("tool_call", createPreEditHandler(deps));
  pi.on("tool_call", createBashSafetyHandler(deps));
  pi.on("tool_call", createInjectionDefenseHandler(deps));
```

with:

```ts
  pi.on("tool_call", createPermissionHandler(deps));       // non-bash tools (skips bash internally)
  pi.on("tool_call", createBashPermissionHandler(deps));   // bash: prompt + allow-list decision path
  pi.on("tool_call", createPreEditHandler(deps));
  pi.on("tool_call", createInjectionDefenseHandler(deps));
```

- [ ] **Step 5: Verify no stale references and tests pass**

Run:

```bash
grep -rn "createBashSafetyHandler\|confirmToolCall\|pendingConfirmations" --include="*.ts" .
```

Expected: no output (all references removed; `permissions.test.ts` was rewritten in Task 3 and `handlers.ts` was cleaned in Task 5).

Run: `npx vitest run`
Expected: PASS — full pi-extension suite green.

- [ ] **Step 6: Commit**

```bash
git add pi-extension/index.ts
git commit -m "feat(pi): wire bash permission handler + guardrail_allow_danger tool"
```

---

### Task 7: Documentation updates

**Files:**
- Modify: `pi-extension/README.md`

- [ ] **Step 1: Update README sections**

a) In the **Tools** table, add a row:

```
| `guardrail_allow_danger` | Manage the persisted dangerous-command allow-list (add/remove/list/clear) |
```

b) Replace the **Bash safety** bullet under "Automatic Enforcement" with:

```
- **Bash permission prompts**: Bash commands are evaluated against the persisted danger allow-list, then classified. Dangerous commands prompt for approval (once / session / always); catastrophic-tier commands (`rm -rf /`, `mkfs`, fork bombs, `dd of=/dev/…`) additionally require typing the command back. All decisions are audited.
```

c) Replace the **Tool Permissions** section with:

````markdown
## Tool Permissions

Per-tool permission levels control which tools the agent can use:

| Level | Behavior |
|-------|----------|
| `auto` | Tool executes without confirmation |
| `ask` | Bash: interactive prompt (once / session / always). Other tools: blocked with a message telling the agent to get user approval |
| `blocked` | Tool is blocked entirely |

Configure via `toolPermissions` in config.json. Session overrides are available through the permission manager.

### Bash permission prompts

Bash commands follow a single decision path:

1. Command matches the persisted allow-list → allowed (audited)
2. Command matches a session allowance → allowed (audited)
3. Command classified safe → resolved by the `bash` permission level (`auto` / `ask` / `blocked`)
4. Command classified dangerous → interactive prompt with scope choices:
   - **Allow once** — allows this call only
   - **Allow for session** — allows this command for the rest of the session
   - **Always allow** — persists the exact command to the allow-list
   - **Deny** (or dismissing the dialog) — blocks the call
5. Catastrophic-tier commands additionally require typing the command back before the scope prompt.

In non-interactive contexts (no UI available), bash that would prompt is denied.

### Danger allow-list

Managed via the `guardrail_allow_danger` tool:

- `add` with `command` — persist an exact command (normalized); `sessionOnly: true` grants a session-limited allowance instead
- `add` with `pattern` — persist a regex pattern; requires `reason`
- `remove` / `list` / `clear` — manage entries

Stored in `~/.pi/agent/extensions/pi-guardrails/allowlist.json`. All mutations are audited; `clear` is logged at critical severity.

To restore the legacy hard-block behavior, set `allowDanger.enabled: false` in config.json.
````

d) In the **Configuration** JSON example, add after `outputValidation`:

```json
  "allowDanger": {
    "enabled": true,
    "requireTypebackForCatastrophic": true
  }
```

e) In the **Storage** section, add:

```
- `allowlist.json` — persisted dangerous-command allow-list
```

- [ ] **Step 2: Verify tests still pass and commit**

Run: `npx vitest run`
Expected: PASS (docs-only change; full suite confirms nothing broke).

```bash
git add pi-extension/README.md
git commit -m "docs(pi): document bash permission prompts and danger allow-list"
```

---

## Self-Review Notes (completed by plan author)

- **Spec coverage:** decision flow (§Architecture) → Tasks 5/6; `DangerAllowList` (§Components 1) → Task 2; `PermissionManager` changes (§Components 2) → Task 3; `HaltChecker` (§Components 4) → Task 4; tool (§Components 5) → Task 6; config (§Components 6) → Task 1; error handling (§Error handling) → implemented in Task 2 (corrupt file, invalid regex), Task 5 (no-UI deny, dismissed dialog, type-back mismatch, legacy fallback); audit (§Audit) → Task 5 (`auditBashDecision`) and Task 6 (tool mutations); testing (§Testing) → Tasks 1–5; README (§Files touched) → Task 7.
- **Type consistency:** `normalizeCommand` defined in Task 2 and reused in Tasks 3/5/6. `CommandCheckResult.category` widened in Task 1, populated in Task 4, consumed in Task 5. `ApprovalScope` declared in Task 1 (type only — the handler uses the string literals from `SCOPE_OPTIONS`; no unused-import risk).
- **Placeholder scan:** none — every code step contains complete code.
