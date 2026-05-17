import { describe, it, expect, afterEach } from "vitest";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { PolicyLoader } from "./policy-loader.js";
import type { GuardrailsConfig } from "../types.js";

describe("PolicyLoader", () => {
  const tmpDir = path.join(os.tmpdir(), `policy-test-${Date.now()}`);

  afterEach(() => {
    try { fs.rmSync(tmpDir, { recursive: true }); } catch {}
  });

  it("loads project policy from .pi-guardrails.json", () => {
    fs.mkdirSync(tmpDir, { recursive: true });
    fs.writeFileSync(
      path.join(tmpDir, ".pi-guardrails.json"),
      JSON.stringify({ maxStrikes: 5, defaultScope: ["/src/"] }),
    );

    const loader = new PolicyLoader();
    const layer = loader.loadProjectPolicy(tmpDir);
    expect(layer).not.toBeNull();
    expect(layer!.name).toBe("project");
    expect(layer!.config.maxStrikes).toBe(5);
  });

  it("returns null when no project config exists", () => {
    const loader = new PolicyLoader();
    const layer = loader.loadProjectPolicy("/nonexistent/path");
    expect(layer).toBeNull();
  });

  it("merges layers with project overriding org", () => {
    fs.mkdirSync(tmpDir, { recursive: true });
    fs.writeFileSync(
      path.join(tmpDir, ".pi-guardrails.json"),
      JSON.stringify({ maxStrikes: 7 }),
    );

    const orgDir = path.join(tmpDir, "org");
    fs.mkdirSync(orgDir, { recursive: true });
    fs.writeFileSync(
      path.join(orgDir, "guardrails.json"),
      JSON.stringify({ maxStrikes: 3, statusBarEnabled: false }),
    );

    const loader = new PolicyLoader();
    loader.loadOrgPolicy(path.join(orgDir, "guardrails.json"));
    loader.loadProjectPolicy(tmpDir);

    const merged = loader.merge({ maxStrikes: 2, statusBarEnabled: true } as GuardrailsConfig);
    // Project (7) overrides org (3) overrides base (2)
    expect(merged.maxStrikes).toBe(7);
    // Org disabled status bar, project didn't override, so org wins
    expect(merged.statusBarEnabled).toBe(false);
  });

  it("returns empty layers when nothing loaded", () => {
    const loader = new PolicyLoader();
    expect(loader.getLayers()).toHaveLength(0);
  });

  it("clears all layers", () => {
    const loader = new PolicyLoader();
    (loader as any).layers = [{ name: "test", source: "test", config: {} }];
    loader.clear();
    expect(loader.getLayers()).toHaveLength(0);
  });

  it("loadAll loads from all sources", () => {
    fs.mkdirSync(tmpDir, { recursive: true });
    fs.writeFileSync(
      path.join(tmpDir, ".pi-guardrails.json"),
      JSON.stringify({ maxStrikes: 10 }),
    );

    const loader = new PolicyLoader();
    loader.loadAll(tmpDir);
    // At minimum the project layer should load
    expect(loader.getLayers().length).toBeGreaterThanOrEqual(1);
  });

  it("handles malformed JSON gracefully", () => {
    fs.mkdirSync(tmpDir, { recursive: true });
    fs.writeFileSync(path.join(tmpDir, ".pi-guardrails.json"), "not json {{{");

    const loader = new PolicyLoader();
    const layer = loader.loadProjectPolicy(tmpDir);
    expect(layer).toBeNull();
  });
});
