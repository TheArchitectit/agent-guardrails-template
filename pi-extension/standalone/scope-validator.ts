import * as path from "node:path";

export class ScopeValidator {
  private paths: string[] = [];
  private reason: string | null = null;

  setScope(paths: string[], reason?: string): void {
    this.paths = paths.map((p) => p.replace(/\/+$/, ""));
    this.reason = reason ?? null;
  }

  isInScope(filePath: string, _operation: "read" | "edit" | "delete"): boolean {
    if (this.paths.length === 0) return true;
    const resolved = path.resolve(filePath);
    return this.paths.some((scopePath) => resolved.startsWith(scopePath));
  }

  getScope(): string[] {
    return this.paths;
  }

  getReason(): string | null {
    return this.reason;
  }

  toJSON(): { paths: string[]; reason: string | null } {
    return { paths: this.paths, reason: this.reason };
  }

  static fromJSON(data: { paths: string[]; reason?: string }): ScopeValidator {
    const validator = new ScopeValidator();
    validator.paths = data.paths;
    validator.reason = data.reason ?? null;
    return validator;
  }
}
