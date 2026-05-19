import { describe, it, expect } from "vitest";
import { FileReadStore } from "./file-read-store.js";

describe("FileReadStore", () => {
  it("records and verifies a file read", () => {
    const store = new FileReadStore();
    store.record("/foo/bar.ts");
    expect(store.wasRead("/foo/bar.ts")).toBe(true);
    expect(store.wasRead("/foo/baz.ts")).toBe(false);
  });

  it("normalizes paths with resolve", () => {
    const store = new FileReadStore();
    store.record("foo/bar.ts");
    expect(store.wasRead("foo/bar.ts")).toBe(true);
  });

  it("returns readAt timestamp", () => {
    const store = new FileReadStore();
    store.record("/foo/bar.ts");
    const ts = store.getReadAt("/foo/bar.ts");
    expect(ts).not.toBeNull();
    expect(() => new Date(ts!)).not.toThrow();
  });

  it("returns null for unread file", () => {
    const store = new FileReadStore();
    expect(store.getReadAt("/unread.ts")).toBeNull();
  });

  it("clears all reads", () => {
    const store = new FileReadStore();
    store.record("/a.ts");
    store.record("/b.ts");
    store.clear();
    expect(store.wasRead("/a.ts")).toBe(false);
    expect(store.size).toBe(0);
  });

  it("tracks size", () => {
    const store = new FileReadStore();
    expect(store.size).toBe(0);
    store.record("/a.ts");
    expect(store.size).toBe(1);
    store.record("/b.ts");
    expect(store.size).toBe(2);
  });

  it("round-trips through JSON serialization", () => {
    const store = new FileReadStore();
    store.record("/a.ts");
    store.record("/b.ts");
    const json = store.toJSON();
    const restored = FileReadStore.fromJSON(json);
    expect(restored.wasRead("/a.ts")).toBe(true);
    expect(restored.wasRead("/b.ts")).toBe(true);
    expect(restored.wasRead("/c.ts")).toBe(false);
    expect(restored.size).toBe(2);
  });
});
