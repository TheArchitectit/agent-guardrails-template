import * as fs from "node:fs";
import * as path from "node:path";
import type { PatternCheckResult } from "../types.js";

interface PatternRule {
  id: string;
  description: string;
  pattern: string;
  severity: "warning" | "critical";
  filePatterns?: string[];
}

export class PatternRuleEngine {
  private rules: PatternRule[] = [];
  private loaded = false;

  loadRules(cwd: string): number {
    this.rules = [];
    this.loaded = false;

    const searchPaths = [
      path.join(cwd, ".guardrails", "prevention-rules", "pattern-rules.json"),
      path.join(cwd, ".pi", "prevention-rules", "pattern-rules.json"),
    ];

    for (const configPath of searchPaths) {
      try {
        if (!fs.existsSync(configPath)) continue;
        const raw = fs.readFileSync(configPath, "utf-8");
        const parsed = JSON.parse(raw);

        if (Array.isArray(parsed)) {
          for (const rule of parsed) {
            if (rule.id && rule.pattern) {
              this.rules.push({
                id: rule.id,
                description: rule.description ?? rule.id,
                pattern: rule.pattern,
                severity: rule.severity ?? "warning",
                filePatterns: rule.filePatterns,
              });
            }
          }
        } else if (parsed.rules && Array.isArray(parsed.rules)) {
          for (const rule of parsed.rules) {
            if (rule.id && rule.pattern) {
              this.rules.push({
                id: rule.id,
                description: rule.description ?? rule.id,
                pattern: rule.pattern,
                severity: rule.severity ?? "warning",
                filePatterns: rule.filePatterns,
              });
            }
          }
        }

        this.loaded = true;
        break;
      } catch {
        // Skip malformed config
      }
    }

    return this.rules.length;
  }

  checkPattern(code: string, filePath?: string): PatternCheckResult[] {
    if (!this.loaded) {
      this.loadRules(process.cwd());
    }

    const results: PatternCheckResult[] = [];

    for (const rule of this.rules) {
      // Skip rules that don't apply to this file type
      if (rule.filePatterns && filePath) {
        const matchesFileType = rule.filePatterns.some((fp) => {
          try {
            return new RegExp(fp).test(filePath);
          } catch {
            return filePath.endsWith(fp);
          }
        });
        if (!matchesFileType) continue;
      }

      try {
        const regex = new RegExp(rule.pattern, "i");
        const match = regex.exec(code);
        if (match) {
          results.push({
            ruleId: rule.id,
            description: rule.description,
            match: match[0],
            severity: rule.severity,
          });
        }
      } catch {
        // Skip invalid regex
      }
    }

    return results;
  }

  getRuleCount(): number {
    return this.rules.length;
  }

  isLoaded(): boolean {
    return this.loaded;
  }
}
