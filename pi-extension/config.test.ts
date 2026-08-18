import { describe, it, expect } from "vitest";
import { getAllowListPath } from "./config.js";
import { DEFAULT_CONFIG } from "./types.js";

describe("config", () => {
  it("exposes an allow-list storage path sibling to config.json", () => {
    expect(getAllowListPath().endsWith("allowlist.json")).toBe(true);
    expect(getAllowListPath()).not.toContain("config.json");
  });

  it("defaults allowDanger to enabled with type-back required", () => {
    expect(DEFAULT_CONFIG.allowDanger).toEqual({
      enabled: true,
      requireTypebackForCatastrophic: true,
    });
  });
});
