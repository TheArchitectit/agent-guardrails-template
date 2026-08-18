import type { GuardrailsConfig } from "./types.js";
import { SessionStore } from "./standalone/session-store.js";
import { FileReadStore } from "./standalone/file-read-store.js";
import { StrikeCounter } from "./standalone/strike-counter.js";
import { ScopeValidator } from "./standalone/scope-validator.js";
import { HaltChecker } from "./standalone/halt-checker.js";
import { ViolationLog } from "./standalone/violation-log.js";
import type { MCPClient } from "./mcp-bridge/mcp-client.js";
import { detectInjection, shouldBlockInjection, type InjectionConfig } from "./injection/detector.js";
import { CanaryTokenManager, type CanaryConfig } from "./injection/canary.js";
import { validateOutput, getValidationSummary, type ValidatorConfig } from "./output-validator/validator.js";
import { ContentFilter, type ContentFilterConfig } from "./output-validator/content-filter.js";
import { PermissionManager, type PermissionConfig } from "./permissions/permissions.js";
import { DangerAllowList, normalizeCommand } from "./permissions/danger-allow-list.js";
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
  dangerAllowList: DangerAllowList;
  injectionConfig?: InjectionConfig;
  validatorConfig?: ValidatorConfig;
  contentFilter?: ContentFilter;
  canaryManager?: CanaryTokenManager;
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
      deps.sessionStore.recordHalt(`Law 1 violation: editing ${filePath} without reading`, "critical");
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
      deps.sessionStore.recordHalt(`Law 2 violation: ${filePath} is outside scope`, "warning");
      updateStatusBar(ctx, deps);
      return { block: true, reason: `Law 2 violation: ${filePath} is outside the authorized scope. Use guardrail_set_scope to expand.` };
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
      deps.sessionStore.recordHalt(`Prompt injection detected (confidence: ${result.confidence})`, "critical");
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

    const output = event.output;
    if (!output || typeof output !== "string") return;

    // Secret/PII scanning
    const result = validateOutput(output, deps.validatorConfig);
    if (result.hasSensitiveData) {
      const summary = getValidationSummary(result);
      deps.violationLog.log({
        law: "verify-before-commit",
        severity: result.findings.some((f) => f.severity === "critical") ? "critical" : "warning",
        details: `Sensitive data in tool output: ${summary}`,
        operation: event.toolName,
      });
      updateStatusBar(ctx, deps);

      if (ctx?.hasUI && result.findings.some((f) => f.severity === "critical")) {
        ctx.ui.setStatus("guardrails", `WARNING: ${summary}`);
      }
    }

    // Content filtering — detect and warn (tool_result handlers cannot block)
    if (deps.contentFilter) {
      const filterResult = deps.contentFilter.filter(output);
      if (filterResult.blocked) {
        deps.violationLog.log({
          law: "verify-before-commit",
          severity: "critical",
          details: `Content matching denied topics detected: ${filterResult.matchedTopics.join(", ")}`,
          operation: event.toolName,
        });
        updateStatusBar(ctx, deps);
        if (ctx?.hasUI) {
          ctx.ui.setStatus("guardrails", `WARNING: Denied content detected: ${filterResult.matchedTopics.join(", ")}`);
        }
      } else if (filterResult.matchedTopics.length > 0) {
        deps.violationLog.log({
          law: "verify-before-commit",
          severity: "warning",
          details: `Content matches watched topics (not denied): ${filterResult.matchedTopics.join(", ")}`,
          operation: event.toolName,
        });
      }
    }

    // Canary token detection — detect and warn (tool_result handlers cannot block)
    if (deps.canaryManager) {
      const triggered = deps.canaryManager.check(output);
      if (triggered.length > 0) {
        deps.violationLog.log({
          law: "verify-before-commit",
          severity: "critical",
          details: `Canary token triggered — possible data exfiltration from: ${triggered.map((c) => c.filePath).join(", ")}`,
          operation: event.toolName,
        });
        updateStatusBar(ctx, deps);
        if (ctx?.hasUI) {
          ctx.ui.setStatus("guardrails", `ALERT: Data exfiltration detected from: ${triggered.map((c) => c.filePath).join(", ")}`);
        }
      }
    }
  };
}

// ---- Sprint 2: Permission Handler ----

export function createPermissionHandler(deps: HandlerDeps) {
  return (_event: any, ctx: any): { block: true; reason: string } | void => {
    const event = _event as { toolName?: string; input?: Record<string, unknown> };
    if (!event.toolName) return;

    // Bash is owned by createBashPermissionHandler (prompt + allow-list flow)
    if (event.toolName === "bash") return;

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

// ---- Bash permission handler: unified prompt + allow-list decision path ----

const SCOPE_OPTIONS = ["Allow once", "Allow for session", "Always allow", "Deny"] as const;

function auditBashDecision(deps: HandlerDeps, ctx: any, severity: "info" | "warning", details: string): void {
  deps.violationLog.log({ law: "halt-when-uncertain", severity, details, operation: "bash" });
  updateStatusBar(ctx, deps);
}

async function promptForBashScope(
  deps: HandlerDeps,
  ctx: any,
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

  try {
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
  } catch {
    auditBashDecision(deps, ctx, "warning", `Bash denied (prompt failed): ${cmd}`);
    return { block: true, reason: "Permission prompt failed; command denied." };
  }
}

export function createBashPermissionHandler(deps: HandlerDeps) {
  return async (_event: any, ctx: any): Promise<{ block: true; reason: string } | void> => {
    const event = _event as { toolName?: string; input?: Record<string, unknown> };
    if (event.toolName !== "bash") return;

    const input = event.input;
    if (!input || typeof input.command !== "string") return;
    const cmd = normalizeCommand(input.command as string);

    const allowDangerCfg = { enabled: true, requireTypebackForCatastrophic: true, ...deps.config.allowDanger };

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
        deps.violationLog.log({
          law: "halt-when-uncertain",
          severity: "warning",
          details: `Tool 'bash' blocked by permission policy${perm.reason ? `: ${perm.reason}` : ""}`,
          operation: "bash",
        });
        updateStatusBar(ctx, deps);
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
      // Safe auto-allows are intentionally not audited (volume/noise); every other allow path logs at info.
      if (level === "auto") return;
      if (level === "blocked") {
        auditBashDecision(deps, ctx, "warning", `Bash blocked by permission policy: ${cmd}`);
        return { block: true, reason: "Tool 'bash' is blocked by permission policy" };
      }
      // level === "ask"
      return promptForBashScope(deps, ctx, cmd, check.category, false);
    }

    // 4. Dangerous: explicit prior grants (allow-list/session, steps 1–2) take
    //    precedence over the level gate; "blocked" only prevents *new* interactive
    //    grants — matching how blocked+safe behaves when an allow-list entry exists.
    const level = deps.permissionManager.getPermission("bash");
    if (level === "blocked") {
      auditBashDecision(deps, ctx, "warning", `Dangerous bash blocked by permission policy: ${cmd}`);
      return { block: true, reason: "Tool 'bash' is blocked by permission policy" };
    }
    const catastrophic = allowDangerCfg.requireTypebackForCatastrophic && deps.haltChecker.isCatastrophic(cmd);
    return promptForBashScope(deps, ctx, cmd, check.category, catastrophic, check.reason);
  };
}
