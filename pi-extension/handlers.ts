import type { GuardrailsConfig } from "./types.js";
import { SessionStore } from "./standalone/session-store.js";
import { FileReadStore } from "./standalone/file-read-store.js";
import { StrikeCounter } from "./standalone/strike-counter.js";
import { ScopeValidator } from "./standalone/scope-validator.js";
import { HaltChecker } from "./standalone/halt-checker.js";
import { ViolationLog } from "./standalone/violation-log.js";
import type { MCPClient } from "./mcp-bridge/mcp-client.js";
import { renderStatusBar } from "./status.js";

export interface HandlerDeps {
  sessionStore: SessionStore;
  fileReadStore: FileReadStore;
  scopeValidator: ScopeValidator;
  strikeCounter: StrikeCounter;
  haltChecker: HaltChecker;
  violationLog: ViolationLog;
  mcpClient: MCPClient;
  config: GuardrailsConfig;
}

function updateStatusBar(ctx: any, deps: HandlerDeps): void {
  if (!deps.config.statusBarEnabled || !ctx?.hasUI) return;
  const text = renderStatusBar({
    strikeCounter: deps.strikeCounter,
    scopeValidator: deps.scopeValidator,
    violationLog: deps.violationLog,
    mcpConnected: deps.sessionStore.getState()?.mcpConnected ?? false,
  });
  ctx.ui.setStatus("guardrails", text);
}

export function createSessionStartHandler(deps: HandlerDeps) {
  return async (_event: any, ctx: any): Promise<void> => {
    if (!deps.sessionStore.isInitialized()) {
      deps.sessionStore.initialize("default");
    }
    updateStatusBar(ctx, deps);
  };
}

export function createSessionShutdownHandler(deps: HandlerDeps) {
  return async (): Promise<void> => {
    deps.violationLog.flush();
    deps.sessionStore.save();
    await deps.mcpClient.close().catch(() => {});
  };
}

export function createReadTrackingHandler(deps: HandlerDeps) {
  return (_event: any, ctx: any): void => {
    const event = _event as { toolName?: string; input?: Record<string, unknown> };
    if (event.toolName !== "read") return;

    const input = event.input;
    if (!input || typeof input.path !== "string") return;

    deps.fileReadStore.record(input.path);
    updateStatusBar(ctx, deps);
  };
}

export function createPreEditHandler(deps: HandlerDeps) {
  return (_event: any, ctx: any): { block: true; reason: string } | void => {
    const event = _event as { toolName?: string; input?: Record<string, unknown> };
    if (event.toolName !== "edit" && event.toolName !== "write") return;

    const input = event.input;
    if (!input || typeof input.path !== "string") return;
    const filePath = input.path as string;

    // Law 1: Read Before Editing
    if (deps.config.enabledRules.includes("four-laws") && !deps.fileReadStore.wasRead(filePath)) {
      deps.violationLog.log({
        law: "read-before-edit",
        severity: "critical",
        details: `Attempted to edit ${filePath} without reading it first`,
        filePath,
        operation: event.toolName,
      });
      updateStatusBar(ctx, deps);
      return { block: true, reason: `Law 1 violation: You must read ${filePath} before editing it. Use guardrail_record_read or read the file first.` };
    }

    // Law 2: Stay in Scope
    if (deps.config.enabledRules.includes("scope-validator") && !deps.scopeValidator.isInScope(filePath, "edit")) {
      deps.violationLog.log({
        law: "stay-in-scope",
        severity: "warning",
        details: `Attempted to edit ${filePath} which is outside the authorized scope`,
        filePath,
        operation: event.toolName,
      });
      updateStatusBar(ctx, deps);
      return { block: true, reason: `Law 2 violation: ${filePath} is outside the authorized scope. Use guardrail_set_scope to expand.` };
    }
  };
}

export function createBashSafetyHandler(deps: HandlerDeps) {
  return async (_event: any, ctx: any): Promise<{ block: true; reason: string } | void> => {
    const event = _event as { toolName?: string; input?: Record<string, unknown> };
    if (event.toolName !== "bash") return;

    const input = event.input;
    if (!input || typeof input.command !== "string") return;
    const cmd = input.command as string;

    const result = deps.haltChecker.checkCommand(cmd);
    if (result.shouldHalt) {
      deps.violationLog.log({
        law: "halt-when-uncertain",
        severity: "critical",
        details: `Blocked dangerous command: ${cmd}`,
        operation: "bash",
      });
      updateStatusBar(ctx, deps);
      return { block: true, reason: `Command blocked: ${result.reason}` };
    }
  };
}
