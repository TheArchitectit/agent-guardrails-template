import { describe, it, expect, beforeEach, afterEach } from "vitest";
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { PatternRuleEngine } from "./pattern-rule-engine.js";
import { LanguageDetector } from "./language-detector.js";

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

describe("PatternRuleEngine with LanguageDetector", () => {
  let tmpDir: string;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "pre-rules-"));
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  it("auto-loads language rules when LanguageDetector is set", () => {
    const rulesDir = path.join(tmpDir, ".guardrails", "prevention-rules", "languages");
    fs.mkdirSync(rulesDir, { recursive: true });
    fs.writeFileSync(
      path.join(rulesDir, "python.json"),
      JSON.stringify({
        language: "python",
        version: "1.0.0",
        detectors: ["*.py"],
        rules: [
          { id: "py-eval", name: "eval usage", pattern: "\\beval\\s*\\(", severity: "critical", message: "Avoid eval()" },
        ],
      }),
    );
    fs.writeFileSync(path.join(tmpDir, "requirements.txt"), "flask\n");

    const detector = new LanguageDetector();
    const engine = new PatternRuleEngine();
    engine.setLanguageDetector(detector);

    const count = engine.loadRules(tmpDir);
    expect(count).toBe(1);
    expect(engine.isLoaded()).toBe(true);

    const results = engine.checkPattern("eval(user_input)", "app.py");
    expect(results).toHaveLength(1);
    expect(results[0].ruleId).toBe("py-eval");
  });

  it("combines generic and language-specific rules", () => {
    const rulesDir = path.join(tmpDir, ".guardrails", "prevention-rules");
    const langDir = path.join(rulesDir, "languages");
    fs.mkdirSync(langDir, { recursive: true });

    fs.writeFileSync(
      path.join(rulesDir, "pattern-rules.json"),
      JSON.stringify({
        rules: [
          { id: "generic-hardcoded", description: "Hardcoded secret", pattern: "password\\s*=\\s*[\"']", severity: "critical" },
        ],
      }),
    );
    fs.writeFileSync(
      path.join(langDir, "python.json"),
      JSON.stringify({
        language: "python",
        version: "1.0.0",
        detectors: ["*.py"],
        rules: [
          { id: "py-eval", name: "eval usage", pattern: "\\beval\\s*\\(", severity: "critical", message: "Avoid eval()" },
        ],
      }),
    );
    fs.writeFileSync(path.join(tmpDir, "requirements.txt"), "flask\n");

    const detector = new LanguageDetector();
    const engine = new PatternRuleEngine();
    engine.setLanguageDetector(detector);

    const count = engine.loadRules(tmpDir);
    expect(count).toBe(2);

    // Check generic rule matches
    const genericResults = engine.checkPattern('password = "secret123"', "config.py");
    expect(genericResults.some((r) => r.ruleId === "generic-hardcoded")).toBe(true);

    // Check language rule matches
    const langResults = engine.checkPattern("eval(user_input)", "app.py");
    expect(langResults.some((r) => r.ruleId === "py-eval")).toBe(true);
  });

  it("works without LanguageDetector (backward compatible)", () => {
    const rulesDir = path.join(tmpDir, ".guardrails", "prevention-rules");
    fs.mkdirSync(rulesDir, { recursive: true });
    fs.writeFileSync(
      path.join(rulesDir, "pattern-rules.json"),
      JSON.stringify({
        rules: [
          { id: "test-rule", description: "Test", pattern: "TODO", severity: "warning" },
        ],
      }),
    );

    const engine = new PatternRuleEngine();
    const count = engine.loadRules(tmpDir);
    expect(count).toBe(1);
    expect(engine.isLoaded()).toBe(true);
  });
});
