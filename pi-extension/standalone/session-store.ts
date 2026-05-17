import * as fs from "node:fs";
import * as crypto from "node:crypto";
import type { SessionState } from "../types.js";
import { getSessionsDir, ensureDirs } from "../config.js";
import { FileReadStore } from "./file-read-store.js";
import { StrikeCounter } from "./strike-counter.js";
import { ScopeValidator } from "./scope-validator.js";

export class SessionStore {
  private state: SessionState | null = null;
  private filePath: string | null = null;
  fileReadStore = new FileReadStore();
  strikeCounter: StrikeCounter;
  readonly scopeValidator = new ScopeValidator();

  constructor(maxStrikes = 3) {
    this.strikeCounter = new StrikeCounter(maxStrikes);
  }

  initialize(projectSlug: string, scope?: string[]): SessionState {
    ensureDirs();

    const id = `session-${crypto.randomUUID().substring(0, 8)}`;
    this.filePath = `${getSessionsDir()}/${id}.json`;

    this.state = {
      id,
      projectSlug,
      createdAt: new Date().toISOString(),
      scope: { paths: scope ?? [], reason: null },
      filesRead: {},
      strikes: {},
      mcpEndpoint: null,
      mcpConnected: false,
    };

    if (scope && scope.length > 0) {
      this.scopeValidator.setScope(scope);
    }

    this.save();
    return this.state;
  }

  getState(): SessionState | null {
    return this.state;
  }

  getSessionId(): string | null {
    return this.state?.id ?? null;
  }

  isInitialized(): boolean {
    return this.state !== null;
  }

  setMcpConnected(endpoint: string, connected: boolean): void {
    if (this.state) {
      this.state.mcpEndpoint = endpoint;
      this.state.mcpConnected = connected;
      this.save();
    }
  }

  save(): void {
    if (!this.state || !this.filePath) return;

    this.state.filesRead = this.fileReadStore.toJSON();
    this.state.strikes = this.strikeCounter.toJSON();

    try {
      fs.writeFileSync(this.filePath, JSON.stringify(this.state, null, 2));
    } catch {
      // Best-effort persist
    }
  }

  load(sessionId: string): SessionState | null {
    const filePath = `${getSessionsDir()}/${sessionId}.json`;
    try {
      if (!fs.existsSync(filePath)) return null;
      const raw = fs.readFileSync(filePath, "utf-8");
      const parsed = JSON.parse(raw) as SessionState;

      this.state = parsed;
      this.filePath = filePath;
      // Hydrate fileReadStore from persisted data
      if (Object.keys(parsed.filesRead).length > 0) {
        this.fileReadStore = FileReadStore.fromJSON(parsed.filesRead);
      }
      // Hydrate strikeCounter from persisted data
      if (Object.keys(parsed.strikes).length > 0) {
        this.strikeCounter = StrikeCounter.fromJSON(parsed.strikes, this.strikeCounter.getMaxStrikes());
      }
      this.scopeValidator.setScope(parsed.scope.paths, parsed.scope.reason ?? undefined);

      return parsed;
    } catch {
      return null;
    }
  }
}
