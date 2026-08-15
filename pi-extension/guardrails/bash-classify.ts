export type CommandCategory = "read_only" | "constructive" | "destructive" | "network" | "elevated";

export interface ClassifyResult {
  category: CommandCategory;
  confidence: number;
  reason?: string;
}

export interface ClassifyConfig {
  /** Glob patterns for allowed commands (override denylist) */
  allowlist?: string[];
  /** Glob patterns for denied commands (override allowlist) */
  denylist?: string[];
  /** Minimum confidence to block (0-1, default 0.7) */
  blockThreshold?: number;
}

// ---- Category patterns ----

const READ_ONLY_PATTERNS: RegExp[] = [
  /\b(ls|cat|head|tail|less|more|file|stat|wc|du|df|find|grep|rg|ag|ack)\b/,
  /\bgit\s+(status|log|diff|branch|tag|remote|show|blame|shortlog|describe|reflog|stash\s+list)/,
  /\b(node|python|ruby|java)\s+--version/,
  /\bwhich\s+/,
  /\becho\s+/,
  /\btype\s+/,
  /\benv\b/,
  /\bprintenv\b/,
  /\bpwd\b/,
  /\b uname\b/,
  /\bwhoami\b/,
  /\bdate\b/,
  /\bcurl\s+--head\b/,
  /\bcurl\s+-I\b/,
];

const CONSTRUCTIVE_PATTERNS: RegExp[] = [
  /\b(touch|mkdir|cp|mv|ln|chmod|chown)\b/,
  /\bgit\s+(add|commit|stash|checkout|switch|merge|rebase|cherry-pick|tag|worktree)/,
  /\bnpm\s+(install|run|init|ci)/,
  /\bpip\s+(install)/,
  /\byarn\s+(add|install)/,
  /\bpnpm\s+(add|install)/,
  /\bgo\s+(mod|build|test|run|fmt|vet)/,
  /\bcargo\s+(build|test|run|add|fmt|clippy)/,
  /\bmake\b/,
  /\btsc\b/,
  /\beslint\b/,
  /\bprettier\b/,
];

const DESTRUCTIVE_PATTERNS: RegExp[] = [
  /\brm\s+(-[a-zA-Z]*f[a-zA-Z]*\s+|.*--no-preserve-root)/,
  /\bgit\s+push\s+.*--force/,
  /\bgit\s+push\s+.*-f\b/,
  /\bgit\s+reset\s+--hard/,
  /\bgit\s+clean\s+-f/,
  /\bdd\s+if=/,
  /\bmkfs\b/,
  /\b:\(\)\{\s*:\|:&\s*\}/,
  /\bdrop\s+database\b/,
  /\btruncate\s+table\b/,
  /\bDROP\s+TABLE\b/i,
];

const NETWORK_PATTERNS: RegExp[] = [
  /\bcurl\s+/,
  /\bwget\s+/,
  /\bscp\s+/,
  /\brsync\s+/,
  /\bssh\s+/,
  /\bnc\s+/,
  /\bncat\b/,
  /\bapt\s+(install|remove|update)/,
  /\byum\s+(install|remove)/,
  /\bbrew\s+(install|uninstall)/,
];

const ELEVATED_PATTERNS: RegExp[] = [
  /\bsudo\s+(?!-l\b)/,
  /\bchmod\s+777/,
  /\bchown\s+/,
  /\bpip\s+install\s+--user/,
];

// ---- Safe commands (exact match) ----

const DANGEROUS_COMMANDS: string[] = [
  "rm -rf /",
  "rm -rf /*",
  "git push --force origin main",
  "git push --force origin master",
  "git reset --hard HEAD~",
  "git clean -f",
  "drop database",
];

// ---- Classification engine ----

function matchesAny(cmd: string, patterns: RegExp[]): boolean {
  return patterns.some((p) => p.test(cmd));
}

function matchesGlob(cmd: string, patterns: string[]): boolean {
  return patterns.some((pattern) => {
    const regex = globToRegex(pattern);
    return regex.test(cmd);
  });
}

function globToRegex(glob: string): RegExp {
  const escaped = glob
    .replace(/[.+^${}()|[\]\\]/g, "\\$&")
    .replace(/\*/g, ".*")
    .replace(/\?/g, ".");
  return new RegExp(`^${escaped}$`, "i");
}

export function classifyCommand(cmd: string, config?: ClassifyConfig): ClassifyResult {
  const trimmed = cmd.trim().toLowerCase();

  // User denylist takes highest priority
  if (config?.denylist && matchesGlob(trimmed, config.denylist)) {
    return { category: "destructive", confidence: 1.0, reason: "Matched project denylist" };
  }

  // User allowlist overrides category-based blocking
  if (config?.allowlist && matchesGlob(trimmed, config.allowlist)) {
    return { category: "read_only", confidence: 1.0, reason: "Matched project allowlist" };
  }

  // Hardcoded dangerous command check
  for (const dangerous of DANGEROUS_COMMANDS) {
    if (trimmed.includes(dangerous.toLowerCase())) {
      return { category: "destructive", confidence: 1.0, reason: `Dangerous command: ${dangerous}` };
    }
  }

  // Pattern-based classification (order matters: most severe first)
  if (matchesAny(trimmed, DESTRUCTIVE_PATTERNS)) {
    return { category: "destructive", confidence: 0.9, reason: "Matches destructive pattern" };
  }
  if (matchesAny(trimmed, ELEVATED_PATTERNS)) {
    return { category: "elevated", confidence: 0.85, reason: "Matches elevated privilege pattern" };
  }
  if (matchesAny(trimmed, NETWORK_PATTERNS)) {
    // Network + pipe to shell = destructive
    if (trimmed.includes("|") && (trimmed.includes("sh") || trimmed.includes("bash"))) {
      return { category: "destructive", confidence: 0.95, reason: "Pipe from network to shell" };
    }
    return { category: "network", confidence: 0.8, reason: "Matches network pattern" };
  }
  if (matchesAny(trimmed, CONSTRUCTIVE_PATTERNS)) {
    return { category: "constructive", confidence: 0.8, reason: "Matches constructive pattern" };
  }
  if (matchesAny(trimmed, READ_ONLY_PATTERNS)) {
    return { category: "read_only", confidence: 0.9, reason: "Matches read-only pattern" };
  }

  // Unknown command: default to constructive with low confidence
  return { category: "constructive", confidence: 0.3, reason: "Unknown command" };
}

export function shouldBlock(
  result: ClassifyResult,
  config?: ClassifyConfig,
): { block: boolean; reason?: string } {
  const threshold = config?.blockThreshold ?? 0.7;

  // Always block destructive commands above confidence threshold
  if (result.category === "destructive" && result.confidence >= threshold) {
    return { block: true, reason: result.reason };
  }

  return { block: false };
}
