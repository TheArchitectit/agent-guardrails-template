import { describe, it, expect } from "vitest";
import { SessionStore } from "./session-store.js";

describe("SessionStore halt lifecycle", () => {
  it("starts with no halt state", () => {
    const store = new SessionStore();
    store.initialize("test-project");
    expect(store.getHaltState()).toBeNull();
    expect(store.isHalted()).toBe(false);
  });

  it("records a halt with reason and severity", () => {
    const store = new SessionStore();
    store.initialize("test-project");
    const halt = store.recordHalt("Dangerous command blocked", "critical");

    expect(halt.status).toBe("halted");
    expect(halt.reason).toBe("Dangerous command blocked");
    expect(halt.severity).toBe("critical");
    expect(halt.haltedAt).toBeTruthy();
    expect(store.isHalted()).toBe(true);
  });

  it("acknowledges a halt", () => {
    const store = new SessionStore();
    store.initialize("test-project");
    store.recordHalt("Test halt", "warning");

    const result = store.acknowledgeHalt("reviewed and approved");
    expect(result).not.toBeNull();
    expect(result!.status).toBe("acknowledged");
    expect(result!.acknowledgedBy).toBe("reviewed and approved");
    expect(result!.acknowledgedAt).toBeTruthy();
    expect(store.isHalted()).toBe(false);
  });

  it("returns null when acknowledging with no active halt", () => {
    const store = new SessionStore();
    store.initialize("test-project");
    const result = store.acknowledgeHalt();
    expect(result).toBeNull();
  });

  it("persists halt state through save/load", () => {
    const store = new SessionStore();
    const state = store.initialize("test-project");
    store.recordHalt("Test halt", "critical");

    const sessionId = state.id;
    const store2 = new SessionStore();
    const loaded = store2.load(sessionId);
    expect(loaded).not.toBeNull();
    expect(loaded!.haltState).toBeDefined();
    expect(loaded!.haltState!.status).toBe("halted");
    expect(loaded!.haltState!.reason).toBe("Test halt");
  });
});
