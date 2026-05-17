import { describe, it, expect } from "vitest";
import { HaltChecker } from "./halt-checker.js";

describe("HaltChecker", () => {
  describe("checkCommand", () => {
    it("blocks rm -rf /", () => {
      const checker = new HaltChecker();
      const result = checker.checkCommand("rm -rf /");
      expect(result.shouldHalt).toBe(true);
      expect(result.category).toBe("destructive");
    });

    it("blocks git push --force to main", () => {
      const checker = new HaltChecker();
      const result = checker.checkCommand("git push --force origin main");
      expect(result.shouldHalt).toBe(true);
    });

    it("blocks sudo", () => {
      const checker = new HaltChecker();
      const result = checker.checkCommand("sudo apt install something");
      expect(result.shouldHalt).toBe(true);
    });

    it("allows safe commands", () => {
      const checker = new HaltChecker();
      expect(checker.checkCommand("ls -la").shouldHalt).toBe(false);
      expect(checker.checkCommand("git status").shouldHalt).toBe(false);
      expect(checker.checkCommand("echo hello").shouldHalt).toBe(false);
    });

    it("blocks chmod 777", () => {
      const checker = new HaltChecker();
      expect(checker.checkCommand("chmod 777 /tmp").shouldHalt).toBe(true);
    });

    it("blocks fork bomb", () => {
      const checker = new HaltChecker();
      expect(checker.checkCommand(":(){ :|:& };").shouldHalt).toBe(true);
    });

    it("uses classification engine when configured", () => {
      const checker = new HaltChecker({ allowlist: ["rm -rf /tmp/*"], denylist: ["npm *"] });
      const allowed = checker.checkCommand("rm -rf /tmp/test");
      expect(allowed.shouldHalt).toBe(false);
      const denied = checker.checkCommand("npm install evil");
      expect(denied.shouldHalt).toBe(true);
    });
  });

  describe("checkHalt", () => {
    it("returns none for safe operations", () => {
      const checker = new HaltChecker();
      const result = checker.checkHalt("edit", "/src/app.ts");
      expect(result.severity).toBe("none");
      expect(result.shouldHalt).toBe(false);
    });

    it("warns on deleting config files", () => {
      const checker = new HaltChecker();
      const result = checker.checkHalt("delete", "/project/.env");
      expect(result.severity).toBe("warning");
      expect(result.reasons.length).toBeGreaterThan(0);
    });

    it("flags production-affected operations", () => {
      const checker = new HaltChecker();
      const result = checker.checkHalt("deploy", undefined, "pushing to production");
      expect(result.reasons).toContain("Operation may affect production environment");
    });
  });
});
