import { spawnSync } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { afterAll, describe, expect, it } from "vitest";

const packageDir = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const manifest = JSON.parse(readFileSync(path.join(packageDir, "package.json"), "utf8"));
const readme = readFileSync(path.join(packageDir, "README.md"), "utf8");
const installer = readFileSync(path.join(packageDir, "install.mjs"), "utf8");
const tempHomes: string[] = [];

afterAll(() => {
  for (const home of tempHomes) rmSync(home, { recursive: true, force: true });
});

describe("Pi package identity", () => {
  it("uses the manifest package name in README install examples and installer help", () => {
    const identity = manifest.name as string;
    const helpHome = mkdtempSync(path.join(tmpdir(), "pi-guardrails-help-"));
    tempHomes.push(helpHome);

    const help = spawnSync(process.execPath, [path.join(packageDir, "install.mjs"), "--help"], {
      cwd: packageDir,
      encoding: "utf8",
      env: { ...process.env, HOME: helpHome, USERPROFILE: helpHome },
    });

    expect(identity).toBe("@architectit/pi-guardrails");
    expect(readme).toContain(`# ${identity}`);
    expect(readme).toContain(`pi install npm:${identity}`);
    expect(readme).toContain(`npx ${identity}`);
    expect(installer).toContain(`npx ${identity}`);
    expect(help.status).toBe(0);
    expect(help.stdout).toContain(`npx ${identity}`);
    expect(help.stdout).toContain(path.join(helpHome, ".pi", "agent", "extensions", "pi-guardrails"));
    expect(existsSync(path.join(helpHome, ".pi"))).toBe(false);
    expect(readme).not.toContain("@thearchitectit/pi-guardrails");
    expect(installer).not.toContain("@thearchitectit/pi-guardrails");
  });
});
