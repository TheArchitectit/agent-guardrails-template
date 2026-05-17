import { describe, it, expect } from "vitest";
import { ScopeValidator } from "./scope-validator.js";

describe("ScopeValidator", () => {
  it("allows all paths when scope is empty", () => {
    const validator = new ScopeValidator();
    expect(validator.isInScope("/foo/bar.ts", "edit")).toBe(true);
  });

  it("restricts to set scope paths", () => {
    const validator = new ScopeValidator();
    validator.setScope(["/src/"]);
    expect(validator.isInScope("/src/app.ts", "edit")).toBe(true);
    expect(validator.isInScope("/etc/passwd", "edit")).toBe(false);
  });

  it("strips trailing slashes from scope paths", () => {
    const validator = new ScopeValidator();
    validator.setScope(["/src///"]);
    expect(validator.isInScope("/src/app.ts", "read")).toBe(true);
  });

  it("stores the reason", () => {
    const validator = new ScopeValidator();
    validator.setScope(["/src/"], "project boundary");
    expect(validator.getReason()).toBe("project boundary");
  });

  it("returns scope paths", () => {
    const validator = new ScopeValidator();
    validator.setScope(["/src/", "/test/"]);
    expect(validator.getScope()).toEqual(["/src/", "/test/"]);
  });

  it("round-trips through JSON serialization", () => {
    const validator = new ScopeValidator();
    validator.setScope(["/src/"], "reason");
    const json = validator.toJSON();
    const restored = ScopeValidator.fromJSON(json);
    expect(restored.getScope()).toEqual(["/src/"]);
    expect(restored.getReason()).toBe("reason");
  });
});
