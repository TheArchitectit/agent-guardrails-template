import { describe, it, expect } from "vitest";
import { PreWorkChecker } from "./pre-work-checker.js";
import type { ViolationLog } from "./violation-log.js";
import type { SessionStore } from "./session-store.js";

describe("PreWorkChecker", () => {
  function makeChecker(violationSummary = { total: 0, critical: 0, warning: 0 }, sessionState: any = null) {
    const vl = { getSummary: () => violationSummary } as unknown as ViolationLog;
    const ss = { getState: () => sessionState } as unknown as SessionStore;
    return new PreWorkChecker(vl, ss);
  }

  it("returns standard checklist with no violations", () => {
    const checker = makeChecker();
    const result = checker.generateChecklist("/project");
    expect(result.recentViolations).toBe(0);
    expect(result.risks).toHaveLength(1); // "no scope defined" warning
    expect(result.checklist.length).toBeGreaterThanOrEqual(4);
  });

  it("flags critical violations", () => {
    const checker = makeChecker({ total: 3, critical: 2, warning: 1 });
    const result = checker.generateChecklist("/project");
    expect(result.recentViolations).toBe(3);
    expect(result.risks.some((r) => r.category === "violations" && r.severity === "critical")).toBe(true);
  });

  it("warns when no scope is set", () => {
    const checker = makeChecker(undefined, null);
    const result = checker.generateChecklist("/project");
    expect(result.risks.some((r) => r.category === "scope")).toBe(true);
  });

  it("includes scope in checklist when set", () => {
    const checker = makeChecker(undefined, {
      scope: { paths: ["src/"], reason: "task" },
      strikes: {},
    });
    const result = checker.generateChecklist("/project");
    expect(result.checklist.some((c) => c.includes("src/"))).toBe(true);
  });

  it("warns about active strikes", () => {
    const checker = makeChecker(undefined, {
      scope: { paths: ["src/"], reason: null },
      strikes: {
        "task-1": { attempts: [{ success: false, error: "fail", timestamp: "2026-01-01" }] },
      },
    });
    const result = checker.generateChecklist("/project");
    expect(result.risks.some((r) => r.category === "strikes")).toBe(true);
  });
});
