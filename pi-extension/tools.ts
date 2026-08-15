import type { HaltResult, Attempt } from "./types.js";
import { SessionStore } from "./standalone/session-store.js";
import { FileReadStore } from "./standalone/file-read-store.js";
import { StrikeCounter } from "./standalone/strike-counter.js";
import { ScopeValidator } from "./standalone/scope-validator.js";
import { HaltChecker } from "./standalone/halt-checker.js";
import { ViolationLog } from "./standalone/violation-log.js";
import type { MCPClient } from "./mcp-bridge/mcp-client.js";

// --- Return Types ---

export interface SessionStatusResult {
  sessionId: string | null;
  mode: "standalone" | "mcp-bridge";
  scope: string[];
  filesRead: number;
  activeStrikes: number;
  violations: { total: number; critical: number; warning: number };
  mcpConnected: boolean;
}

// --- Tool Implementations ---

export function initSession(
  sessionStore: SessionStore,
  mcpClient: MCPClient,
  params: { projectSlug: string; agentType?: string; scope?: string[]; rules?: string[] },
): { sessionId: string; mode: "standalone" | "mcp-bridge"; availableTools: string[]; mcpConnected: boolean } {
  const state = sessionStore.initialize(params.projectSlug, params.scope);

  if (params.scope && params.scope.length > 0) {
    sessionStore.scopeValidator.setScope(params.scope);
  }

  const mcpConnected = sessionStore.getState()?.mcpConnected ?? false;

  return {
    sessionId: state.id,
    mode: mcpConnected ? "mcp-bridge" : "standalone",
    availableTools: mcpConnected
      ? ["guardrail_mcp"]
      : [
          "guardrail_init",
          "guardrail_record_read",
          "guardrail_verify_read",
          "guardrail_set_scope",
          "guardrail_check_scope",
          "guardrail_record_attempt",
          "guardrail_check_strikes",
          "guardrail_reset_strikes",
          "guardrail_check_halt",
          "guardrail_log_violation",
          "guardrail_status",
        ],
    mcpConnected,
  };
}

export function recordRead(
  fileReadStore: FileReadStore,
  params: { filePath: string },
): { recorded: boolean; filePath: string; readAt: string } {
  fileReadStore.record(params.filePath);
  const readAt = fileReadStore.getReadAt(params.filePath) ?? new Date().toISOString();
  return { recorded: true, filePath: params.filePath, readAt };
}

export function verifyRead(
  fileReadStore: FileReadStore,
  params: { filePath: string },
): { wasRead: boolean; filePath: string; readAt?: string } {
  const wasRead = fileReadStore.wasRead(params.filePath);
  const readAt = fileReadStore.getReadAt(params.filePath) ?? undefined;
  return { wasRead, filePath: params.filePath, readAt };
}

export function setScope(
  scopeValidator: ScopeValidator,
  sessionStore: SessionStore,
  params: { paths: string[]; reason?: string },
): { scope: string[]; reason: string | null } {
  scopeValidator.setScope(params.paths, params.reason);
  sessionStore.save();
  return { scope: scopeValidator.getScope(), reason: scopeValidator.getReason() };
}

export function checkScope(
  scopeValidator: ScopeValidator,
  params: { filePath: string; operation: "read" | "edit" | "delete" },
): { inScope: boolean; reason?: string } {
  const inScope = scopeValidator.isInScope(params.filePath, params.operation);
  if (!inScope) {
    return {
      inScope: false,
      reason: `${params.filePath} is outside the authorized scope for ${params.operation} operations`,
    };
  }
  return { inScope: true };
}

export function recordAttempt(
  strikeCounter: StrikeCounter,
  sessionStore: SessionStore,
  params: { task: string; success: boolean; error?: string },
): { task: string; strikeCount: number; maxReached: boolean } {
  const result = strikeCounter.recordAttempt(params.task, params.success, params.error);
  sessionStore.save();
  return { task: params.task, strikeCount: result.strikeCount, maxReached: result.maxReached };
}

export function checkStrikes(
  strikeCounter: StrikeCounter,
  params: { task: string },
): { task: string; strikeCount: number; maxReached: boolean; details: Attempt[] } {
  const result = strikeCounter.getStrikes(params.task);
  return { task: params.task, strikeCount: result.strikeCount, maxReached: result.maxReached, details: result.details };
}

export function resetStrikes(
  strikeCounter: StrikeCounter,
  sessionStore: SessionStore,
  params: { task: string },
): { task: string; reset: boolean } {
  const reset = strikeCounter.reset(params.task);
  sessionStore.save();
  return { task: params.task, reset };
}

export function checkHalt(
  haltChecker: HaltChecker,
  params: { operation: string; filePath?: string; details?: string },
): HaltResult {
  return haltChecker.checkHalt(params.operation, params.filePath, params.details);
}

export function logViolation(
  violationLog: ViolationLog,
  params: { law: string; severity: string; details: string; filePath?: string; operation?: string },
): { logged: boolean; violationId: string; timestamp: string } {
  const entry = violationLog.log({
    law: params.law,
    severity: params.severity as "warning" | "critical",
    details: params.details,
    filePath: params.filePath,
    operation: params.operation,
  });
  return { logged: true, violationId: entry.id, timestamp: entry.timestamp };
}

export function getStatus(
  sessionStore: SessionStore,
  fileReadStore: FileReadStore,
  strikeCounter: StrikeCounter,
  scopeValidator: ScopeValidator,
  violationLog: ViolationLog,
  mcpClient: MCPClient,
): SessionStatusResult {
  const state = sessionStore.getState();

  const { total: totalViolations, critical: criticalViolations, warning: warningViolations } = violationLog.getSummary();

  let activeStrikes = 0;
  for (const [, attempts] of strikeCounter.getAllStrikes()) {
    const failed = attempts.filter((a) => !a.success).length;
    if (failed > 0) activeStrikes++;
  }

  return {
    sessionId: state?.id ?? null,
    mode: state?.mcpConnected ? "mcp-bridge" : "standalone",
    scope: scopeValidator.getScope(),
    filesRead: fileReadStore.size,
    activeStrikes,
    violations: { total: totalViolations, critical: criticalViolations, warning: warningViolations },
    mcpConnected: state?.mcpConnected ?? false,
  };
}
