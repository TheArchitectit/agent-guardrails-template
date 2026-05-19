Now I have a thorough understanding of the design spec, gap analysis, sprint plan, and all reference implementations. Here is my complete review and implementation plan.

---

# Pi Guardrails Extension: Spec Review, Sprint Critique, and Implementation Plans

---

## Part 1: Spec Review -- Issues, Ambiguities, and Gaps

### CRITICAL (blocks implementation)

**1. Peer dependency namespace conflict (design spec line 53, 551-553)**

The design spec states peer dependencies are `@earendil-works/pi-coding-agent` and `@earendil-works/pi-tui`. However, the reference implementation pi-messenger (which is the primary pattern the spec cites) imports from `@mariozechner/pi-coding-agent` and `@mariozechner/pi-tui` (pi-messenger `index.ts:11-12`). Meanwhile, pi-subagents uses `@earendil-works/pi-coding-agent` and `@earendil-works/pi-tui` (subagents `index.ts:18-20`). These are different package namespaces, and the wrong one will cause a build failure. Before implementation starts, the developer must confirm which namespace the current pi runtime actually provides. The package.json `peerDependencies` must match whatever the installed pi runtime exports.

**2. Standalone tool schemas diverge from MCP server schemas (design spec lines 118-222 vs. server.go lines 518-697)**

The standalone tools use different parameter names and structures than the MCP server equivalents. When the MCP bridge proxies calls, it will need to translate between these schemas. The mismatches:

| Standalone tool | Standalone params | MCP server tool | MCP server params |
|---|---|---|---|
| `guardrail_log_violation` | `law`, `severity` (warning/critical), `details` | `guardrail_log_violation` | `rule_id`, `severity` (error/warning/info), `message`, `file_path` |
| `guardrail_record_attempt` | `task`, `success`, `error` | `guardrail_record_attempt` | `session_token`, `task_id`, `error_message`, `error_category` |
| `guardrail_check_strikes` | `task` | `guardrail_validate_three_strikes` | `session_token`, `task_id` |
| `guardrail_verify_read` | `filePath` | `guardrail_verify_file_read` | `session_token`, `file_path`, `expected_content` |
| `guardrail_set_scope` | `paths`, `reason` | `guardrail_validate_scope` | `file_path`, `authorized_scope` |

The MCP bridge section (design spec line 92) says stubs proxy calls to the Go server, but there is no mapping layer defined. The developer needs either: (a) a translation layer in the MCP client that maps standalone params to MCP server params, or (b) the standalone tools should use the same parameter names as the MCP server tools. Option (b) is simpler but the spec explicitly chose different names "to avoid shadowing." This must be resolved.

**3. MCP bridge tool registration is contradictory (design spec lines 520-526 vs. 91-112)**

Line 520: "All MCP bridge tool stubs are registered at extension init." Line 526: "The tool list is read dynamically from the server at connection time." These are contradictory. You cannot register stubs for specific tools at init time AND dynamically discover the tool list at connection time -- either the stubs are hardcoded (known at init) or they are dynamic (discovered at connection). The design spec uses a single `guardrail_mcp` tool with an `action` parameter (line 92), which resolves this: one stub is registered at init, and the action names become available after connection. But the section on line 109 says "Each stub's `execute()` method" (plural), implying multiple stubs. The spec needs to choose one model. The pi-messenger pattern (single tool with action dispatch) is the correct approach.

**4. No `pi-extension/` directory exists in the template repo**

The glob of `/mnt/data/agent-guardrails-template/` shows no `pi-extension/` directory. The design spec's file structure (lines 489-516) describes files inside `pi-extension/` but this directory must be created from scratch. The spec should state whether this directory lives at the template repo root or is a separate package/repo.

### HIGH (would cause bugs or confused users)

**5. `guardrail_standalone_init` naming is misleading (design spec line 78)**

