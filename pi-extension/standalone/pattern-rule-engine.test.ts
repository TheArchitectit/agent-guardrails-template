import { describe, it, expect, vi } from "vitest";
import { PatternRuleEngine } from "./pattern-rule-engine.js";

describe("PatternRuleEngine", () => {
  it("starts with no rules loaded", () => {
    const engine = new PatternRuleEngine();
    expect(engine.isLoaded()).toBe(false);
    expect(engine.getRuleCount()).toBe(0);
  });

  it("checks patterns against loaded rules", () => {
    const engine = new PatternRuleEngine();

    // Manually inject rules for testing (bypass file loading)
    (engine as any).rules = [
      {
        id: "no-eval",
        description: "Avoid eval()",
        pattern: "\\beval\\s*\\(",
        severity: "critical" as const,
      },
      {
        id: "no-console",
        description: "Avoid console.log",
        pattern: "console\\.log\\(",
        severity: "warning" as const,
      },
    ];
    (engine as any).loaded = true;

    const results = engine.checkPattern("eval(userInput)");
    expect(results).toHaveLength(1);
    expect(results[0].ruleId).toBe("no-eval");
    expect(results[0].severity).toBe("critical");
  });

  it("returns empty results when no patterns match", () => {
    const engine = new PatternRuleEngine();
    (engine as any).rules = [
      { id: "test", description: "Test", pattern: "NEVER_MATCH_THIS_12345", severity: "warning" },
    ];
    (engine as any).loaded = true;

    const results = engine.checkPattern("const x = 1;");
    expect(results).toHaveLength(0);
  });

  it("filters rules by file pattern", () => {
    const engine = new PatternRuleEngine();
    (engine as any).rules = [
      {
        id: "python-rule",
        description: "Python rule",
        pattern: "import\\s+os",
        severity: "warning" as const,
        filePatterns: ["\\.py$"],
      },
    ];
    (engine as any).loaded = true;

    // Should match for .py files
    const pyResults = engine.checkPattern("import os", "script.py");
    expect(pyResults).toHaveLength(1);

    // Should not match for .ts files
    const tsResults = engine.checkPattern("import os", "script.ts");
    expect(tsResults).toHaveLength(0);
  });
});
