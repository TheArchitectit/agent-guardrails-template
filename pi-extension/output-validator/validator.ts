export interface ValidationResult {
  hasSensitiveData: boolean;
  findings: SensitiveFinding[];
  redacted?: string;
}

export interface SensitiveFinding {
  type: SecretType;
  value: string;
  start: number;
  end: number;
  severity: "warning" | "critical";
}

export type SecretType =
  | "aws_access_key"
  | "aws_secret_key"
  | "github_token"
  | "gitlab_token"
  | "slack_token"
  | "stripe_key"
  | "private_key"
  | "api_key_generic"
  | "database_url"
  | "jwt"
  | "email"
  | "ip_address";

export interface ValidatorConfig {
  /** Enable PII detection (emails, IPs) */
  enablePII?: boolean;
  /** Auto-redact in result */
  autoRedact?: boolean;
  /** Redaction replacement text */
  redactionText?: string;
}

// Secret patterns with named capture groups for precise matching
const SECRET_PATTERNS: { pattern: RegExp; type: SecretType; severity: "warning" | "critical" }[] = [
  // Cloud provider keys
  { pattern: /AKIA[0-9A-Z]{16}/g, type: "aws_access_key", severity: "critical" },
  { pattern: /aws_secret_access_key\s*[:=]\s*["']?[A-Za-z0-9/+=]{40}["']?/gi, type: "aws_secret_key", severity: "critical" },

  // GitHub / GitLab tokens
  { pattern: /gh[pousr]_[A-Za-z0-9_]{36,255}/g, type: "github_token", severity: "critical" },
  { pattern: /glpat-[A-Za-z0-9\-]{20}/g, type: "gitlab_token", severity: "critical" },

  // Slack tokens
  { pattern: /xox[baprs]-[0-9]{10,13}-[0-9]{10,13}-[a-zA-Z0-9]{24,34}/g, type: "slack_token", severity: "critical" },

  // Stripe
  { pattern: /sk_live_[A-Za-z0-9]{24,99}/g, type: "stripe_key", severity: "critical" },
  { pattern: /sk_test_[A-Za-z0-9]{24,99}/g, type: "stripe_key", severity: "warning" },

  // Private keys
  { pattern: /-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----/g, type: "private_key", severity: "critical" },

  // JWTs (three base64url segments separated by dots)
  { pattern: /eyJ[A-Za-z0-9-_]+\.eyJ[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+/g, type: "jwt", severity: "warning" },

  // Database URLs
  { pattern: /(?:postgres|mysql|mongodb|redis):\/\/[^\s"']+/gi, type: "database_url", severity: "critical" },

  // Generic API key patterns
  { pattern: /(?:api[_-]?key|apikey|api[_-]?secret)\s*[:=]\s*["']?[A-Za-z0-9\-_]{20,}["']?/gi, type: "api_key_generic", severity: "warning" },
  { pattern: /(?:secret|token|password|passwd)\s*[:=]\s*["']?[A-Za-z0-9\-_]{20,}["']?/gi, type: "api_key_generic", severity: "warning" },
];

const PII_PATTERNS: { pattern: RegExp; type: SecretType; severity: "warning" | "critical" }[] = [
  { pattern: /[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}/g, type: "email", severity: "warning" },
  { pattern: /\b(?:\d{1,3}\.){3}\d{1,3}\b/g, type: "ip_address", severity: "warning" },
];

export function validateOutput(text: string, config?: ValidatorConfig): ValidationResult {
  if (!text || typeof text !== "string") {
    return { hasSensitiveData: false, findings: [] };
  }

  const findings: SensitiveFinding[] = [];
  const patterns = [...SECRET_PATTERNS, ...(config?.enablePII ? PII_PATTERNS : [])];

  for (const { pattern, type, severity } of patterns) {
    // Reset lastIndex for global regex
    pattern.lastIndex = 0;
    let match: RegExpExecArray | null;
    while ((match = pattern.exec(text)) !== null) {
      findings.push({
        type,
        value: match[0],
        start: match.index,
        end: match.index + match[0].length,
        severity,
      });
    }
  }

  // Deduplicate overlapping findings
  findings.sort((a, b) => a.start - b.start);
  const deduped: SensitiveFinding[] = [];
  for (const f of findings) {
    if (deduped.length === 0 || f.start >= deduped[deduped.length - 1].end) {
      deduped.push(f);
    }
  }

  // Auto-redact if configured
  let redacted: string | undefined;
  if (config?.autoRedact && deduped.length > 0) {
    const replacement = config.redactionText ?? "[REDACTED]";
    let result = text;
    // Replace from end to start to preserve indices
    for (let i = deduped.length - 1; i >= 0; i--) {
      const f = deduped[i];
      result = result.substring(0, f.start) + replacement + result.substring(f.end);
    }
    redacted = result;
  }

  return {
    hasSensitiveData: deduped.length > 0,
    findings: deduped,
    redacted,
  };
}

export function getValidationSummary(result: ValidationResult): string {
  if (!result.hasSensitiveData) return "No sensitive data detected";

  const critical = result.findings.filter((f) => f.severity === "critical").length;
  const warnings = result.findings.filter((f) => f.severity === "warning").length;
  const types = [...new Set(result.findings.map((f) => f.type))].join(", ");

  return `Sensitive data detected: ${critical} critical, ${warnings} warning (${types})`;
}
