import type { GuardrailsConfig } from "./types.js";
import { SessionStore } from "./standalone/session-store.js";
import { FileReadStore } from "./standalone/file-read-store.js";
import { StrikeCounter } from "./standalone/strike-counter.js";
import { ScopeValidator } from "./standalone/scope-validator.js";
import { HaltChecker } from "./standalone/halt-checker.js";
import { ViolationLog } from "./standalone/violation-log.js";
import type { MCPClient } from "./mcp-bridge/mcp-client.js";
import { detectInjection, shouldBlockInjection, type InjectionConfig } from "./injection/detector.js";
import { validateOutput, getValidationSummary, type ValidatorConfig } from "./output-validator/validator.js";
import { PermissionManager, type PermissionConfig } from "./permissions/permissions.js";
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
  permissionManager: PermissionManager;
  injectionConfig?: InjectionConfig;
  validatorConfig?: ValidatorConfig;
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

// ---- Sprint 2: Injection Defense Handler ----

export function createInjectionDefenseHandler(deps: HandlerDeps) {
  return async (_event: any, ctx: any): Promise<{ block: true; reason: string } | void> => {
    const event = _event as { toolName?: string; input?: Record<string, unknown> };

    // Only scan tools that accept free-form user input
    const scannableTools = ["bash", "write", "edit"];
    if (!event.toolName || !scannableTools.includes(event.toolName)) return;

    // Extract the text to scan
    const input = event.input;
    if (!input) return;
    const textToScan = typeof input.command === "string"
      ? input.command
      : typeof input.content === "string"
        ? input.content
        : null;
    if (!textToScan) return;

    const result = detectInjection(textToScan, deps.injectionConfig);

    if (shouldBlockInjection(result, deps.injectionConfig)) {
      deps.violationLog.log({
        law: "halt-when-uncertain",
        severity: "critical",
        details: `Prompt injection detected (confidence: ${result.confidence}, patterns: ${result.patterns.join(", ")})`,
        operation: event.toolName,
      });
      updateStatusBar(ctx, deps);
      return { block: true, reason: `Prompt injection detected (confidence: ${result.confidence}). Patterns: ${result.patterns.join(", ")}. If this is a false positive, use guardrail_set_scope to allow.` };
    }

    // Low-confidence detection: warn but don't block
    if (result.detected && result.severity === "medium") {
      deps.violationLog.log({
        law: "halt-when-uncertain",
        severity: "warning",
        details: `Possible injection (confidence: ${result.confidence}, patterns: ${result.patterns.join(", ")})`,
        operation: event.toolName,
      });
      updateStatusBar(ctx, deps);
    }
  };
}

// ---- Sprint 2: Output Validation Handler ----

export function createOutputValidationHandler(deps: HandlerDeps) {
  return (_event: any, ctx: any): void => {
    const event = _event as { toolName?: string; output?: string; input?: Record<string, unknown> };

    // Scan tool output for sensitive data
    const output = event.output;
    if (!output || typeof output !== "string") return;

    const result = validateOutput(output, deps.validatorConfig);
    if (result.hasSensitiveData) {
      const summary = getValidationSummary(result);
      deps.violationLog.log({
        law: "halt-when-uncertain",
        severity: result.findings.some((f) => f.severity === "critical") ? "critical" : "warning",
        details: `Sensitive data in tool output: ${summary}`,
        operation: event.toolName,
      });
      updateStatusBar(ctx, deps);

      // Notify the user via status bar — we can't block tool_result, but we can warn
      if (ctx?.hasUI && result.findings.some((f) => f.severity === "critical")) {
        ctx.ui.setStatus("guardrails", `WARNING: ${summary}`);
      }
    }
  };
}

// ---- Sprint 2: Permission Handler ----

export function createPermissionHandler(deps: HandlerDeps) {
  return (_event: any, ctx: any): { block: true; reason: string } | void => {
    const event = _event as { toolName?: string; input?: Record<string, unknown> };
    if (!event.toolName) return;

    const result = deps.permissionManager.checkTool(event.toolName, event.input as Record<string, unknown>);

    if (!result.allowed) {
      deps.violationLog.log({
        law: "halt-when-uncertain",
        severity: "warning",
        details: `Tool '${event.toolName}' blocked by permission policy${result.reason ? `: ${result.reason}` : ""}`,
        operation: event.toolName,
      });
      updateStatusBar(ctx, deps);
      return { block: true, reason: result.reason ?? `Tool '${event.toolName}' requires permission.` };
    }
  };
}
