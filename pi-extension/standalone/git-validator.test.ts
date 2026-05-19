import { describe, it, expect } from "vitest";
import { GitValidator } from "./git-validator.js";

describe("GitValidator", () => {
  const validator = new GitValidator();

  it("allows normal git push", () => {
    const result = validator.validateGitOp("git push origin feature-branch");
    expect(result.allowed).toBe(true);
  });

  it("blocks force-push to protected branch (main)", () => {
    const result = validator.validateGitOp("git push --force origin main");
    expect(result.allowed).toBe(false);
    expect(result.category).toBe("protected-branch");
  });

  it("blocks force-push to protected branch (master)", () => {
    const result = validator.validateGitOp("git push -f origin master");
    expect(result.allowed).toBe(false);
    expect(result.category).toBe("protected-branch");
  });

  it("blocks force-push to non-protected branch with warning", () => {
    const result = validator.validateGitOp("git push --force origin feature-branch");
    expect(result.allowed).toBe(false);
    expect(result.category).toBe("force-push");
  });

  it("blocks git reset --hard", () => {
    const result = validator.validateGitOp("git reset --hard HEAD~1");
    expect(result.allowed).toBe(false);
    expect(result.category).toBe("destructive");
  });

  it("blocks git clean -f", () => {
    const result = validator.validateGitOp("git clean -f");
    expect(result.allowed).toBe(false);
    expect(result.category).toBe("destructive");
  });

  it("allows git add, commit, status, etc.", () => {
    expect(validator.validateGitOp("git add .").allowed).toBe(true);
    expect(validator.validateGitOp("git status").allowed).toBe(true);
    expect(validator.validateGitOp("git log --oneline").allowed).toBe(true);
  });

  it("validates commit message format when configured", () => {
    const strictValidator = new GitValidator({
      commitFormat: "^(feat|fix|chore):",
    });
    const bad = strictValidator.validateGitOp('git commit -m "added feature"');
    expect(bad.allowed).toBe(false);
    expect(bad.category).toBe("commit-format");

    const good = strictValidator.validateGitOp('git commit -m "feat: add feature"');
    expect(good.allowed).toBe(true);
  });

  it("requires AI attribution when configured", () => {
    const attrValidator = new GitValidator({ requireAIAttribution: true });
    const bad = attrValidator.validateGitOp('git commit -m "feat: add feature"');
    expect(bad.allowed).toBe(false);

    const good = attrValidator.validateGitOp('git commit -m "feat: add feature\n\nCo-Authored-By: Claude <noreply@anthropic.com>"');
    expect(good.allowed).toBe(true);
  });

  it("respects custom protected branches", () => {
    const customValidator = new GitValidator({
      protectedBranches: ["main", "release"],
    });
    const result = customValidator.validateGitOp("git push --force origin release");
    expect(result.allowed).toBe(false);
    expect(result.category).toBe("protected-branch");
  });

  it("blocks +refspec force-push to protected branch", () => {
    const result = validator.validateGitOp("git push origin +main");
    expect(result.allowed).toBe(false);
    expect(result.category).toBe("protected-branch");
  });

  it("blocks +refspec:refspec force-push to protected branch", () => {
    const result = validator.validateGitOp("git push origin +main:main");
    expect(result.allowed).toBe(false);
    expect(result.category).toBe("protected-branch");
  });

  it("blocks +refspec force-push to non-protected branch", () => {
    const result = validator.validateGitOp("git push origin +feature-branch");
    expect(result.allowed).toBe(false);
    expect(result.category).toBe("force-push");
  });
});
