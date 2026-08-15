import type { StrikeCounter } from "./standalone/strike-counter.js";
import type { ScopeValidator } from "./standalone/scope-validator.js";
import type { ViolationLog } from "./standalone/violation-log.js";

export function renderStatusBar(deps: {
  strikeCounter: StrikeCounter;
  scopeValidator: ScopeValidator;
  violationLog: ViolationLog;
  mcpConnected: boolean;
}): string {
  const parts: string[] = [];

  // Strike indicator
  let maxActive = 0;
  for (const [, attempts] of deps.strikeCounter.getAllStrikes()) {
    let consec = 0;
    for (let i = attempts.length - 1; i >= 0; i--) {
      if (!attempts[i].success) consec++;
      else break;
    }
    if (consec > maxActive) maxActive = consec;
  }
  if (maxActive > 0) {
    parts.push(`!!${maxActive}/${deps.strikeCounter.getMaxStrikes()}`);
  }

  // Scope indicator
  const scope = deps.scopeValidator.getScope();
  if (scope.length > 0) {
    const first = scope[0].split("/").pop() ?? scope[0];
    const suffix = scope.length > 1 ? `+${scope.length - 1}` : "";
    parts.push(`${first}/${suffix}`);
  }

  // Violation count
  const { total } = deps.violationLog.getSummary();
  if (total > 0) {
    parts.push(`!${total}v`);
  }

  // MCP status
  parts.push(deps.mcpConnected ? "mcp:*" : "mcp:.");

  return parts.length > 0 ? `g: ${parts.join(" ")}` : "g: ok";
}
