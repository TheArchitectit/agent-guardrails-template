import { describe, it, expect } from "vitest";
import { FeatureCreepDetector } from "./feature-creep-detector.js";

describe("FeatureCreepDetector", () => {
  const detector = new FeatureCreepDetector();

  it("detects out-of-scope modifications", () => {
    const result = detector.detectCreep(
      ["src/components/"],
      ["src/components/App.tsx", "lib/utils.ts"],
    );
    expect(result.hasCreep).toBe(true);
    expect(result.outOfScopeModified).toContain("lib/utils.ts");
    expect(result.inScopeModified).toContain("src/components/App.tsx");
  });

  it("returns no creep when all files are in scope", () => {
    const result = detector.detectCreep(
      ["src/"],
      ["src/App.tsx", "src/utils.ts"],
    );
    expect(result.hasCreep).toBe(false);
    expect(result.outOfScopeModified).toHaveLength(0);
  });

  it("warns about config file modifications", () => {
    const result = detector.detectCreep(
      ["src/"],
      ["src/App.tsx", "src/package.json"],
    );
    expect(result.warnings.some((w) => w.includes("Config file"))).toBe(true);
  });

  it("warns when only test files are modified", () => {
    const result = detector.detectCreep(
      ["src/"],
      ["src/App.test.ts"],
    );
    expect(result.warnings.some((w) => w.includes("production-first"))).toBe(true);
  });

  it("returns empty when no files modified", () => {
    const result = detector.detectCreep(["src/"], []);
    expect(result.hasCreep).toBe(false);
    expect(result.inScopeModified).toHaveLength(0);
  });
});
