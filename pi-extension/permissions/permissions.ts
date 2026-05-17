import * as fs from "node:fs";
import { getConfigPath } from "../config.js";

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
  private pendingConfirmations: Map<string, { toolName: string; args: string; resolved: boolean; approved: boolean }> = new Map();

  constructor(config?: Partial<PermissionConfig>) {
    this.config = { ...DEFAULT_PERMISSIONS, ...config };
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

  setPermission(toolName: string, level: PermissionLevel, reason?: string): void {
    this.sessionOverrides.set(toolName, level);
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

  confirmToolCall(toolName: string, approved: boolean): void {
    const pending = this.pendingConfirmations.get(toolName);
    if (pending) {
      pending.resolved = true;
      pending.approved = approved;
    }
  }

  getPermissionMatrix(): Record<string, PermissionLevel> {
    const result: Record<string, PermissionLevel> = { ...this.config.tools };
    for (const [tool, level] of this.sessionOverrides) {
      result[tool] = level;
    }
    return result;
  }

  private loadPersistedConfig(): void {
    try {
      const configPath = getConfigPath();
      if (!fs.existsSync(configPath)) return;
      const raw = fs.readFileSync(configPath, "utf-8");
      const parsed = JSON.parse(raw);
      if (parsed.toolPermissions) {
        this.config = { ...this.config, ...parsed.toolPermissions };
      }
    } catch {
      // Best-effort load
    }
  }
}
