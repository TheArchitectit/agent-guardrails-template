import { describe, it, expect } from "vitest";
import { validateOutput, getValidationSummary } from "./validator.js";

describe("validateOutput", () => {
  it("detects AWS access keys", () => {
    const result = validateOutput("My key is AKIAIOSFODNN7EXAMPLE");
    expect(result.hasSensitiveData).toBe(true);
    expect(result.findings.some((f) => f.type === "aws_access_key")).toBe(true);
  });

  it("detects GitHub tokens", () => {
    const result = validateOutput("Token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij");
    expect(result.hasSensitiveData).toBe(true);
    expect(result.findings.some((f) => f.type === "github_token")).toBe(true);
  });

  it("detects private keys", () => {
    const result = validateOutput("-----BEGIN RSA PRIVATE KEY-----\nMIIEowI...");
    expect(result.hasSensitiveData).toBe(true);
    expect(result.findings.some((f) => f.type === "private_key")).toBe(true);
  });

  it("detects database URLs", () => {
    const result = validateOutput("postgres://user:pass@host:5432/db");
    expect(result.hasSensitiveData).toBe(true);
    expect(result.findings.some((f) => f.type === "database_url")).toBe(true);
  });

  it("detects generic API keys", () => {
    const result = validateOutput('api_key: "sk-abcdefghijklmnopqrstuvwxyz123456"');
    expect(result.hasSensitiveData).toBe(true);
    expect(result.findings.some((f) => f.type === "api_key_generic")).toBe(true);
  });

  it("detects emails when PII enabled", () => {
    const result = validateOutput("Contact: user@example.com", { enablePII: true });
    expect(result.hasSensitiveData).toBe(true);
    expect(result.findings.some((f) => f.type === "email")).toBe(true);
  });

  it("does not detect emails when PII disabled", () => {
    const result = validateOutput("Contact: user@example.com");
    expect(result.hasSensitiveData).toBe(false);
  });

  it("auto-redacts when configured", () => {
    const result = validateOutput("Key: AKIAIOSFODNN7EXAMPLE here", { autoRedact: true });
    expect(result.redacted).toContain("[REDACTED]");
    expect(result.redacted).not.toContain("AKIAIOSFODNN7EXAMPLE");
  });

  it("returns clean for safe text", () => {
    const result = validateOutput("The function returns true on success");
    expect(result.hasSensitiveData).toBe(false);
    expect(result.findings).toHaveLength(0);
  });

  it("handles empty input", () => {
    const result = validateOutput("");
    expect(result.hasSensitiveData).toBe(false);
  });
});

describe("getValidationSummary", () => {
  it("summarizes findings", () => {
    const result = validateOutput("Key: AKIAIOSFODNN7EXAMPLE", { enablePII: false });
    const summary = getValidationSummary(result);
    expect(summary).toContain("Sensitive data detected");
    expect(summary).toContain("aws_access_key");
  });

  it("reports no findings for clean text", () => {
    const result = validateOutput("safe text");
    expect(getValidationSummary(result)).toBe("No sensitive data detected");
  });
});
