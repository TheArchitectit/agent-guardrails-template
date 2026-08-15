import { describe, it, expect, beforeEach, afterEach } from "vitest";
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { RegressionGuard } from "./regression-guard.js";

describe("RegressionGuard", () => {
  let tmpDir: string;
  let registryDir: string;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "regression-"));
    registryDir = path.join(tmpDir, "regression");
    fs.mkdirSync(registryDir, { recursive: true });
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  it("returns empty matches when no failures registered", () => {
    const guard = new RegressionGuard(registryDir);
    const result = guard.checkRegression(["src/app.py"]);
    expect(result.matches).toEqual([]);
    expect(result.riskLevel).toBe("none");
    expect(result.checked).toBe(1);
  });

  it("detects regression when file path overlaps with registered failure", () => {
    const guard = new RegressionGuard(registryDir);
    guard.registerFailure({
      category: "security",
      severity: "critical",
      message: "SQL injection in user query",
      rootCause: "String concatenation in SQL query",
      regressionPattern: "execute\\s*\\(.*\\+\\s*",
      affectedFiles: ["src/db/query.py"],
      fixedAt: "2025-01-01T00:00:00Z",
    });

    const result = guard.checkRegression(["src/db/query.py"]);
    expect(result.matches).toHaveLength(1);
    expect(result.matches[0].failureId).toMatch(/^fr-/);
    expect(result.riskLevel).toBe("high");
  });

  it("matches regression pattern against code content", () => {
    const guard = new RegressionGuard(registryDir);
    guard.registerFailure({
      category: "security",
      severity: "critical",
      message: "SQL injection",
      rootCause: "String concat in SQL",
      regressionPattern: "execute\\s*\\(.*\\+\\s*",
      affectedFiles: ["app.py"],
      fixedAt: "2025-01-01T00:00:00Z",
    });

    // Code that matches the regression pattern
    const result = guard.checkRegression(["app.py"], 'cursor.execute("SELECT * FROM users WHERE id=" + user_id)');
    expect(result.matches).toHaveLength(1);
    expect(result.matches[0].failureId).toMatch(/^fr-/);
  });

  it("does not match when code content does not match regression pattern", () => {
    const guard = new RegressionGuard(registryDir);
    guard.registerFailure({
      category: "security",
      severity: "critical",
      message: "SQL injection",
      rootCause: "String concat in SQL",
      regressionPattern: "execute\\s*\\(.*\\+\\s*",
      affectedFiles: ["app.py"],
      fixedAt: "2025-01-01T00:00:00Z",
    });

    // Safe parameterized query
    const result = guard.checkRegression(["app.py"], 'cursor.execute("SELECT * FROM users WHERE id=?", (user_id,))');
    expect(result.matches).toHaveLength(0);
    expect(result.riskLevel).toBe("none");
  });

  it("returns low risk for warning-level matches", () => {
    const guard = new RegressionGuard(registryDir);
    guard.registerFailure({
      category: "style",
      severity: "warning",
      message: "Bare except clause",
      rootCause: "Missing specific exception type",
      regressionPattern: "except\\s*:",
      affectedFiles: ["handler.py"],
      fixedAt: "2025-01-01T00:00:00Z",
    });

    const result = guard.checkRegression(["handler.py"], "try:\n    pass\nexcept:\n    pass");
    expect(result.matches).toHaveLength(1);
    expect(result.riskLevel).toBe("low");
  });

  it("verifies fixes are intact", () => {
    const guard = new RegressionGuard(registryDir);
    guard.registerFailure({
      category: "security",
      severity: "critical",
      message: "SQL injection",
      rootCause: "String concat",
      regressionPattern: "execute\\s*\\(.*\\+\\s*",
      affectedFiles: ["app.py"],
      fixedAt: "2025-01-01T00:00:00Z",
    });

    // Safe code — fix is intact
    const intact = guard.verifyFixesIntact("app.py", 'cursor.execute("SELECT * FROM users WHERE id=?", (id,))');
    expect(intact.allFixesIntact).toBe(true);
    expect(intact.fixes).toHaveLength(1);
    expect(intact.fixes[0].intact).toBe(true);

    // Regressed code — fix broken
    const regressed = guard.verifyFixesIntact("app.py", 'cursor.execute("SELECT * FROM " + table)');
    expect(regressed.allFixesIntact).toBe(false);
    expect(regressed.fixes[0].intact).toBe(false);
  });

  it("returns summary for files with no registered fixes", () => {
    const guard = new RegressionGuard(registryDir);
    const result = guard.verifyFixesIntact("new_file.py", "print('hello')");
    expect(result.allFixesIntact).toBe(true);
    expect(result.summary).toContain("No registered fixes");
  });

  it("persists failure registry to disk", () => {
    const guard1 = new RegressionGuard(registryDir);
    guard1.registerFailure({
      category: "security",
      severity: "critical",
      message: "Hardcoded secret",
      rootCause: "Credential in source",
      regressionPattern: "password\\s*=\\s*[\"']",
      affectedFiles: ["config.py"],
      fixedAt: "2025-01-01T00:00:00Z",
    });

    // New instance reading same registry dir
    const guard2 = new RegressionGuard(registryDir);
    const failures = guard2.getActiveFailures();
    expect(failures).toHaveLength(1);
    expect(failures[0].message).toBe("Hardcoded secret");
  });
});
