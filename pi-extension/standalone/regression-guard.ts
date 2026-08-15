import * as fs from "node:fs";
import * as path from "node:path";
import type { Violation } from "../types.js";

interface FailureRegistryEntry {
  id: string;
  category: string;
  severity: "warning" | "critical";
  message: string;
  rootCause: string;
  regressionPattern: string;
  affectedFiles: string[];
  fixedAt: string;
}

export interface RegressionMatch {
  failureId: string;
  category: string;
  severity: "warning" | "critical";
  message: string;
  rootCause: string;
  regressionPattern: string;
  affectedFiles: string[];
}

export interface RegressionCheckResult {
  matches: RegressionMatch[];
  checked: number;
  riskLevel: "none" | "low" | "medium" | "high";
}

export interface FixVerificationResult {
  allFixesIntact: boolean;
  fixes: { failureId: string; intact: boolean; reason: string }[];
  summary: string;
}

export class RegressionGuard {
  private registryPath: string;
  private violationLogPath: string;

  constructor(registryDir?: string, violationLogPath?: string) {
    const base = registryDir ?? path.join(process.cwd(), ".guardrails", "regression");
    this.registryPath = path.join(base, "failure-registry.jsonl");
    this.violationLogPath = violationLogPath ?? "";
  }

  checkRegression(filePaths: string[], codeContent?: string): RegressionCheckResult {
    const failures = this.loadRegistry();
    const matches: RegressionMatch[] = [];

    for (const failure of failures) {
      const affectsRelevantFile = filePaths.some((fp) =>
        failure.affectedFiles.some((af) => fp.endsWith(af) || af.endsWith(fp) || fp.includes(af)),
      );

      if (!affectsRelevantFile) continue;

      // If code content provided, check regression pattern
      if (codeContent && failure.regressionPattern) {
        try {
          const regex = new RegExp(failure.regressionPattern);
          if (regex.test(codeContent)) {
            matches.push({
              failureId: failure.id,
              category: failure.category,
              severity: failure.severity,
              message: failure.message,
              rootCause: failure.rootCause,
              regressionPattern: failure.regressionPattern,
              affectedFiles: failure.affectedFiles,
            });
          }
        } catch {
          // Skip invalid patterns
        }
      } else {
        // Without code content, flag any file overlap
        matches.push({
          failureId: failure.id,
          category: failure.category,
          severity: failure.severity,
          message: failure.message,
          rootCause: failure.rootCause,
          regressionPattern: failure.regressionPattern,
          affectedFiles: failure.affectedFiles,
        });
      }
    }

    const riskLevel = matches.length === 0 ? "none" as const
      : matches.some((m) => m.severity === "critical") ? "high" as const
      : matches.length > 2 ? "medium" as const
      : "low" as const;

    return { matches, checked: filePaths.length, riskLevel };
  }

  verifyFixesIntact(filePath: string, currentContent: string): FixVerificationResult {
    const failures = this.loadRegistry();
    const fixes: { failureId: string; intact: boolean; reason: string }[] = [];

    for (const failure of failures) {
      const affectsFile = failure.affectedFiles.some(
        (af) => filePath.endsWith(af) || af.endsWith(filePath) || filePath.includes(af),
      );
      if (!affectsFile) continue;

      if (failure.regressionPattern) {
        try {
          const regex = new RegExp(failure.regressionPattern);
          const isIntact = !regex.test(currentContent);
          fixes.push({
            failureId: failure.id,
            intact: isIntact,
            reason: isIntact
              ? `Fix intact — regression pattern no longer present`
              : `Fix regressed — regression pattern detected again`,
          });
        } catch {
          fixes.push({
            failureId: failure.id,
            intact: true,
            reason: "Cannot verify — invalid regression pattern",
          });
        }
      }
    }

    const allIntact = fixes.length === 0 || fixes.every((f) => f.intact);
    const regressed = fixes.filter((f) => !f.intact).length;
    const summary = fixes.length === 0
      ? `No registered fixes for ${filePath}`
      : regressed > 0
        ? `${regressed} of ${fixes.length} fixes have regressed in ${filePath}`
        : `All ${fixes.length} fixes intact in ${filePath}`;

    return { allFixesIntact: allIntact, fixes, summary };
  }

  registerFailure(entry: Omit<FailureRegistryEntry, "id">): string {
    const id = `fr-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
    const record: FailureRegistryEntry = { ...entry, id };

    try {
      const dir = this.registryPath.substring(0, this.registryPath.lastIndexOf("/"));
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
      }
      // Append existing violations as registry seed if log exists
      this.seedFromViolationLog();
      fs.appendFileSync(this.registryPath, JSON.stringify(record) + "\n");
    } catch {
      // Best-effort
    }

    return id;
  }

  getActiveFailures(): FailureRegistryEntry[] {
    return this.loadRegistry();
  }

  loadRegistry(): FailureRegistryEntry[] {
    const entries: FailureRegistryEntry[] = [];

    try {
      if (!fs.existsSync(this.registryPath)) return entries;
      const lines = fs.readFileSync(this.registryPath, "utf-8").split("\n").filter(Boolean);
      for (const line of lines) {
        try {
          entries.push(JSON.parse(line) as FailureRegistryEntry);
        } catch {
          // Skip malformed entries
        }
      }
    } catch {
      // Best-effort
    }

    return entries;
  }

  private violationLogSeeded = false;

  private seedFromViolationLog(): void {
    if (this.violationLogSeeded || !this.violationLogPath) return;
    this.violationLogSeeded = true;

    try {
      if (!fs.existsSync(this.violationLogPath)) return;
      const lines = fs.readFileSync(this.violationLogPath, "utf-8").split("\n").filter(Boolean);
      // Only seed critical violations as failure registry entries
      const criticals = lines
        .map((l) => {
          try { return JSON.parse(l) as Violation; } catch { return null; }
        })
        .filter((v): v is Violation => v !== null && v.severity === "critical");

      // Seed up to 50 most recent critical violations
      const recent = criticals.slice(-50);
      const existing = this.loadRegistry();
      const existingIds = new Set(existing.map((e) => e.id));

      for (const v of recent) {
        // Derive a stable ID from violation ID to avoid re-seeding
        const seedId = `seed-${v.id}`;
        if (existingIds.has(seedId)) continue;

        const entry: FailureRegistryEntry = {
          id: seedId,
          category: v.law,
          severity: v.severity,
          message: v.details,
          rootCause: v.details,
          regressionPattern: "",
          affectedFiles: v.filePath ? [v.filePath] : [],
          fixedAt: v.timestamp,
        };

        try {
          fs.appendFileSync(this.registryPath, JSON.stringify(entry) + "\n");
        } catch {
          // Best-effort
        }
      }
    } catch {
      // Best-effort
    }
  }
}