The tool is named `guardrail_standalone_init` but it also initializes MCP connections (design spec lines 235-244, and the MCP bridge behavior says "run guardrail_standalone_init to retry connection"). A tool named "standalone_init" that also connects to MCP is confusing. Consider `guardrail_init_session` (matching the MCP server's `guardrail_init_session`) or `guardrail_init`.

**6. Session ID generation is undefined (design spec line 441)**

The session file structure includes `id: "session-abc123"` but the spec does not define how this ID is generated. Is it a UUID, a timestamp-based slug, a hash of the project path? The session lifecycle handler (line 235) calls `sessionStore.initialize(ctx.cwd)` but `initialize` is not specified. The developer needs to know the ID format for file naming (sessions/{sessionId}.json).

**7. `production-first.ts` is in the file structure but has no enforcement logic defined (design spec line 34, 501)**

The file structure lists `standalone/production-first.ts` but the event handlers section has no handler that calls it, no tool exposes it, and no schema is defined for it. The spec should either define its interface or remove it from the initial file structure.

**8. `haltChecker.checkCommand()` is underspecified (design spec lines 309-328)**

The bash safety handler calls `haltChecker.checkCommand(cmd)` which returns `{ shouldHalt, reason }`. But the halt checker is also supposed to evaluate general halt conditions (line 34). Is `checkCommand` a bash-specific method or a general method? The `guardrail_check_halt` tool schema (line 187) takes `operation`, `filePath`, `details` -- this is a different interface than `checkCommand`. The spec conflates bash command checking with general halt evaluation.

**9. Status bar string format may not render correctly (design spec lines 365-371)**

The proposed format `[g: !!2/3 src/ !3v mcp:*]` uses square brackets and special characters. The pi-messenger status bar (line 286) uses a simpler format: `msg: ${nameStr}${countStr}...`. The square brackets and exclamation marks may conflict with terminal rendering or pi's status bar parsing. This needs testing against the actual pi status bar implementation.

**10. No `renderCall` or `renderResult` defined for any tool (design spec missing)**

Both pi-messenger and pi-subagents define `renderCall` and `renderResult` methods on their tools (subagents `index.ts:438-471`). These control how the tool call and result appear in the TUI. The design spec's 11 tools have no render methods defined. Without them, pi will use default rendering which will be ugly for structured tool output like strike counts and violation logs.

### MEDIUM (should be clarified before implementation)

**11. TypeBox import source is ambiguous (design spec line 116, 553)**

The spec says TypeBox is "(via pi-coding-agent re-export)" at line 553, but pi-messenger imports it directly from `@sinclair/typebox` (line 14). The developer needs to know whether to import from `@sinclair/typebox` directly (like pi-messenger) or from the re-export.

**12. `ctx.hasUI` availability on `session_start` (design spec line 241)**

The session_start handler checks `ctx.hasUI` before calling `ctx.ui.setStatus()`. The question is whether `ctx.hasUI` is always available on `session_start` or only in interactive sessions. If a pi agent runs headless, `ctx.hasUI` may be false and the status bar code path is dead. This should be documented.

**13. No spec for `install.mjs` behavior (design spec line 489)**

The file structure lists `install.mjs` but the spec provides no details about its behavior. The pi-messenger installer (`install.mjs` lines 1-162) copies the npm package to `~/.pi/agent/extensions/pi-messenger/`. The guardrails installer would need to copy to `~/.pi/agent/extensions/pi-guardrails/`. The spec should specify the exact install target directory and any special behavior.

**14. No `promptSnippet` defined for any tool**

Both reference extensions define `promptSnippet` on their tools (pi-messenger line 396). This is the short text that appears in the agent's tool suggestion UI. The design spec's 11 standalone tools have no `promptSnippet` values, which means the agent will see the full `description` text in suggestions. This degrades the agent UX.

**15. Event handler conflict resolution is underspecified (design spec lines 336-338)**

The spec says "whichever handler returns `{ block: true }` first stops the tool call" but then says "the pi runtime processes all handlers." These are contradictory: if the first blocker stops the call, later handlers don't run. If all handlers run, then the runtime must merge multiple block reasons. The pi-messenger pattern (line 1110) shows a single handler that checks one condition. The spec should clarify whether multiple `tool_call` handlers run in parallel or sequentially, and what happens when more than one returns `{ block: true }`.

---

## Part 2: Sprint Plan Critique

### Sprint 1A Issues (Template Repo)

**Item 1: Bash Command Classification Engine -- Too large for "S (2 days)"**

This item requires: (1) a classification engine with 5 categories, (2) glob/regex matching, (3) configurable overrides at 3 levels, (4) a JSON command mapping file, (5) reference implementations for both Claude Code and pi, and (6) tests. That is at least 4 days of work for a single developer. The effort estimate should be M (4 days). Additionally, "reference implementations for each agent" is vague -- what does a reference implementation look like? A wrapper function? A config file? An adapter class?

**Item 2: Scope Patterns Library -- Correct size but underspecified**

One day is reasonable for this, but the spec says "patterns are composable" without defining the composition API. `src_only + no_lockfiles` -- is that a union, an intersection, a chain? The developer needs to know the composition semantics.

**Item 3: Pre-commit Hook -- Missing critical detail**

The pre-commit hook calls out scope pattern validation (Item 2) but also does secret scanning with regex patterns. The secret regex patterns listed are Python-style regex but the hook is a `.sh` file. Shell scripts cannot run regex like `(?i)(api_key|apikey|api-key)...\s*[:=]...` natively. This hook either needs to call a Node.js script (which means `node` must be available in the git hook environment) or use a simpler approach like `grep -E`. The spec should define the runtime requirement.

**Item 4: GitHub Actions -- Should be deferred**

This is the only CI/CD item and it depends on Items 1 and 2. It adds 3 days of effort but does not help the core pi extension ship. The template repo's existing GitHub workflows (`.github/workflows/`) already exist for other purposes. This item should move to Sprint 2 and not block the pi extension.

**Sprint 1A ordering problem:** Item 3 depends on Item 2, and Item 4 depends on Items 1-2. But Items 1 and 2 are independent and should be developed in parallel. The sprint plan does not call this out.

### Sprint 1B Issues (Pi Extension)

**The fundamental problem: Sprint 1B does not implement the core pi extension at all.**

Sprint 1B lists Items 5-9, which are all *new features* from the gap analysis. But the design spec (lines 1-590) describes a completely different thing: the core pi extension with 11 standalone tools, MCP bridge, TUI panel, status bar, skills, and event handlers. None of that core extension is in the sprint plan. The sprint plan skips the entire core extension and jumps to gap-closure features.

**This is the single biggest problem with the sprint plan.** The core extension must ship first, because:
1. Without it, there is no pi guardrails extension at all
2. Gap-closure features (injection defense, output validation) are layered *on top of* the core extension
3. The user cannot test anything until the core extension exists
4. Items 5-9 reference files like `pi-extension/injection/` and `pi-extension/tool-permissions/` but the `pi-extension/` directory does not exist yet

**Item 5: Prompt Injection Defense -- Too large for one sprint item**

This is 4 days of effort and includes: (1) pattern-based detection, (2) heuristic scoring, (3) canary token management, and (4) all with confidence thresholds and enforcement modes. The canary token system alone is a separate feature. Split into: 5a (pattern matching + heuristic scoring) and 5b (canary tokens, which the gap analysis itself marks as deferred in the "What's NOT in this Sprint" section -- but then Item 5 includes canary.ts). This is internally contradictory.

**Item 6: Output Validation -- Missing integration point**

This item defines an output validator that scans agent responses, but the design spec's architecture has no output rail. The spec only defines input-side enforcement (tool_call blockers). Where does the output validator hook in? Is it a `tool_result` handler? A new event type? The pi extension API table (design spec lines 55-66) shows `pi.on("tool_result", handler)` -- this could work, but the handler signature on tool_result returns void (it is a side-effect handler, not a blocking handler). You cannot block a tool result from reaching the agent once it has been produced. This needs a different mechanism.

**Item 7: Per-Tool Permission System -- Missing pi API mapping**

The permission levels (auto, ask, blocked) require pi to support "ask the user before executing" semantics. But pi's `tool_call` handler only supports `{ block: true, reason }` -- there is no "ask" return type. To implement "ask" level, the extension would need to either: (a) block the call and tell the agent to ask the user (via a message), or (b) use a pi API that does not exist yet. The spec should define which approach to use.

**Item 8: Team Policy Configuration -- Should be Sprint 2+**

This is marked P2 in the gap analysis priority table and has no blocker in Sprint 1B. It requires an organization-level config hierarchy, RBAC enforcement, and audit logging. This is enterprise architecture, not a first-release feature. Move it to Sprint 2.

**Item 9: Enhanced Bash Safety -- References nonexistent file**

The spec says "Update `pi-extension/standalone/bash-safety.ts`" but this file does not exist. The core extension must be built first, including the halt checker that contains the bash safety logic. This item should be reframed as "enhance the bash safety logic within the core extension's halt-checker" rather than "update an existing file."

### Missing Items (Should Be Added)

**MISSING: Core Pi Extension implementation (the entire design spec)**

The sprint plan completely omits the core extension that the design spec describes. This is the 11-tool, MCP-bridge, TUI-panel, event-handler extension. It must be Sprint 0 or Sprint 1A. Without it, nothing else is buildable or testable.

**MISSING: End-to-end integration test**

Neither sprint defines an integration test that runs the full extension in a real pi session. The design spec mentions "Integration test: create full extension session, run tool calls, verify state persistence" (line 558) but the sprint plan has no item for this.

**MISSING: npm package publish workflow**

The design spec marks NPM publishing as out of scope (line 578) but the success criteria (line 583) says "pi install npm:@thearchitectit/pi-guardrails installs and registers the extension." The package must be published to npm for this criteria to be met. There should be a publish step.

---

## Part 3: Implementation Sprint Plans

### Restructured Sprint Order

The sprint plan needs reorganization. The core extension must ship first. Here is the corrected order:

- **Sprint 0 (1 week): Core Pi Extension** -- the 11 tools, event handlers, session store, status bar, install script
- **Sprint 1 (1 week): TUI Panel, Skills, MCP Bridge** -- the overlay dashboard, skill files, MCP client
- **Sprint 2 (1 week): Gap Closure (partial)** -- bash classification, injection defense, output validation

---

### Sprint 0: Core Pi Extension

#### Item 0.1: Project Scaffold and package.json

**Files to create:**
- `/mnt/data/agent-guardrails-template/pi-extension/package.json`
- `/mnt/data/agent-guardrails-template/pi-extension/tsconfig.json` (if needed by pi runtime)
- `/mnt/data/agent-guardrails-template/pi-extension/.gitignore`

**What package.json should contain:**

```json
{
  "name": "@thearchitectit/pi-guardrails",
  "version": "0.1.0",
  "description": "Four Laws guardrails enforcement for pi coding agent",
  "type": "module",
  "author": "TheArchitectit",
  "license": "MIT",
  "bin": {
    "pi-guardrails": "install.mjs"
  },
  "files": [
    "*.ts",
    "*.mjs",
    "standalone/**",
    "mcp-bridge/**",
    "tui/**",
    "skills/**",
    "README.md"
  ],
  "pi": {
    "extensions": ["./index.ts"],
    "skills": ["./skills"]
  },
  "peerDependencies": {
    "@earendil-works/pi-coding-agent": "*",
    "@earendil-works/pi-tui": "*"
  },
  "peerDependenciesMeta": {
    "@earendil-works/pi-coding-agent": { "optional": true },
    "@earendil-works/pi-tui": { "optional": true }
  },
  "devDependencies": {
    "vitest": "^2.1.8"
  },
  "scripts": {
    "test": "vitest run",
    "test:watch": "vitest"
  }
}
```

**Critical decision before starting:** Confirm the peer dependency namespace. Check which namespace the installed pi runtime provides by examining `pi-messenger` (uses `@mariozechner`) vs `pi-subagents` (uses `@earendil-works`). The `peerDependencies` above uses `@earendil-works` based on pi-subagents, which appears to be the newer package. Verify by checking `ls ~/.pi/agent/node_modules/` for which namespace is actually installed.

**Step-by-step:**
1. Create the `pi-extension/` directory at the template repo root
2. Create `package.json` with the structure above
3. Create `.gitignore` with `node_modules/` and `*.js` (pi runs .ts directly)
4. Verify the peer dependency namespace by checking the installed pi runtime

**Test strategy:** After creating package.json, run `npm install` in the pi-extension directory to verify peer dependencies resolve correctly. No unit tests for this item.

**Integration points:** Every other item depends on this scaffold.

---

#### Item 0.2: Types and Config

**Files to create:**
- `/mnt/data/agent-guardrails-template/pi-extension/types.ts`
- `/mnt/data/agent-guardrails-template/pi-extension/config.ts`

**types.ts should contain:**

Key exported types:
- `FileReadStore`: interface with `record(filePath: string): void`, `wasRead(filePath: string): boolean`, `getReadAt(filePath: string): string | null`, `toJSON(): Record<string, string>`
- `StrikeCounter`: interface with `recordAttempt(task: string, success: boolean, error?: string): { strikeCount: number; maxReached: boolean }`, `getStrikes(task: string): { strikeCount: number; maxReached: boolean; details: Attempt[] }`, `reset(task: string): boolean`
- `ScopeValidator`: interface with `setScope(paths: string[], reason?: string): void`, `isInScope(filePath: string, operation: 'read' | 'edit' | 'delete'): boolean`, `getScope(): string[]`
- `HaltChecker`: interface with `checkCommand(cmd: string): { shouldHalt: boolean; reason?: string }`, `checkHalt(operation: string, filePath?: string, details?: string): HaltResult`
- `HaltResult`: `{ shouldHalt: boolean; reasons: string[]; severity: 'none' | 'warning' | 'critical'; suggestions: string[] }`
- `SessionState`: the JSON structure from design spec lines 441-463
- `Attempt`: `{ success: boolean; error?: string; timestamp: string }`
- `Violation`: `{ id: string; law: string; severity: string; details: string; filePath?: string; operation?: string; timestamp: string }`
- `GuardrailsConfig`: the config structure from design spec lines 467-478

Also export TypeBox schema definitions for each of the 11 tool parameter types and return types. Follow pi-messenger's pattern: import `{ Type }` from `@sinclair/typebox` directly (confirmed by pi-messenger line 14). Define a `StringEnum` helper function (same as pi-messenger lines 16-26).

Schemas for each tool (using the parameter names from the design spec, lines 118-222):
- `InitSessionSchema`: `Type.Object({ projectSlug: Type.String(), agentType: Type.Optional(Type.String()), scope: Type.Optional(Type.Array(Type.String())), rules: Type.Optional(Type.Array(Type.String())) })`
- `RecordReadSchema`: `Type.Object({ filePath: Type.String() })`
- `VerifyReadSchema`: `Type.Object({ filePath: Type.String() })`
- `SetScopeSchema`: `Type.Object({ paths: Type.Array(Type.String()), reason: Type.Optional(Type.String()) })`
- `CheckScopeSchema`: `Type.Object({ filePath: Type.String(), operation: Type.Union([Type.Literal("read"), Type.Literal("edit"), Type.Literal("delete")]) })`
- `RecordAttemptSchema`: `Type.Object({ task: Type.String(), success: Type.Boolean(), error: Type.Optional(Type.String()) })`
- `CheckStrikesSchema`: `Type.Object({ task: Type.String() })`
- `ResetStrikesSchema`: `Type.Object({ task: Type.String() })`
- `CheckHaltSchema`: `Type.Object({ operation: Type.String(), filePath: Type.Optional(Type.String()), details: Type.Optional(Type.String()) })`
- `LogViolationSchema`: `Type.Object({ law: StringEnum(["read-before-edit", "stay-in-scope", "verify-before-commit", "halt-when-uncertain"]), severity: StringEnum(["warning", "critical"]), details: Type.String(), filePath: Type.Optional(Type.String()), operation: Type.Optional(Type.String()) })`
- `StatusSchema`: `Type.Object({})`

**config.ts should contain:**

- `loadConfig(cwd: string): GuardrailsConfig` -- reads from `~/.pi/agent/extensions/pi-guardrails/config.json`, falls back to defaults
- `DEFAULT_CONFIG: GuardrailsConfig` -- the default values from design spec lines 467-478
- `MCP_API_KEY`: read from `process.env.PI_GUARDRAILS_MCP_API_KEY` (design spec line 481)
- `getStorageDir(): string` -- returns `~/.pi/agent/extensions/pi-guardrails/`
- `getSessionsDir(): string` -- returns `~/.pi/agent/extensions/pi-guardrails/sessions/`
- `getViolationsLogPath(): string` -- returns `~/.pi/agent/extensions/pi-guardrails/violations.jsonl`

**Step-by-step:**
1. Write types.ts with all interfaces and TypeBox schemas
2. Write config.ts with loadConfig and path helpers
3. Import `Type` from `@sinclair/typebox` directly (not from re-export)
4. Write a test file `types.test.ts` that validates each schema against valid and invalid inputs

**Test strategy:**
- Unit test each TypeBox schema: pass valid input and verify it parses, pass invalid input and verify it rejects
- Unit test `loadConfig`: test with no config file (returns defaults), test with valid config file, test with malformed config file (returns defaults + logs warning)

**Integration points:** Every module in the extension imports from types.ts. Config.ts is used by session-store.ts and index.ts.

---

#### Item 0.3: Standalone Core Modules

**Files to create:**
- `/mnt/data/agent-guardrails-template/pi-extension/standalone/file-read-store.ts`
- `/mnt/data/agent-guardrails-template/pi-extension/standalone/file-read-store.test.ts`
- `/mnt/data/agent-guardrails-template/pi-extension/standalone/strike-counter.ts`
- `/mnt/data/agent-guardrails-template/pi-extension/standalone/strike-counter.test.ts`
- `/mnt/data/agent-guardrails-template/pi-extension/standalone/scope-validator.ts`
- `/mnt/data/agent-guardrails-template/pi-extension/standalone/scope-validator.test.ts`
- `/mnt/data/agent-guardrails-template/pi-extension/standalone/halt-checker.ts`
- `/mnt/data/agent-guardrails-template/pi-extension/standalone/halt-checker.test.ts`
- `/mnt/data/agent-guardrails-template/pi-extension/standalone/violation-log.ts`
- `/mnt/data/agent-guardrails-template/pi-extension/standalone/violation-log.test.ts`
- `/mnt/data/agent-guardrails-template/pi-extension/standalone/session-store.ts`
- `/mnt/data/agent-guardrails-template/pi-extension/standalone/session-store.test.ts`

**Skip for now:** `production-first.ts` (design spec lists it but defines no interface for it). Add a stub file with a TODO comment so the directory structure matches the design spec, but do not implement logic.

**file-read-store.ts:**

```typescript
// Key exports:
export class FileReadStore {
  private reads: Map<string, string>; // filePath -> ISO timestamp

  record(filePath: string): void;
  wasRead(filePath: string): boolean;
  getReadAt(filePath: string): string | null;
  clear(): void;
  toJSON(): Record<string, string>;
  static fromJSON(data: Record<string, string>): FileReadStore;
}
```

Implementation details:
- `record()` normalizes the path using `path.resolve()` before storing
- `wasRead()` also normalizes before checking (handles relative vs. absolute path differences)
- Timestamp is `new Date().toISOString()`

**strike-counter.ts:**

```typescript
export interface Attempt {
  success: boolean;
  error?: string;
  timestamp: string;
}

export class StrikeCounter {
  private strikes: Map<string, Attempt[]>; // task -> attempts
  private maxStrikes: number;

  constructor(maxStrikes?: number); // default from config (3)
  recordAttempt(task: string, success: boolean, error?: string): { strikeCount: number; maxReached: boolean };
  getStrikes(task: string): { strikeCount: number; maxReached: boolean; details: Attempt[] };
  reset(task: string): boolean;
  getAllStrikes(): Map<string, Attempt[]>;
  toJSON(): Record<string, { attempts: Attempt[] }>;
  static fromJSON(data: Record<string, { attempts: Attempt[] }>, maxStrikes: number): StrikeCounter;
}
```

Implementation details:
- `strikeCount` counts consecutive failed attempts (resets on success)
- `maxReached` is `strikeCount >= maxStrikes`
- Even successful attempts are recorded in `details` for audit

**scope-validator.ts:**

```typescript
export class ScopeValidator {
  private paths: string[];
  private reason: string | null;

  setScope(paths: string[], reason?: string): void;
  isInScope(filePath: string, operation: 'read' | 'edit' | 'delete'): boolean;
  getScope(): string[];
  getReason(): string | null;
  toJSON(): { paths: string[]; reason: string | null };
  static fromJSON(data: { paths: string[]; reason?: string }): ScopeValidator;
}
```

Implementation details:
- If `paths` is empty (no scope set), all paths are in scope (default permissive)
- Matching is prefix-based: `src/auth/login.ts` is in scope if `src/` is an authorized path
- Path normalization: strip trailing slashes from scope paths, resolve the file path before checking
- `operation` parameter is reserved for future use (all operations treated the same for now)

**halt-checker.ts:**

```typescript
export interface CommandCheckResult {
  shouldHalt: boolean;
  reason?: string;
  category?: 'destructive' | 'elevated' | 'network';
}

export class HaltChecker {
  private blockedPatterns: RegExp[];
  private destructiveCommands: string[];

  constructor();
  checkCommand(cmd: string): CommandCheckResult;
  checkHalt(operation: string, filePath?: string, details?: string): HaltResult;
}
```

Implementation details for `checkCommand`:
- Hardcoded denylist for Sprint 0 (this is replaced by the classification engine in Sprint 2):
  - `rm -rf /` -> block (destructive)
  - `sudo` -> block (elevated) -- excluding `sudo -l` (read-only)
  - `chmod 777` -> block (elevated)
  - `git push --force main` / `git push --force master` -> block (destructive)
  - `git reset --hard` -> block (destructive)
  - `curl.*|.*sh` (pipe to shell) -> block (injection risk)
- Return `{ shouldHalt: false }` for anything not matched

Implementation details for `checkHalt`:
- Check strike counter: if any task has reached max strikes, halt with "critical"
- Check scope: if scope is set and a filePath is provided and out of scope, halt with "warning"
- Return `{ shouldHalt: false, reasons: [], severity: 'none', suggestions: [] }` if no issues

**violation-log.ts:**

```typescript
export class ViolationLog {
  private logPath: string;
  private violations: Violation[];

  constructor(logPath: string);
  log(violation: Omit<Violation, 'id' | 'timestamp'>): Violation;
  getViolations(): Violation[];
  getSummary(): { total: number; critical: number; warning: number };
  flush(): void; // write to disk
  static load(logPath: string): ViolationLog;
}
```

Implementation details:
- `id` is generated as `v-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
- `flush()` writes all violations as JSONL to `logPath`
- `load()` reads existing JSONL file on startup
- Use append-only writes: `fs.appendFileSync(logPath, JSON.stringify(violation) + '\n')` on each `log()` call, not just on `flush()` -- this minimizes data loss if the process crashes

**session-store.ts:**

```typescript
export class SessionStore {
  private sessionsDir: string;
  private currentSessionId: string | null;
  private state: SessionState | null;

  initialize(cwd: string): SessionState;
  getCurrentSession(): SessionState | null;
  getSessionId(): string | null;
  updateSession(updater: (state: SessionState) => void): void;
  save(): void; // persist to disk
  load(sessionId: string): SessionState | null;
}
```

Implementation details:
- `initialize()` creates a new session with `id: session-${Date.now()}-${randomSuffix}`, or loads an existing session for the current project if one exists and is less than 24 hours old
- Session files are stored in `~/.pi/agent/extensions/pi-guardrails/sessions/{sessionId}.json`
- `save()` is called automatically by `updateSession()` (synchronous write)
- Handle corrupted session files by recreating from defaults (design spec line 540)
- Cleanup: delete session files older than 7 days on `initialize()`

**Step-by-step implementation order:**
1. file-read-store.ts + test (simplest module, no dependencies)
2. scope-validator.ts + test (depends on nothing)
3. strike-counter.ts + test (depends on nothing)
4. halt-checker.ts + test (depends on nothing for now; will depend on strike-counter later)
5. violation-log.ts + test (depends on types.ts for Violation)
6. session-store.ts + test (depends on all others for SessionState composition)

**Test strategy:**
- Unit test each class in isolation
- file-read-store: test record, wasRead, wasRead with relative/absolute paths, clear, serialization round-trip
- strike-counter: test recordAttempt success/failure, getStrikes after failures, reset, maxReached threshold, serialization round-trip
- scope-validator: test setScope, isInScope with matching/non-matching paths, empty scope (permissive), serialization round-trip
- halt-checker: test checkCommand with blocked commands, safe commands, piped commands; test checkHalt with/without active strikes
- violation-log: test log, getSummary, flush + reload, append-only behavior
- session-store: test initialize, updateSession, save + load, corrupted file recovery

**Integration points:** These modules are the foundation for the event handlers and tool implementations. The session-store composes the others.

---

#### Item 0.4: Tool Implementations

**Files to create:**
- `/mnt/data/agent-guardrails-template/pi-extension/tools.ts`
- `/mnt/data/agent-guardrails-template/pi-extension/tools.test.ts`

**tools.ts should contain:**

Exported functions that implement the `execute` logic for each of the 11 standalone tools. These are **not** registered here -- registration happens in `index.ts`. This file provides the pure business logic.

Function signatures:

```typescript
export function initSession(
  sessionStore: SessionStore,
  mcpClient: MCPClient | null,
  params: { projectSlug: string; agentType?: string; scope?: string[]; rules?: string[] }
): { sessionId: string; mode: 'standalone' | 'mcp-bridge'; availableTools: string[]; mcpConnected: boolean };

export function recordRead(
  fileReadStore: FileReadStore,
  params: { filePath: string }
): { recorded: boolean; filePath: string; readAt: string };

export function verifyRead(
  fileReadStore: FileReadStore,
  params: { filePath: string }
): { wasRead: boolean; filePath: string; readAt?: string };

export function setScope(
  scopeValidator: ScopeValidator,
  params: { paths: string[]; reason?: string }
): { scope: string[]; reservedAt: string };

export function checkScope(
  scopeValidator: ScopeValidator,
  params: { filePath: string; operation: 'read' | 'edit' | 'delete' }
): { inScope: boolean; reason?: string };

export function recordAttempt(
  strikeCounter: StrikeCounter,
  params: { task: string; success: boolean; error?: string }
): { task: string; strikeCount: number; maxReached: boolean };

export function checkStrikes(
  strikeCounter: StrikeCounter,
  params: { task: string }
): { task: string; strikeCount: number; maxReached: boolean; details: Attempt[] };

export function resetStrikes(
  strikeCounter: StrikeCounter,
  params: { task: string }
): { task: string; reset: boolean };

export function checkHalt(
  haltChecker: HaltChecker,
  params: { operation: string; filePath?: string; details?: string }
): HaltResult;

export function logViolation(
  violationLog: ViolationLog,
  params: { law: string; severity: string; details: string; filePath?: string; operation?: string }
): { logged: boolean; violationId: string; timestamp: string };

export function getStatus(
  sessionStore: SessionStore,
  fileReadStore: FileReadStore,
  strikeCounter: StrikeCounter,
  scopeValidator: ScopeValidator,
  violationLog: ViolationLog,
  mcpClient: MCPClient | null
): SessionStatusResult;
```

Each function takes the relevant stores as explicit parameters rather than accessing globals. This makes them testable in isolation.

**Step-by-step:**
1. Implement each function in the order listed above (simpler ones first)
2. `initSession` is the most complex: creates or loads a session, attempts MCP connection if configured
3. `getStatus` composes state from all stores -- it is a read-only aggregation
4. `recordRead` and `verifyRead` are thin wrappers around FileReadStore
5. `checkHalt` delegates to HaltChecker.checkHalt()

**Test strategy:**
- Unit test each function with mock store instances
- initSession: test fresh session creation, session reuse, MCP disabled/enabled
- recordRead/verifyRead: test the round-trip (record then verify)
- checkScope: test in-scope, out-of-scope, no-scope-set
- recordAttempt/checkStrikes: test strike increment, max reached
- checkHalt: test with no issues, with active strikes, with scope violation
- logViolation: test logging and ID generation
- getStatus: test with empty state and with populated state

**Integration points:** `index.ts` calls these functions from tool execute handlers. The MCP bridge (Item 1.3) will also call some of them as fallbacks.

---

#### Item 0.5: Event Handlers

**Files to create:**
- `/mnt/data/agent-guardrails-template/pi-extension/handlers.ts`
- `/mnt/data/agent-guardrails-template/pi-extension/handlers.test.ts`

**handlers.ts should contain:**

Factory functions that return event handler closures, capturing the stores via closure. This keeps them separate from the tool registration logic and testable.

```typescript
export function createSessionStartHandler(deps: HandlerDeps): (event: any, ctx: ExtensionContext) => Promise<void>;
export function createSessionShutdownHandler(deps: HandlerDeps): () => Promise<void>;
export function createReadTrackingHandler(deps: HandlerDeps): (event: any, ctx: ExtensionContext) => void;
export function createPreEditHandler(deps: HandlerDeps): (event: any, ctx: ExtensionContext) => { block: true; reason: string } | void;
export function createBashSafetyHandler(deps: HandlerDeps): (event: any, ctx: ExtensionContext) => Promise<{ block: true; reason: string } | void>;

interface HandlerDeps {
  sessionStore: SessionStore;
  fileReadStore: FileReadStore;
  scopeValidator: ScopeValidator;
  strikeCounter: StrikeCounter;
  haltChecker: HaltChecker;
  violationLog: ViolationLog;
  mcpClient: MCPClient | null;
  config: GuardrailsConfig;
}
```

Implementation details for each handler:

**createSessionStartHandler:**
- `sessionStore.initialize(ctx.cwd)`
- `mcpClient.tryConnect(config.mcpEndpoint).catch(() => {})` (fire-and-forget, non-blocking)
- `ctx.ui.setStatus("guardrails", "[g: ok]")` if `config.statusBarEnabled && ctx.hasUI`

**createSessionShutdownHandler:**
- `violationLog.flush()`
- `await mcpClient.close()`
- `sessionStore.save()`

**createReadTrackingHandler:**
- Check `event.toolName === "read"`
- Extract `input.path` as string (same pattern as pi-messenger line 714: `const path = input.path as string`)
- `fileReadStore.record(filePath)`

**createPreEditHandler:**
- Check `event.toolName` is `"edit"` or `"write"` (same as pi-messenger line 1111)
- Extract `input.path` using typeof check (same as pi-messenger line 1114: `typeof input.path === "string"`)
- Check Law 1: `if (!fileReadStore.wasRead(filePath))` -> block with violation message, also call `violationLog.log()`
- Check Law 2: `if (!scopeValidator.isInScope(filePath, "edit"))` -> block with violation message, also call `violationLog.log()`
- Return type matches pi-messenger pattern: `{ block: true, reason: "..." }` (line 1131)

**createBashSafetyHandler:**
- Check `event.toolName === "bash"`
- Extract `input.command` as string
- `haltChecker.checkCommand(cmd)` -- if `shouldHalt`, return `{ block: true, reason }`, also call `violationLog.log()`
- This handler is async (for future MCP validation) but the standalone check is synchronous

**Step-by-step:**
1. Define the `HandlerDeps` interface
2. Implement `createReadTrackingHandler` (simplest, no blocking)
3. Implement `createPreEditHandler` (blocking, Law 1 + Law 2)
4. Implement `createBashSafetyHandler` (blocking, command denylist)
5. Implement `createSessionStartHandler` and `createSessionShutdownHandler`

**Test strategy:**
- Create mock `HandlerDeps` with controlled store instances
- ReadTracking: emit a fake `tool_result` event with `toolName: "read"` and verify `fileReadStore.record()` was called
- PreEdit: emit fake `tool_call` with `toolName: "edit"`, test both block cases (unread file, out of scope) and the pass case (read + in scope)
- PreEdit: verify that violationLog.log() is called when a block occurs
- BashSafety: emit fake `tool_call` with `toolName: "bash"`, test dangerous commands and safe commands
- Session lifecycle: test initialize and cleanup

**Integration points:** index.ts uses these factories to register handlers with `pi.on()`. The preEdit handler is the most critical path -- it enforces Laws 1 and 2.

---

#### Item 0.6: Extension Entry Point (index.ts)

**Files to create:**
- `/mnt/data/agent-guardrails-template/pi-extension/index.ts`

**What index.ts should contain:**

This is the main entry point, following the pi-messenger pattern (line 74: `export default function piMessengerExtension(pi: ExtensionAPI)`). All state is instantiated here. Tools are registered. Event handlers are wired up.

Structure:

```typescript
import { homedir } from "node:os";
import * as fs from "node:fs";
import { join } from "node:path";
import type { ExtensionAPI, ExtensionContext, ToolDefinition } from "@earendil-works/pi-coding-agent";  // VERIFY NAMESPACE
import { Type } from "@sinclair/typebox";
import { loadConfig, getStorageDir, getSessionsDir, getViolationsLogPath } from "./config.js";
import { FileReadStore } from "./standalone/file-read-store.js";
import { StrikeCounter } from "./standalone/strike-counter.js";
import { ScopeValidator } from "./standalone/scope-validator.js";
import { HaltChecker } from "./standalone/halt-checker.js";
import { ViolationLog } from "./standalone/violation-log.js";
import { SessionStore } from "./standalone/session-store.js";
import { initSession, recordRead, verifyRead, setScope, checkScope, recordAttempt, checkStrikes, resetStrikes, checkHalt, logViolation, getStatus } from "./tools.js";
import { createSessionStartHandler, createSessionShutdownHandler, createReadTrackingHandler, createPreEditHandler, createBashSafetyHandler, type HandlerDeps } from "./handlers.js";
// Schemas from types.ts
import { InitSessionSchema, RecordReadSchema, /* ... */ } from "./types.js";

export default function piGuardrailsExtension(pi: ExtensionAPI) {
  // ===========================================================================
  // State initialization
  // ===========================================================================
  const config = loadConfig(process.cwd());

  // Ensure storage directories exist
  fs.mkdirSync(getSessionsDir(), { recursive: true });

  // Instantiate core modules
  const fileReadStore = new FileReadStore();
  const strikeCounter = new StrikeCounter(config.maxStrikes);
  const scopeValidator = new ScopeValidator();
  const haltChecker = new HaltChecker();
  const violationLog = ViolationLog.load(getViolationsLogPath());
  const sessionStore = new SessionStore(getSessionsDir());
  // MCP client placeholder (null for Sprint 0, implemented in Sprint 1)
  const mcpClient: any = null;

  // Handler dependencies
  const deps: HandlerDeps = {
    sessionStore, fileReadStore, scopeValidator, strikeCounter,
    haltChecker, violationLog, mcpClient, config
  };

  // ===========================================================================
  // Tool Registration
  // ===========================================================================

  pi.registerTool({
    name: "guardrail_standalone_init",
    label: "Guardrails Init",
    description: "Initialize a guardrails session. Sets up scope, strike tracking, and file read enforcement.",
    promptSnippet: "Initialize guardrails session for the project",
    parameters: InitSessionSchema,
    execute(_id: string, params: any, _signal: any, _onUpdate: any, _ctx: any) {
      return initSession(sessionStore, mcpClient, params);
    }
  });

  pi.registerTool({
    name: "guardrail_record_read",
    label: "Record File Read",
    description: "Mark a file as having been read by the agent. Required before editing (Law 1).",
    promptSnippet: "Record that a file was read",
    parameters: RecordReadSchema,
    execute(_id: string, params: any) { return recordRead(fileReadStore, params); }
  });

  // ... (register all 11 tools following the same pattern)

  // For each tool, set name/label/description/promptSnippet/parameters/execute
  // renderCall and renderResult can be omitted for Sprint 0, added later

  // ===========================================================================
  // Event Handlers
  // ===========================================================================

  pi.on("session_start", createSessionStartHandler(deps));
  pi.on("session_shutdown", createSessionShutdownHandler(deps));
  pi.on("tool_result", createReadTrackingHandler(deps));
  pi.on("tool_call", createPreEditHandler(deps));
  pi.on("tool_call", createBashSafetyHandler(deps));

  // ===========================================================================
  // Slash Command (placeholder for Sprint 1)
  // ===========================================================================
  // pi.registerCommand("guardrails", { ... }); -- implemented in Sprint 1 with TUI panel
}
```

**Key decisions for index.ts:**
- All state is local to the extension function (not global), matching pi-messenger's pattern
- Tools are registered one at a time with `pi.registerTool()` (not as a batch)
- Each tool's `execute` is a thin wrapper that delegates to the tools.ts functions
- Event handlers are created via factory functions and registered in the order specified in the design spec (lines 332-337)
- No slash command for Sprint 0 (TUI panel is Sprint 1)
- No status bar updates from handlers for Sprint 0 -- add in Item 0.7

**Step-by-step:**
1. Write the extension function skeleton with state initialization
2. Register all 11 tools
3. Wire up all 5 event handlers
4. Test manually: start a pi session, call `guardrail_standalone_init`, verify tools work

**Test strategy:**
- Integration test: simulate a pi session by calling the extension function with a mock `pi` object
- Verify all 11 tools are registered
- Verify all 5 event handlers are registered
- Test the event handler flow: session_start -> record_read -> edit (blocked) -> read -> edit (allowed)

**Integration points:** This is the central wiring point. Every previous module converges here.

---

#### Item 0.7: Status Bar

**Files to create/modify:**
- `/mnt/data/agent-guardrails-template/pi-extension/status.ts` (new)
- `/mnt/data/agent-guardrails-template/pi-extension/handlers.ts` (modify: add status bar updates)

**status.ts should contain:**

```typescript
export function renderStatusBar(deps: {
  strikeCounter: StrikeCounter;
  scopeValidator: ScopeValidator;
  violationLog: ViolationLog;
  mcpClient: any;
}): string;
```

Implementation:
- Build the status string segment by segment, omitting empty segments
- Strikes: check `strikeCounter.getAllStrikes()`, if any task has strikes > 0, show `!!{count}/{max}`
- Scope: `scopeValidator.getScope()`, show first path or `unscoped`
- Violations: `violationLog.getSummary()`, show `!{total}v` if > 0
- MCP: `mcpClient?.isConnected() ? 'mcp:*' : 'mcp:.'`
- Join with spaces, prefix with `g: `

Example outputs:
- No issues, no MCP: `g: ok`
- Active strikes, scope set, MCP connected: `g: !!2/3 src/ mcp:*`
- Violations: `g: src/ !3v`

**Modify handlers.ts:**
- Add a `updateStatusBar(ctx, deps)` helper that calls `ctx.ui.setStatus("guardrails", renderStatusBar(deps))` if `config.statusBarEnabled && ctx?.hasUI`
- Call `updateStatusBar` from the session_start handler
- Call `updateStatusBar` after each violation is logged (in preEdit and bashSafety handlers)
- Call `updateStatusBar` after each read tracking (in readTracking handler -- shows updated read count)

**Step-by-step:**
1. Write `status.ts` with `renderStatusBar`
2. Unit test `renderStatusBar` with various states
3. Add `updateStatusBar` to handlers.ts
4. Wire it into each handler

**Test strategy:**
- Unit test `renderStatusBar`: empty state, strikes only, violations only, all populated, MCP connected/disconnected

**Integration points:** handlers.ts, index.ts

---

#### Item 0.8: Install Script

**Files to create:**
- `/mnt/data/agent-guardrails-template/pi-extension/install.mjs`

**What install.mjs should contain:**

Follow the pi-messenger installer pattern exactly (pi-messenger `install.mjs` lines 1-162). Key behaviors:
- Default mode: copy package contents to `~/.pi/agent/extensions/pi-guardrails/`
- `--remove` mode: delete the extension directory
- `--help` mode: print usage
- Skip `.git`, `node_modules`, `.DS_Store` during copy
- For updates: remove the old directory first (clean slate), then copy fresh
- Create the sessions subdirectory during install

**Step-by-step:**
1. Write install.mjs based on pi-messenger's template
2. Change `EXTENSION_DIR` to `~/.pi/agent/extensions/pi-guardrails/`
3. Remove crew agent logic (not needed for guardrails)
4. Remove any git-clone logic
5. Test: run `node install.mjs` and verify files are copied correctly

**Test strategy:**
- Manual test: run the installer, check the extension directory exists with all files
- Manual test: run `--remove`, check the directory is deleted
- Manual test: run the installer again (update mode), verify clean slate

**Integration points:** This is required for the success criteria "pi install npm:@thearchitectit/pi-guardrails installs and registers the extension"

---

#### Item 0.9: End-to-End Verification

**This is not a code item -- it is a manual verification step.**

After all Sprint 0 items are complete:

1. Run `node install.mjs` to install the extension
2. Start a pi coding agent session
3. Verify the extension loads (no errors in console)
4. Call `guardrail_standalone_init` with a project slug
5. Call `guardrail_record_read` for a file
6. Call `guardrail_verify_read` for that file (should return wasRead: true)
7. Try to edit a file that was NOT read -- verify it is blocked by the event handler
8. Read the file first, then try to edit it -- verify it is allowed
9. Call `guardrail_set_scope` with a restricted path
10. Try to edit a file outside scope -- verify it is blocked
11. Try a dangerous bash command -- verify it is blocked
12. Call `guardrail_status` -- verify it returns the full state summary
13. Verify the status bar appears and updates after each action

**If any of these fail, debug and fix before proceeding to Sprint 1.**

---

### Sprint 1: TUI Panel, Skills, MCP Bridge

#### Item 1.1: TUI Guardrails Panel

**Files to create:**
- `/mnt/data/agent-guardrails-template/pi-extension/tui/guardrails-panel.ts`

**What this file should contain:**

A TUI overlay component following pi-messenger's `MessengerOverlay` pattern. Import `Component`, `Focusable`, `Box`, `Text`, `Container`, `Spacer` from `@earendil-works/pi-tui` (same as pi-subagents `index.ts:20`).

The panel should display:
- **Header row**: "Guardrails Dashboard" with safety score badge (green/yellow/red based on violation state)
- **Four Laws section**: 4 rows, one per law, each with a checkmark or X indicator
- **Strike Tracker section**: table of tasks with strike counts, color-coded (green < 2/3, yellow 2/3, red = max)
- **Scope section**: list of authorized paths
- **Violation Log section**: scrollable list of violations with severity badges
- **MCP Status row**: connected/disconnected indicator

The panel is registered via `pi.registerCommand("guardrails", { handler: ... })` (same as pi-messenger line 490):

```typescript
pi.registerCommand("guardrails", {
  description: "Open guardrails dashboard",
  handler: async (_args, ctx) => {
    if (!ctx.hasUI) return;
    await ctx.ui.custom<void>(
      (tui, theme, _keybindings, done) => {
        return new GuardrailsPanel(tui, theme, done, deps);
      },
      { overlay: true }
    );
  }
});
```

**Step-by-step:**
1. Create `GuardrailsPanel` class extending pi-tui's Component
2. Implement `render()` method that builds the panel layout
3. Add a `refresh()` method that re-reads state from the stores
4. Register the `/guardrails` command in index.ts
5. Test by opening the panel in a live pi session

**Test strategy:**
- Visual test in a live pi session (TUI is hard to unit test)
- Verify the panel opens with `/guardrails`
- Verify it displays correct data after guardrail operations
- Verify it closes cleanly (Escape key or `done()` callback)

**Integration points:** index.ts (command registration), handlers.ts (trigger panel refresh after violations)

---

#### Item 1.2: Skill Files

**Files to create:**
- `/mnt/data/agent-guardrails-template/pi-extension/skills/guardrails-core/SKILL.md`
- `/mnt/data/agent-guardrails-template/pi-extension/skills/guardrails-dashboard/SKILL.md`

**guardrails-core/SKILL.md** should contain:

The YAML frontmatter (following the template's existing pattern from `skills/four-laws/SKILL.md`):
```yaml
id: guardrails-core
name: Pi Guardrails Core
description: Available guardrail tools and automatic enforcement behavior for pi agents
version: 1.0.0
tags: [safety, core, pi]
applies_to: [pi]
tools: [guardrail_standalone_init, guardrail_record_read, guardrail_verify_read,
        guardrail_set_scope, guardrail_check_scope, guardrail_record_attempt,
        guardrail_check_strikes, guardrail_reset_strikes, guardrail_check_halt,
        guardrail_log_violation, guardrail_status]
```

The body should teach the agent:
1. The Four Laws are enforced automatically via event handlers -- reads are tracked, unread edits are blocked, out-of-scope edits are blocked
2. The agent does NOT need to call guardrail tools manually before every edit -- the event handlers handle enforcement
3. Explicit tool calls are available for self-checking: `guardrail_verify_read` before editing, `guardrail_check_scope` before operating on a file, `guardrail_check_halt` before uncertain operations
4. Three-strikes workflow: `guardrail_record_attempt` after each attempt, `guardrail_check_strikes` to check status, `guardrail_reset_strikes` after user escalation
5. `guardrail_status` gives a full state summary
6. MCP bridge tools require the Go server; check mode with `guardrail_status`

**guardrails-dashboard/SKILL.md** should contain:

```yaml
id: guardrails-dashboard
name: Pi Guardrails Dashboard
description: How to use and interpret the guardrails panel and status bar in the pi TUI
version: 1.0.0
tags: [safety, pi, tui]
applies_to: [pi]
```

The body should teach the agent:
1. `/guardrails` slash command opens the overlay panel
2. Status bar shows compact state: `g: ok`, `g: !!2/3 src/ mcp:*`, etc.
3. Panel sections: safety score, four laws status, strike tracker, scope, violations, MCP status

**Step-by-step:**
1. Create the `skills/guardrails-core/` and `skills/guardrails-dashboard/` directories
2. Write each SKILL.md
3. Verify skills are discovered by pi (the `"pi": { "skills": "./skills" }` entry in package.json handles this)

**Test strategy:**
- Run `pi skills list` (or equivalent) and verify both guardrails skills appear
- Verify the skill content is loaded when the agent references it

---

#### Item 1.3: MCP Bridge Client

**Files to create:**
- `/mnt/data/agent-guardrails-template/pi-extension/mcp-bridge/mcp-client.ts`
- `/mnt/data/agent-guardrails-template/pi-extension/mcp-bridge/mcp-tools.ts`

**mcp-client.ts:**

```typescript
export class MCPClient {
  private transport: any; // StdioClientTransport from @modelcontextprotocol/sdk
  private client: any;    // Client from @modelcontextprotocol/sdk
  private connected: boolean;
  private tools: string[];
  private reconnectAttempts: number;
  private maxReconnectAttempts: number;

  constructor();
  async tryConnect(endpoint: string): Promise<boolean>;
  async callTool(toolName: string, params: any): Promise<any>;
  isConnected(): boolean;
  getTools(): string[];
  async close(): Promise<void>;
}
```

Implementation details:
- Import `@modelcontextprotocol/sdk` conditionally (try/catch on import) -- if not available, MCP is permanently unavailable
- `tryConnect()` spawns the Go server binary as a child process using `StdioClientTransport`
- After connection, call `client.listTools()` to discover available tools, store them in `this.tools`
- Reconnection: exponential backoff (1s base, 30s max, 5 attempts max) with jitter
- Each `callTool()` checks `isConnected()` first; if not connected, returns an informative message instead of throwing

**mcp-tools.ts:**

Register a single `guardrail_mcp` tool with an `action` parameter (following pi-messenger's single-tool pattern):

```typescript
export function registerMCPBridgeTool(pi: ExtensionAPI, mcpClient: MCPClient): void {
  pi.registerTool({
    name: "guardrail_mcp",
    label: "Guardrails MCP Bridge",
    description: "Proxy to the Guardrail MCP server. Requires the Go server to be running.",
    promptSnippet: "Access MCP guardrail server tools when connected",
    parameters: Type.Object({
      action: Type.String({ description: "MCP tool name to call (e.g., 'guardrail_validate_bash')" }),
      params: Type.Optional(Type.Record(Type.String(), Type.Any(), { description: "Parameters for the MCP tool" }))
    }),
    async execute(_id: string, args: any) {
      if (!mcpClient.isConnected()) {
        return "Tool requires the Guardrail MCP server. Start it and run guardrail_standalone_init with MCP endpoint configured.";
      }
      return mcpClient.callTool(args.action, args.params || {});
    }
  });
}
```

**Step-by-step:**
1. Write `mcp-client.ts` with full reconnection logic
2. Write `mcp-tools.ts` with the single bridge tool
3. Wire into index.ts: instantiate MCPClient, call `registerMCPBridgeTool(pi, mcpClient)`
4. Update the session_start handler to attempt MCP connection
5. Test with the Go MCP server running and not running

**Test strategy:**
- Unit test MCPClient with a mock MCP server (or skip if `@modelcontextprotocol/sdk` is not available)
- Integration test: start the Go MCP server, verify the bridge connects and tools are discoverable
- Integration test: with the server NOT running, verify the bridge tool returns the "requires server" message
- Test reconnection: start the server after the session starts, verify the bridge connects on retry

**Integration points:** index.ts (registration), handlers.ts (session_start), tools.ts (initSession)

---

### Sprint 2: Gap Closure (Outline Only)

These items are deprioritized relative to the core extension. They should be implemented after Sprint 0 and Sprint 1 are complete and verified.

**Item 2.1: Bash Command Classification Engine**
- Location: `/mnt/data/agent-guardrails-template/guardrails/bash-classify.ts`
- Replaces the hardcoded denylist in `halt-checker.ts`
- Import: add `import { classifyCommand } from "../../guardrails/bash-classify.js"` in halt-checker.ts
- The `checkCommand` method should call `classifyCommand(cmd)` and use the category to decide whether to block/warn/allow
- Depends on: Sprint 0 Item 0.3 (halt-checker.ts must exist)

**Item 2.2: Prompt Injection Defense**
- Location: `/mnt/data/agent-guardrails-template/pi-extension/injection/detector.ts`
- New `tool_call` handler: scan `event.input` for injection patterns when the tool is a user-facing tool
- New `tool_result` handler: canary token detection in agent output
- Requires deciding: which tools receive injection scanning? (Probably: any tool that accepts free-form user input)
- Depends on: Sprint 0 core extension (for handler registration mechanism)

**Item 2.3: Output Validation / Sensitive Data Filter**
- Location: `/mnt/data/agent-guardrails-template/pi-extension/output-validator/validator.ts`
- Critical design gap: pi's `tool_result` handler cannot block output (it is side-effect only, not blocking). Output validation must either:
  - (a) Run in a `tool_call` handler on the agent's response tool (if pi has such a thing), or
  - (b) Work as a post-hoc scanner that warns the user via `ctx.ui.notify()` after detecting sensitive data
  - (c) Require a new pi API for output interception
- This needs resolution before implementation can proceed
- Depends on: Sprint 0 core extension

**Item 2.4: Per-Tool Permission System**
- Location: `/mnt/data/agent-guardrails-template/pi-extension/permissions/permissions.ts`
- New `tool_call` handler: check tool name against permission matrix, return `{ block: true }` if "blocked", return nothing if "auto"
- The "ask" level is problematic: pi's tool_call handler can only block or allow, not "ask." Implementation: block the call with a message telling the agent to ask the user first. The agent must then re-attempt after user confirmation.
- Depends on: Sprint 0 core extension

**Item 2.5: Team Policy Configuration**
- Deferred to Sprint 3+. Enterprise feature, not needed for first release.

---

### Build Sequence Checklist

Sprint 0 (must complete before anything else):

- [ ] Item 0.1: Create project scaffold and package.json (verify peer dep namespace first)
- [ ] Item 0.2: Create types.ts and config.ts
- [ ] Item 0.3: Implement file-read-store.ts + tests
- [ ] Item 0.3: Implement scope-validator.ts + tests
- [ ] Item 0.3: Implement strike-counter.ts + tests
- [ ] Item 0.3: Implement halt-checker.ts + tests
- [ ] Item 0.3: Implement violation-log.ts + tests
- [ ] Item 0.3: Implement session-store.ts + tests
- [ ] Item 0.4: Implement tools.ts (11 tool execute functions) + tests
- [ ] Item 0.5: Implement handlers.ts (5 event handler factories) + tests
- [ ] Item 0.6: Implement index.ts (wire everything together)
- [ ] Item 0.7: Implement status.ts + integrate into handlers
- [ ] Item 0.8: Write install.mjs
- [ ] Item 0.9: End-to-end verification in live pi session

Sprint 1 (after Sprint 0 passes all verification):

- [ ] Item 1.1: Implement guardrails-panel.ts (TUI overlay)
- [ ] Item 1.1: Register `/guardrails` slash command in index.ts
- [ ] Item 1.2: Write guardrails-core skill file
- [ ] Item 1.2: Write guardrails-dashboard skill file
- [ ] Item 1.3: Implement mcp-client.ts (conditional MCP SDK import)
- [ ] Item 1.3: Implement mcp-tools.ts (single guardrail_mcp bridge tool)
- [ ] Item 1.3: Wire MCP bridge into index.ts and session_start handler
- [ ] Full integration test: standalone mode + MCP bridge mode

Sprint 2 (gap closure, after Sprint 1 is stable):

- [ ] Item 2.1: Bash command classification engine (replace hardcoded denylist)
- [ ] Item 2.2: Prompt injection defense (pattern matching + heuristics)
- [ ] Item 2.3: Output validation (resolve blocking mechanism first)
- [ ] Item 2.4: Per-tool permission system

---

### Key Files Referenced

Design documents:
- `/mnt/data/agent-guardrails-template/docs/superpowers/specs/2026-05-16-pi-guardrails-extension-design.md`
- `/mnt/data/agent-guardrails-template/docs/superpowers/specs/2026-05-16-gap-analysis.md`
- `/mnt/data/agent-guardrails-template/docs/superpowers/specs/2026-05-17-pi-guardrails-sprint-1.md`

Reference extensions (API patterns):
- `/home/user001/.pi/agent/extensions/pi-messenger/package.json` (pi manifest format: lines 46-49)
- `/home/user001/.pi/agent/extensions/pi-messenger/index.ts` (handler patterns: lines 703, 781, 1110; tool registration: line 356; command registration: line 490; status bar: line 286; TypeBox import: line 14; import namespace: lines 11-12)
- `/home/user001/.pi/agent/extensions/pi-messenger/install.mjs` (installer pattern: lines 1-162)
- `/home/user001/.pi/agent/extensions/subagent/package.json` (peer deps: lines 55-73; type: module: line 8)
- `/home/user001/.pi/agent/extensions/subagent/src/extension/index.ts` (ToolDefinition pattern: lines 398-475; event handler registration: lines 511, 543, 547; import namespace: lines 18-20)

Existing guardrails:
- `/mnt/data/agent-guardrails-template/skills/guardrails-enforcer/SKILL.md` (skill format reference)
- `/mnt/data/agent-guardrails-template/skills/four-laws/SKILL.md` (Four Laws content)
- `/mnt/data/agent-guardrails-template/mcp-server/internal/mcp/server.go` (MCP tool schemas: lines 112-700; schema mismatches: lines 361, 518, 547, 573, 594, 682)
