import type { HaltResult, CommandCheckResult } from "../types.js";

const DESTRUCTIVE_PATTERNS: RegExp[] = [
  /\brm\s+(-[a-zA-Z]*f[a-zA-Z]*\s+|.*--no-preserve-root\s+.*\/)/,
  /\bgit\s+push\s+.*--force/,
  /\bgit\s+push\s+.*-f\b/,
  /\bgit\s+reset\s+--hard/,
  /\bsudo\s+/,
  /\bchmod\s+777/,
  /\bdd\s+if=/,
  /\bmkfs\b/,
  /\b:\(\)\{\s*:\|:&\s*\}/, // fork bomb
];

const DANGEROUS_COMMANDS: string[] = [
  "rm -rf /",
  "rm -rf /*",
  "git push --force origin main",
  "git push --force origin master",
  "git reset --hard HEAD~",
  "git clean -f",
  "drop database",
];

export class HaltChecker {
  checkCommand(cmd: string): CommandCheckResult {
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

    return { shouldHalt: false };
  }

  checkHalt(operation: string, filePath?: string, details?: string): HaltResult {
    const reasons: string[] = [];
    const suggestions: string[] = [];

    if (operation === "delete" && filePath) {
      if (filePath.includes(".env") || filePath.includes("config")) {
        reasons.push(`Deleting ${filePath} could remove environment or configuration data`);
        suggestions.push("Verify this file is safe to delete before proceeding");
      }
    }

    if (details?.toLowerCase().includes("production")) {
      reasons.push("Operation may affect production environment");
      suggestions.push("Confirm test/production separation compliance");
    }

    const severity: HaltResult["severity"] =
      reasons.length === 0 ? "none" : reasons.length >= 2 ? "critical" : "warning";

    return {
      shouldHalt: severity === "critical",
      reasons,
      severity,
      suggestions,
    };
  }
}
