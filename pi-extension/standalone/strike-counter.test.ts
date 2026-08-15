import { describe, it, expect } from "vitest";
import { StrikeCounter } from "./strike-counter.js";

describe("StrikeCounter", () => {
  it("starts with zero strikes", () => {
    const counter = new StrikeCounter(3);
    const result = counter.getStrikes("task-1");
    expect(result.strikeCount).toBe(0);
    expect(result.maxReached).toBe(false);
  });

  it("increments strikes on consecutive failures", () => {
    const counter = new StrikeCounter(3);
    counter.recordAttempt("task-1", false, "error");
    counter.recordAttempt("task-1", false, "error");
    const result = counter.getStrikes("task-1");
    expect(result.strikeCount).toBe(2);
    expect(result.maxReached).toBe(false);
  });

  it("reaches max strikes", () => {
    const counter = new StrikeCounter(3);
    counter.recordAttempt("task-1", false);
    counter.recordAttempt("task-1", false);
    counter.recordAttempt("task-1", false);
    const result = counter.getStrikes("task-1");
    expect(result.strikeCount).toBe(3);
    expect(result.maxReached).toBe(true);
  });

  it("resets consecutive failures on success", () => {
    const counter = new StrikeCounter(3);
    counter.recordAttempt("task-1", false);
    counter.recordAttempt("task-1", false);
    counter.recordAttempt("task-1", true);
    counter.recordAttempt("task-1", false);
    const result = counter.getStrikes("task-1");
    expect(result.strikeCount).toBe(1);
    expect(result.maxReached).toBe(false);
  });

  it("resets a task explicitly", () => {
    const counter = new StrikeCounter(3);
    counter.recordAttempt("task-1", false);
    const reset = counter.reset("task-1");
    expect(reset).toBe(true);
    const result = counter.getStrikes("task-1");
    expect(result.strikeCount).toBe(0);
  });

  it("returns false when resetting nonexistent task", () => {
    const counter = new StrikeCounter(3);
    expect(counter.reset("nonexistent")).toBe(false);
  });

  it("tracks separate tasks independently", () => {
    const counter = new StrikeCounter(3);
    counter.recordAttempt("task-1", false);
    counter.recordAttempt("task-2", false);
    counter.recordAttempt("task-2", false);
    expect(counter.getStrikes("task-1").strikeCount).toBe(1);
    expect(counter.getStrikes("task-2").strikeCount).toBe(2);
  });

  it("round-trips through JSON serialization", () => {
    const counter = new StrikeCounter(3);
    counter.recordAttempt("task-1", false, "err");
    counter.recordAttempt("task-1", true);
    const json = counter.toJSON();
    const restored = StrikeCounter.fromJSON(json, 3);
    expect(restored.getStrikes("task-1").strikeCount).toBe(0);
    expect(restored.getStrikes("task-1").details).toHaveLength(2);
  });

  it("exposes maxStrikes", () => {
    const counter = new StrikeCounter(5);
    expect(counter.getMaxStrikes()).toBe(5);
  });
});
