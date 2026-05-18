import * as fs from "node:fs";
import type { Violation } from "../types.js";
import { getViolationsLogPath } from "../config.js";

let counter = 0;

export class ViolationLog {
  private logPath: string;

  constructor(logPath?: string) {
    this.logPath = logPath ?? getViolationsLogPath();
  }

  log(violation: Omit<Violation, "id" | "timestamp">): Violation {
    const entry: Violation = {
      ...violation,
      id: `v-${Date.now()}-${++counter}`,
      timestamp: new Date().toISOString(),
    };

    const line = JSON.stringify(entry) + "\n";
    try {
      const dir = this.logPath.substring(0, this.logPath.lastIndexOf("/"));
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
      }
      fs.appendFileSync(this.logPath, line);
    } catch {
      // Best-effort write; don't block tool execution on log failure
    }

    return entry;
  }

  flush(): void {
    // No-op: log() writes synchronously via appendFileSync
  }

  getLogPath(): string {
    return this.logPath;
  }

  getSummary(): { total: number; critical: number; warning: number } {
    let total = 0;
    let critical = 0;
    let warning = 0;
    try {
      if (!fs.existsSync(this.logPath)) return { total, critical, warning };
      const lines = fs.readFileSync(this.logPath, "utf-8").split("\n").filter(Boolean);
      for (const line of lines) {
        const v = JSON.parse(line) as Violation;
        total++;
        if (v.severity === "critical") critical++;
        else if (v.severity === "warning") warning++;
      }
    } catch {
      // Best-effort read
    }
    return { total, critical, warning };
  }
}
