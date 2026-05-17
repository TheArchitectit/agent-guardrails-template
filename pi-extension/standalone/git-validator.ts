import type { GitValidationResult, GuardrailsConfig } from "../types.js";

const DEFAULT_PROTECTED_BRANCHES = ["main", "master"];

export class GitValidator {
  private protectedBranches: Set<string>;
  private commitFormat?: RegExp;
  private requireAIAttribution: boolean;

  constructor(config?: GuardrailsConfig["gitPolicy"]) {
    this.protectedBranches = new Set(config?.protectedBranches ?? DEFAULT_PROTECTED_BRANCHES);
    this.requireAIAttribution = config?.requireAIAttribution ?? false;

    if (config?.commitFormat) {
      try {
        this.commitFormat = new RegExp(config.commitFormat);
      } catch {
        // Invalid regex — skip format checking
      }
    }
  }

  validateGitOp(command: string): GitValidationResult {
    const trimmed = command.trim();

    // Force push via --force, -f, or +refspec syntax
    const isForcePush = /\bgit\s+push\b/.test(trimmed) &&
      (/--force/.test(trimmed) || /-f\b/.test(trimmed) || /\s\+[^+\s]/.test(trimmed));

    if (isForcePush) {
      const targetBranch = this.extractBranchFromPush(trimmed);
      if (targetBranch && this.isProtectedBranch(targetBranch)) {
        return {
          allowed: false,
          reason: `Force-push to protected branch "${targetBranch}" is not allowed`,
          category: "protected-branch",
        };
      }
      return {
        allowed: false,
        reason: "Force-push detected — confirm this is intentional",
        category: "force-push",
      };
    }

    // Git push to protected branch (general check)
    if (/\bgit\s+push\b/.test(trimmed)) {
      const targetBranch = this.extractBranchFromPush(trimmed);
      if (targetBranch && this.isProtectedBranch(targetBranch)) {
        // Regular push to protected branch is allowed but worth noting
        return { allowed: true };
      }
    }

    // Git reset --hard
    if (/\bgit\s+reset\s+--hard\b/.test(trimmed)) {
      return {
        allowed: false,
        reason: "git reset --hard is destructive and can lose uncommitted work",
        category: "destructive",
      };
    }

    // Git clean
    if (/\bgit\s+clean\b/.test(trimmed) && /-f|--force/.test(trimmed)) {
      return {
        allowed: false,
        reason: "git clean -f will permanently delete untracked files",
        category: "destructive",
      };
    }

    // Git commit message format (when commit -m is used)
    if (/\bgit\s+commit\b/.test(trimmed) && /-m\b/.test(trimmed)) {
      const message = this.extractCommitMessage(trimmed);
      if (message && this.commitFormat && !this.commitFormat.test(message)) {
        return {
          allowed: false,
          reason: `Commit message does not match required format: ${this.commitFormat.source}`,
          category: "commit-format",
        };
      }
      if (message && this.requireAIAttribution && !/Co-Authored-By:/i.test(message)) {
        return {
          allowed: false,
          reason: "Commit message must include AI attribution (Co-Authored-By:)",
          category: "commit-format",
        };
      }
    }

    return { allowed: true };
  }

  private extractBranchFromPush(cmd: string): string | null {
    // Strip "git push" prefix and all flags (--force, -f, --force-with-lease, etc.)
    const afterPush = cmd.replace(/\bgit\s+push\s+/, "");
    const parts = afterPush.split(/\s+/).filter((p) => !p.startsWith("-"));
    // parts[0] = remote (origin), parts[1] = branch (possibly with + prefix or :refspec)
    if (parts.length < 2) return null;
    let branch = parts[1];
    // Strip + refspec prefix (e.g., +main or +main:main)
    if (branch.startsWith("+")) branch = branch.substring(1);
    // Strip :refspec suffix (e.g., main:main -> main)
    const colonIdx = branch.indexOf(":");
    if (colonIdx !== -1) branch = branch.substring(0, colonIdx);
    // Skip remotes that look like refs
    if (branch.startsWith("refs/")) branch = branch.replace(/^refs\/heads\//, "");
    return branch;
  }

  private isProtectedBranch(branch: string): boolean {
    return this.protectedBranches.has(branch);
  }

  private extractCommitMessage(cmd: string): string | null {
    // Match -m "message" or -m 'message'
    const doubleQuoteMatch = cmd.match(/-m\s+"([^"]*)"/);
    if (doubleQuoteMatch) return doubleQuoteMatch[1];
    const singleQuoteMatch = cmd.match(/-m\s+'([^']*)'/);
    if (singleQuoteMatch) return singleQuoteMatch[1];
    // -m without quotes (until next flag or end)
    const bareMatch = cmd.match(/-m\s+([^-]\S*)/);
    if (bareMatch) return bareMatch[1];
    return null;
  }
}
