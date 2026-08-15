import { describe, it, expect, beforeEach, afterEach } from "vitest";
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { ExactReplacementValidator } from "./replacement-validator.js";

describe("ExactReplacementValidator", () => {
  let tmpDir: string;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "replace-"));
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  it("validates edit when old content matches file", () => {
    const filePath = path.join(tmpDir, "app.py");
    fs.writeFileSync(filePath, "def hello():\n    print('world')\n");

    const validator = new ExactReplacementValidator();
    const result = validator.validateEdit(filePath, "    print('world')");
    expect(result.valid).toBe(true);
  });

  it("rejects edit when old content does not match file", () => {
    const filePath = path.join(tmpDir, "app.py");
    fs.writeFileSync(filePath, "def hello():\n    print('changed')\n");

    const validator = new ExactReplacementValidator();
    const result = validator.validateEdit(filePath, "    print('world')");
    expect(result.valid).toBe(false);
    expect(result.reason).toContain("modified");
  });

  it("rejects edit when file does not exist", () => {
    const validator = new ExactReplacementValidator();
    const result = validator.validateEdit(path.join(tmpDir, "missing.py"), "some content");
    expect(result.valid).toBe(false);
    expect(result.reason).toContain("does not exist");
  });

  it("validates write for new file (always valid)", () => {
    const validator = new ExactReplacementValidator();
    const result = validator.validateWrite(path.join(tmpDir, "new.py"), "new content");
    expect(result.valid).toBe(true);
  });

  it("validates write for existing file with different content", () => {
    const filePath = path.join(tmpDir, "app.py");
    fs.writeFileSync(filePath, "old content");

    const validator = new ExactReplacementValidator();
    const result = validator.validateWrite(filePath, "new content");
    expect(result.valid).toBe(true);
  });

  it("notes when write content is identical to existing", () => {
    const filePath = path.join(tmpDir, "app.py");
    fs.writeFileSync(filePath, "same content");

    const validator = new ExactReplacementValidator();
    const result = validator.validateWrite(filePath, "same content");
    expect(result.valid).toBe(true);
    expect(result.reason).toContain("identical");
  });
});
