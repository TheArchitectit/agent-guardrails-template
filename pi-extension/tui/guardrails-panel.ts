import type { Component, Focusable, TUI, Theme } from "@earendil-works/pi-tui";
import { truncateToWidth, visibleWidth } from "@earendil-works/pi-tui";
import type { StrikeCounter } from "../standalone/strike-counter.js";
import type { ScopeValidator } from "../standalone/scope-validator.js";
import type { ViolationLog } from "../standalone/violation-log.js";
import type { FileReadStore } from "../standalone/file-read-store.js";
import type { SessionStore } from "../standalone/session-store.js";
import type { HaltChecker } from "../standalone/halt-checker.js";

interface PanelDeps {
  sessionStore: SessionStore;
  fileReadStore: FileReadStore;
  scopeValidator: ScopeValidator;
  strikeCounter: StrikeCounter;
  haltChecker: HaltChecker;
  violationLog: ViolationLog;
  mcpConnected: boolean;
}

function pad(str: string, width: number): string {
  const vis = visibleWidth(str);
  return vis >= width ? str : str + " ".repeat(width - vis);
}

function rpad(str: string, width: number): string {
  const vis = visibleWidth(str);
  return vis >= width ? str : " ".repeat(width - vis) + str;
}

export class GuardrailsPanel implements Component, Focusable {
  focused = false;
  private scrollOffset = 0;

  constructor(
    private tui: TUI,
    private theme: Theme,
    private done: () => void,
    private deps: PanelDeps,
  ) {}

  invalidate(): void {
    // no-op; state is read fresh on each render
  }

  handleInput(data: string): void {
    if (data === "\x1b" || data === "q") {
      this.done();
    } else if (data === "\x1b[A" || data === "k") {
      this.scrollOffset = Math.max(0, this.scrollOffset - 1);
    } else if (data === "\x1b[B" || data === "j") {
      this.scrollOffset++;
    }
  }

  render(width: number): string[] {
    const w = Math.min(width, 80);
    const innerW = w - 2;
    const lines: string[] = [];
    const b = (s: string) => this.theme.fg("dim", s);
    const accent = (s: string) => this.theme.fg("accent", s);
    const success = (s: string) => this.theme.fg("success", s);
    const warning = (s: string) => this.theme.fg("warning", s);
    const error = (s: string) => this.theme.fg("error", s);

    const row = (content: string) => b("│") + " " + pad(content, innerW - 2) + " " + b("│");
    const divider = b("├" + "─".repeat(innerW) + "┤");

    // Top border
    lines.push(b("┌" + "─".repeat(innerW) + "┐"));

    // Header
    const state = this.deps.sessionStore.getState();
    const headerText = ` Guardrails Dashboard  ${state ? accent(`[${state.id}]`) : ""}`;
    lines.push(row(this.theme.fg("toolTitle", headerText)));

    // Safety score
    const { total, critical, warning: warnCount } = this.deps.violationLog.getSummary();
    let scoreBadge: string;
    if (total === 0) {
      scoreBadge = success("SAFE");
    } else if (critical > 0) {
      scoreBadge = error("AT RISK");
    } else {
      scoreBadge = warning("CAUTION");
    }
    lines.push(row(`  Safety: ${scoreBadge}  Violations: ${total} (critical: ${critical}, warning: ${warnCount})`));

    lines.push(divider);

    // Four Laws section
    lines.push(row(this.theme.fg("toolTitle", " Four Laws of Agent Safety")));

    const filesRead = this.deps.fileReadStore.size;
    const hasScope = this.deps.scopeValidator.getScope().length > 0;

    const laws = [
      { name: "Law 1: Read Before Editing", status: filesRead > 0 ? success("✓") : warning("✗") },
      { name: "Law 2: Stay in Scope", status: hasScope ? success("✓") : warning("✗") },
      { name: "Law 3: Verify Before Committing", status: success("✓") },
      { name: "Law 4: Halt When Uncertain", status: success("✓") },
    ];

    for (const law of laws) {
      lines.push(row(`  ${law.status} ${law.name}`));
    }

    lines.push(divider);

    // Strike Tracker
    lines.push(row(this.theme.fg("toolTitle", " Strike Tracker")));

    const allStrikes = this.deps.strikeCounter.getAllStrikes();
    if (allStrikes.size === 0) {
      lines.push(row("  No active strikes"));
    } else {
      const maxStrikes = this.deps.strikeCounter.getMaxStrikes();
      for (const [task, attempts] of allStrikes) {
        let consec = 0;
        for (let i = attempts.length - 1; i >= 0; i--) {
          if (!attempts[i].success) consec++;
          else break;
        }
        const badge =
          consec >= maxStrikes
            ? error(`${consec}/${maxStrikes}`)
            : consec >= maxStrikes - 1
              ? warning(`${consec}/${maxStrikes}`)
              : `${consec}/${maxStrikes}`;
        lines.push(row(`  ${badge} ${task}`));
      }
    }

    lines.push(divider);

    // Scope
    lines.push(row(this.theme.fg("toolTitle", " Scope")));
    const scope = this.deps.scopeValidator.getScope();
    if (scope.length === 0) {
      lines.push(row("  " + warning("Unscoped") + " — all paths authorized"));
    } else {
      for (const p of scope) {
        lines.push(row(`  ${accent(p)}`));
      }
    }
    if (this.deps.scopeValidator.getReason()) {
      lines.push(row(`  Reason: ${this.deps.scopeValidator.getReason()}`));
    }

    lines.push(divider);

    // Files Read
    lines.push(row(this.theme.fg("toolTitle", ` Files Read (${filesRead})`)));

    // MCP Status
    lines.push(divider);
    const mcpStatus = this.deps.mcpConnected ? success("MCP Connected") : warning("MCP Disconnected (standalone mode)");
    lines.push(row(`  ${mcpStatus}`));

    // Footer
    lines.push(b("└" + "─".repeat(innerW) + "┘"));
    lines.push(b("  Press Esc or q to close, j/k to scroll"));

    return lines;
  }

  dispose(): void {
    // cleanup if needed
  }
}
