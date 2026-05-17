import { describe, it, expect, afterEach } from "vitest";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { ViolationLog } from "./violation-log.js";

describe("ViolationLog", () => {
  let tmpFile: string;

  afterEach(() => {
    if (tmpFile) {
      try { fs.unlinkSync(tmpFile); } catch {}
    }
  });

  it("logs a violation and returns it with id and timestamp", () => {
    tmpFile = path.join(os.tmpdir(), `test-violations-${Date.now()}.jsonl`);
    const log = new ViolationLog(tmpFile);
    const entry = log.log({
      law: "read-before-edit",
      severity: "critical",
      details: "test violation",
    });
    expect(entry.id).toMatch(/^v-/);
    expect(entry.law).toBe("read-before-edit");
    expect(entry.timestamp).toBeTruthy();
    log.flush();
  });

  it("returns summary with counts", () => {
    tmpFile = path.join(os.tmpdir(), `test-violations-${Date.now()}.jsonl`);
    const log = new ViolationLog(tmpFile);
    log.log({ law: "test", severity: "critical", details: "a" });
    log.log({ law: "test", severity: "warning", details: "b" });
    log.log({ law: "test", severity: "warning", details: "c" });
    log.flush();
    const summary = log.getSummary();
    expect(summary.total).toBe(3);
    expect(summary.critical).toBe(1);
    expect(summary.warning).toBe(2);
  });

  it("persists to disk", () => {
    tmpFile = path.join(os.tmpdir(), `test-violations-${Date.now()}.jsonl`);
    const log = new ViolationLog(tmpFile);
    log.log({ law: "test", severity: "critical", details: "persist test" });
    log.flush();
    const content = fs.readFileSync(tmpFile, "utf-8");
    expect(content).toContain("persist test");
  });
});
