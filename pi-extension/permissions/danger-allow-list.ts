import * as fs from "node:fs";
import { getAllowListPath } from "../config.js";
import type { AllowListEntry } from "../types.js";

/** Normalize a command for storage and comparison: trim + collapse whitespace. */
export function normalizeCommand(cmd: string): string {
  return cmd.trim().replace(/\s+/g, " ");
}

export class DangerAllowList {
  private entries: AllowListEntry[] = [];
  private compiledPatterns: Map<string, RegExp> = new Map();
  private storagePath: string;

  constructor(storagePath?: string) {
    this.storagePath = storagePath ?? getAllowListPath();
    this.load();
  }

  matches(cmd: string): AllowListEntry | undefined {
    const normalized = normalizeCommand(cmd);
    for (const entry of this.entries) {
      if (entry.type === "exact" && entry.command === normalized) return entry;
      if (entry.type === "pattern") {
        const re = this.compiledPatterns.get(entry.regex);
        if (re?.test(normalized)) return entry;
      }
    }
    return undefined;
  }

  /** Add an exact-command entry. Returns false if the normalized command is already allow-listed. */
  addExact(command: string, source: "prompt" | "tool", reason?: string): boolean {
    const normalized = normalizeCommand(command);
    if (this.entries.some((e) => e.type === "exact" && e.command === normalized)) return false;
    this.entries.push({ type: "exact", command: normalized, addedAt: new Date().toISOString(), reason, source });
    this.persist();
    return true;
  }

  /** Add a regex pattern entry. Returns false if the regex is invalid or already present. */
  addPattern(regex: string, reason: string): boolean {
    if (this.entries.some((e) => e.type === "pattern" && e.regex === regex)) return false;
    try {
      this.compiledPatterns.set(regex, new RegExp(regex));
    } catch {
      return false;
    }
    this.entries.push({ type: "pattern", regex, addedAt: new Date().toISOString(), reason, source: "tool" });
    this.persist();
    return true;
  }

  /**
   * Remove an entry by exact command (normalized) or pattern regex string.
   * Exact commands are matched after normalization; patterns are identified by
   * their exact stored string (regexes are never normalized, as trimming/altering
   * whitespace would change regex meaning).
   */
  remove(commandOrRegex: string): boolean {
    const normalized = normalizeCommand(commandOrRegex);
    const before = this.entries.length;
    this.entries = this.entries.filter((e) => {
      const hit = (e.type === "exact" && e.command === normalized) || (e.type === "pattern" && e.regex === commandOrRegex); // Patterns matched by exact stored string (never normalized)
      if (hit && e.type === "pattern") this.compiledPatterns.delete(e.regex);
      return !hit;
    });
    if (this.entries.length !== before) {
      this.persist();
      return true;
    }
    return false;
  }

  list(): AllowListEntry[] {
    return [...this.entries];
  }

  clear(): void {
    this.entries = [];
    this.compiledPatterns.clear();
    this.persist();
  }

  private load(): void {
    try {
      if (!fs.existsSync(this.storagePath)) return;
      const raw = JSON.parse(fs.readFileSync(this.storagePath, "utf-8"));
      if (!Array.isArray(raw)) return;
      for (const entry of raw) {
        if (entry?.type === "exact" && typeof entry.command === "string") {
          this.entries.push(entry as AllowListEntry);
        } else if (entry?.type === "pattern" && typeof entry.regex === "string") {
          try {
            this.compiledPatterns.set(entry.regex, new RegExp(entry.regex));
            this.entries.push(entry as AllowListEntry);
          } catch {
            console.warn(`[pi-guardrails] Skipping allow-list entry with invalid regex: ${entry.regex}`);
          }
        }
      }
    } catch {
      // Corrupt or unreadable file: start with an empty list; next persist() recreates it.
    }
  }

  private persist(): void {
    try {
      const dir = this.storagePath.substring(0, this.storagePath.lastIndexOf("/"));
      if (dir && !fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
      fs.writeFileSync(this.storagePath, JSON.stringify(this.entries, null, 2));
    } catch {
      // Best-effort persistence; the in-memory list still functions for this session.
    }
  }
}
