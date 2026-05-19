import { describe, it, expect, beforeEach, afterEach } from "vitest";
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { LanguageDetector } from "./language-detector.js";

describe("LanguageDetector", () => {
  let tmpDir: string;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "lang-detect-"));
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  it("starts with no detected languages", () => {
    const detector = new LanguageDetector();
    expect(detector.isScanned()).toBe(false);
    expect(detector.getDetectedLanguages()).toEqual([]);
  });

  it("detects Python from requirements.txt", () => {
    fs.writeFileSync(path.join(tmpDir, "requirements.txt"), "flask==2.0\n");
    const detector = new LanguageDetector();
    const profile = detector.detectLanguages(tmpDir);
    expect(profile.languages).toContain("python");
    expect(profile.detectedBy.python).toContain("requirements.txt");
  });

  it("detects TypeScript from tsconfig.json", () => {
    fs.writeFileSync(path.join(tmpDir, "tsconfig.json"), "{}");
    const detector = new LanguageDetector();
    const profile = detector.detectLanguages(tmpDir);
    expect(profile.languages).toContain("typescript");
    expect(profile.detectedBy.typescript).toContain("tsconfig.json");
  });

  it("detects Go from go.mod", () => {
    fs.writeFileSync(path.join(tmpDir, "go.mod"), "module example\n");
    const detector = new LanguageDetector();
    const profile = detector.detectLanguages(tmpDir);
    expect(profile.languages).toContain("go");
    expect(profile.detectedBy.go).toContain("go.mod");
  });

  it("detects Rust from Cargo.toml", () => {
    fs.writeFileSync(path.join(tmpDir, "Cargo.toml"), "[package]\nname = \"test\"\n");
    const detector = new LanguageDetector();
    const profile = detector.detectLanguages(tmpDir);
    expect(profile.languages).toContain("rust");
    expect(profile.detectedBy.rust).toContain("Cargo.toml");
  });

  it("detects languages from source file extensions", () => {
    fs.mkdirSync(path.join(tmpDir, "src"));
    fs.writeFileSync(path.join(tmpDir, "src", "main.py"), "print('hello')");
    const detector = new LanguageDetector();
    const profile = detector.detectLanguages(tmpDir);
    expect(profile.languages).toContain("python");
    expect(profile.detectedBy.python).toContain("*.py");
  });

  it("detects multiple languages", () => {
    fs.writeFileSync(path.join(tmpDir, "requirements.txt"), "flask\n");
    fs.writeFileSync(path.join(tmpDir, "go.mod"), "module example\n");
    const detector = new LanguageDetector();
    const profile = detector.detectLanguages(tmpDir);
    expect(profile.languages).toContain("python");
    expect(profile.languages).toContain("go");
  });

  it("loads language-specific rules", () => {
    // Create a language rule file
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

    const detector = new LanguageDetector();
    detector.detectLanguages(tmpDir);
    const rules = detector.loadLanguageRules(tmpDir, ["python"]);
    expect(rules).toHaveLength(1);
    expect(rules[0].id).toBe("py-eval");
    expect(rules[0].severity).toBe("critical");
  });

  it("returns ruleCount in profile when rules exist", () => {
    const rulesDir = path.join(tmpDir, ".guardrails", "prevention-rules", "languages");
    fs.mkdirSync(rulesDir, { recursive: true });
    fs.writeFileSync(
      path.join(rulesDir, "python.json"),
      JSON.stringify({
        language: "python",
        version: "1.0.0",
        detectors: ["*.py"],
        rules: [
          { id: "py-eval", name: "eval", pattern: "\\beval\\s*\\(", severity: "critical", message: "Avoid eval()" },
          { id: "py-exec", name: "exec", pattern: "\\bexec\\s*\\(", severity: "critical", message: "Avoid exec()" },
        ],
      }),
    );
    fs.writeFileSync(path.join(tmpDir, "requirements.txt"), "flask\n");

    const detector = new LanguageDetector();
    const profile = detector.detectLanguages(tmpDir);
    expect(profile.ruleCount).toBe(2);
  });

  it("skips hidden and vendor directories", () => {
    fs.mkdirSync(path.join(tmpDir, ".git"));
    fs.mkdirSync(path.join(tmpDir, "vendor"));
    fs.writeFileSync(path.join(tmpDir, ".git", "foo.py"), "");
    fs.writeFileSync(path.join(tmpDir, "vendor", "bar.py"), "");
    const detector = new LanguageDetector();
    const profile = detector.detectLanguages(tmpDir);
    expect(profile.languages).not.toContain("python");
  });
});
