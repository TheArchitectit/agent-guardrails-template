export interface InjectionResult {
  detected: boolean;
  confidence: number;
  patterns: string[];
  severity: "low" | "medium" | "high";
}

export interface InjectionConfig {
  /** Minimum confidence to block (0-1, default 0.8) */
  blockThreshold?: number;
  /** Minimum confidence to warn (0-1, default 0.5) */
  warnThreshold?: number;
  /** Enable heuristic scoring on top of pattern matching */
  heuristicEnabled?: boolean;
}

// Patterns adapted from common prompt injection indicators
const INJECTION_PATTERNS: { pattern: RegExp; name: string; baseScore: number }[] = [
  // Direct instruction override
  { pattern: /ignore\s+(previous|above|all|your)\s+(instructions|rules|guidelines|directives)/i, name: "instruction-override", baseScore: 0.95 },
  { pattern: /forget\s+(everything|all|your)\s+(you|instructions|rules)/i, name: "memory-wipe", baseScore: 0.9 },
  { pattern: /disregard\s+(your|all|previous|the)\s+(instructions|rules|guidelines)/i, name: "disregard-instructions", baseScore: 0.9 },
  { pattern: /do not\s+(follow|obey|adhere|comply)/i, name: "disobey", baseScore: 0.85 },
  { pattern: /override\s+(safety|security|guardrail|restriction)/i, name: "safety-override", baseScore: 0.95 },

  // Role manipulation
  { pattern: /you\s+are\s+now\s+a/i, name: "role-reassign", baseScore: 0.7 },
  { pattern: /pretend\s+(you\s+are|to\s+be)/i, name: "pretend", baseScore: 0.65 },
  { pattern: /act\s+as\s+(if\s+you|a)/i, name: "act-as", baseScore: 0.6 },
  { pattern: /simulate\s+(being|a|an)/i, name: "simulate-role", baseScore: 0.6 },

  // Output manipulation
  { pattern: /output\s+(the|your|all)\s+(system|hidden|internal|prompt)/i, name: "prompt-extraction", baseScore: 0.85 },
  { pattern: /reveal\s+(your|the|system)\s+(instructions|prompt|rules)/i, name: "prompt-reveal", baseScore: 0.85 },
  { pattern: /print\s+(your|the|system)\s+(instructions|prompt)/i, name: "prompt-print", baseScore: 0.8 },
  { pattern: /show\s+me\s+(your|the)\s+(system|hidden|internal)\s+prompt/i, name: "prompt-show", baseScore: 0.8 },

  // Jailbreak patterns
  { pattern: /DAN\s*(jailbreak|mode|prompt)/i, name: "dan-jailbreak", baseScore: 0.9 },
  { pattern: /jailbreak/i, name: "jailbreak-keyword", baseScore: 0.75 },
  { pattern: /bypass\s+(safety|security|filter|guardrail|restriction)/i, name: "bypass-safety", baseScore: 0.9 },
  { pattern: /no\s+(rules|restrictions|limits|boundaries|guardrails)/i, name: "no-rules", baseScore: 0.8 },

  // Encoding tricks
  { pattern: /\\x[0-9a-f]{2}/i, name: "hex-encoding", baseScore: 0.6 },
  { pattern: /\\u[0-9a-f]{4}/i, name: "unicode-escape", baseScore: 0.5 },
  { pattern: /base64/i, name: "base64-reference", baseScore: 0.4 },

  // Delimiter injection
  { pattern: /={3,}|-{3,}|~{3,}/, name: "delimiter-injection", baseScore: 0.45 },
];

// Heuristic signals
const HEURISTIC_SIGNALS: { check: (text: string) => number; name: string }[] = [
  {
    name: "excessive-imperatives",
    check: (text) => {
      const imperatives = (text.match(/\b(do|run|execute|perform|complete|finish|write|create|delete|remove|modify|change|update|replace|override)\b/gi) || []).length;
      return imperatives > 8 ? 0.3 : imperatives > 5 ? 0.15 : 0;
    },
  },
  {
    name: "system-referencing",
    check: (text) => {
      const refs = (text.match(/\b(system|internal|hidden|backend|admin|root|elevated|privileged)\b/gi) || []).length;
      return refs > 4 ? 0.3 : refs > 2 ? 0.15 : 0;
    },
  },
  {
    name: "unusual-structure",
    check: (text) => {
      const hasMultipleSections = (text.match(/^(#{1,3}\s|\d+\.\s|[-*]\s)/gm) || []).length > 5;
      return hasMultipleSections ? 0.15 : 0;
    },
  },
];

export function detectInjection(text: string, config?: InjectionConfig): InjectionResult {
  if (!text || typeof text !== "string") {
    return { detected: false, confidence: 0, patterns: [], severity: "low" };
  }

  const matchedPatterns: string[] = [];
  let maxScore = 0;

  // Pattern matching
  for (const { pattern, name, baseScore } of INJECTION_PATTERNS) {
    if (pattern.test(text)) {
      matchedPatterns.push(name);
      maxScore = Math.max(maxScore, baseScore);
    }
  }

  // Multiple pattern matches increase confidence
  let confidence = maxScore;
  if (matchedPatterns.length > 1) {
    confidence = Math.min(confidence + (matchedPatterns.length - 1) * 0.05, 1.0);
  }

  // Heuristic scoring
  if (config?.heuristicEnabled !== false) {
    let heuristicScore = 0;
    for (const signal of HEURISTIC_SIGNALS) {
      heuristicScore += signal.check(text);
    }
    confidence = Math.min(confidence + heuristicScore, 1.0);
  }

  const blockThreshold = config?.blockThreshold ?? 0.8;
  const warnThreshold = config?.warnThreshold ?? 0.5;

  const severity: InjectionResult["severity"] =
    confidence >= blockThreshold ? "high" : confidence >= warnThreshold ? "medium" : "low";

  return {
    detected: confidence >= warnThreshold,
    confidence: Math.round(confidence * 100) / 100,
    patterns: matchedPatterns,
    severity,
  };
}

export function shouldBlockInjection(result: InjectionResult, config?: InjectionConfig): boolean {
  return result.severity === "high" && result.confidence >= (config?.blockThreshold ?? 0.8);
}
