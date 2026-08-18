import { describe, it, expect, beforeEach, afterEach } from "vitest";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { DangerAllowList, normalizeCommand } from "./danger-allow-list.js";

describe("DangerAllowList", () => {
  let dir: string;
  let file: string;

  beforeEach(() => {
    dir = fs.mkdtempSync(path.join(os.tmpdir(), "allowlist-"));
    file = path.join(dir, "allowlist.json");
  });

  afterEach(() => {
    fs.rmSync(dir, { recursive: true, force: true });
  });

  it("normalizes commands (trim + collapse whitespace)", () => {
    expect(normalizeCommand("  git   push   --force ")).toBe("git push --force");
  });

  it("matches exact commands after normalization", () => {
    const list = new DangerAllowList(file);
    expect(list.addExact("git push --force origin main", "prompt")).toBe(true);
    expect(list.matches("git   push   --force   origin   main")?.type).toBe("exact");
    expect(list.matches("git status")).toBeUndefined();
  });

  it("rejects duplicate exact entries", () => {
    const list = new DangerAllowList(file);
    expect(list.addExact("sudo rm x", "tool", "cleanup")).toBe(true);
    expect(list.addExact("sudo   rm   x", "tool")).toBe(false);
  });

  it("matches pattern entries", () => {
    const list = new DangerAllowList(file);
    expect(list.addPattern("^git push --force", "repeated force-push workflow")).toBe(true);
    expect(list.matches("git push --force origin feature")?.type).toBe("pattern");
    expect(list.matches("git push origin feature")).toBeUndefined();
  });

  it("rejects invalid regex without throwing", () => {
    const list = new DangerAllowList(file);
    expect(list.addPattern("[unclosed", "bad")).toBe(false);
    expect(list.list()).toHaveLength(0);
  });

  it("persists entries across instances", () => {
    const list = new DangerAllowList(file);
    list.addExact("mkfs.ext4 /dev/sdb1", "tool", "disk prep");
    list.addPattern("^sudo ", "elevated ops");
    const reloaded = new DangerAllowList(file);
    expect(reloaded.matches("mkfs.ext4 /dev/sdb1")).toBeDefined();
    expect(reloaded.matches("sudo apt update")?.type).toBe("pattern");
  });

  it("starts empty when the file is corrupt", () => {
    fs.writeFileSync(file, "{not json");
    const list = new DangerAllowList(file);
    expect(list.list()).toHaveLength(0);
  });

  it("skips persisted pattern entries with invalid regex", () => {
    fs.writeFileSync(file, JSON.stringify([
      { type: "pattern", regex: "[bad", reason: "broken", addedAt: new Date().toISOString(), source: "tool" },
      { type: "exact", command: "rm -rf /", addedAt: new Date().toISOString(), source: "prompt" },
    ]));
    const list = new DangerAllowList(file);
    expect(list.list()).toHaveLength(1);
    expect(list.matches("rm -rf /")).toBeDefined();
  });

  it("removes entries by value", () => {
    const list = new DangerAllowList(file);
    list.addExact("git push --force", "prompt");
    list.addPattern("^sudo ", "elevated ops");
    expect(list.remove("git   push   --force")).toBe(true);
    expect(list.remove("^sudo ")).toBe(true);
    expect(list.remove("not there")).toBe(false);
    expect(list.list()).toHaveLength(0);
    const reloaded = new DangerAllowList(file);
    expect(reloaded.list()).toHaveLength(0);
  });

  it("clears all entries", () => {
    const list = new DangerAllowList(file);
    list.addExact("git push --force", "prompt");
    list.clear();
    expect(list.list()).toHaveLength(0);
    const reloaded = new DangerAllowList(file);
    expect(reloaded.list()).toHaveLength(0);
  });

  it("removes patterns only by exact stored string (no normalization)", () => {
    const list = new DangerAllowList(file);
    list.addPattern("^sudo ", "elevated ops");
    // A whitespace-padded variant must NOT remove the pattern — regexes are never normalized.
    expect(list.remove("  ^sudo ")).toBe(false);
    expect(list.list()).toHaveLength(1); // still present
    // The exact stored string removes it.
    expect(list.remove("^sudo ")).toBe(true);
    expect(list.list()).toHaveLength(0);
  });
});
