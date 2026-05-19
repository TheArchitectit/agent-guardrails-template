import type { ViolationLog } from "./violation-log.js";
import type { SessionStore } from "./session-store.js";
import type { PreWorkCheckResult } from "../types.js";

export class PreWorkChecker {
  private violationLog: ViolationLog;
  private sessionStore: SessionStore;

  constructor(violationLog: ViolationLog, sessionStore: SessionStore) {
    this.violationLog = violationLog;
    this.sessionStore = sessionStore;
  }

  generateChecklist(cwd: string): PreWorkCheckResult {
    const summary = this.violationLog.getSummary();
    const state = this.sessionStore.getState();

    const risks: PreWorkCheckResult["risks"] = [];
    const checklist: string[] = [];

    // Check recent violations
    if (summary.critical > 0) {
      risks.push({
        category: "violations",
        description: `${summary.critical} critical violations in recent history — review before proceeding`,
        severity: "critical",
      });
      checklist.push("Review recent critical violations and ensure they won't recur");
    }
    if (summary.warning > 0) {
      risks.push({
        category: "violations",
        description: `${summary.warning} warnings in recent history`,
        severity: "warning",
      });
      checklist.push("Check if recent warnings indicate systemic issues");
    }

    // Check scope
    if (state?.scope?.paths && state.scope.paths.length > 0) {
      checklist.push(`Stay within authorized scope: ${state.scope.paths.join(", ")}`);
    } else {
      risks.push({
        category: "scope",
        description: "No scope defined — set scope before making changes",
        severity: "warning",
      });
      checklist.push("Set guardrail scope using guardrail_set_scope");
    }

    // Check strike status
    if (state?.strikes) {
      const taskIds = Object.keys(state.strikes);
      for (const taskId of taskIds) {
        const attempts = state.strikes[taskId].attempts;
        const failures = attempts.filter((a) => !a.success).length;
        if (failures > 0) {
          risks.push({
            category: "strikes",
            description: `Task "${taskId}" has ${failures} recent strike(s)`,
            severity: failures >= 3 ? "critical" : "warning",
          });
        }
      }
      checklist.push("Check strike counts before retrying failed tasks");
    }

    // Standard pre-work checks
    checklist.push("Read all target files before editing (Law 1)");
    checklist.push("Verify changes are within authorized scope (Law 2)");
    checklist.push("Plan to test/validate changes before committing (Law 3)");
    checklist.push("Halt and ask if uncertain about any aspect (Law 4)");

    return {
      risks,
      recentViolations: summary.total,
      checklist,
    };
  }
}
