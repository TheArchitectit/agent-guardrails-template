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
