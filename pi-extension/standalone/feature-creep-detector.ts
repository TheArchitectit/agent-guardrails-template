import type { CreepResult } from "../types.js";

export class FeatureCreepDetector {
  detectCreep(scopePaths: string[], modifiedFiles: string[]): CreepResult {
    const inScopeModified: string[] = [];
    const outOfScopeModified: string[] = [];
    const warnings: string[] = [];

    for (const file of modifiedFiles) {
      const isInScope = scopePaths.some((scope) => file.startsWith(scope));
      if (isInScope) {
        inScopeModified.push(file);
      } else {
        outOfScopeModified.push(file);
      }
    }

    // Out-of-scope files are definite creep
    if (outOfScopeModified.length > 0) {
      warnings.push(
        `${outOfScopeModified.length} file(s) modified outside authorized scope: ${outOfScopeModified.join(", ")}`,
      );
    }

    // Check for common creep signals in in-scope files
    const configFiles = inScopeModified.filter((f) =>
      /package\.json|go\.mod|Cargo\.toml|\.env|tsconfig\.json|pyproject\.toml/i.test(f),
    );
    if (configFiles.length > 0) {
      warnings.push(
        `Config file(s) modified: ${configFiles.join(", ")} — verify these are task-required, not scope creep`,
      );
    }

    const testFiles = inScopeModified.filter((f) =>
      /\.(test|spec)\.(ts|js|py|go|rs)$/i.test(f),
    );
    if (testFiles.length > 0 && inScopeModified.length - testFiles.length === 0) {
      warnings.push("Only test files modified — ensure production code exists first (production-first rule)");
    }

    return {
      hasCreep: outOfScopeModified.length > 0 || warnings.length > 0,
      inScopeModified,
      outOfScopeModified,
      warnings,
    };
  }
}
