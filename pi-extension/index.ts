import * as fs from "node:fs";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { loadConfig, getSessionsDir, getViolationsLogPath } from "./config.js";
import { FileReadStore } from "./standalone/file-read-store.js";
import { StrikeCounter } from "./standalone/strike-counter.js";
import { ScopeValidator } from "./standalone/scope-validator.js";
import { HaltChecker } from "./standalone/halt-checker.js";
import { ViolationLog } from "./standalone/violation-log.js";
import { SessionStore } from "./standalone/session-store.js";
import { MCPClient } from "./mcp-bridge/mcp-client.js";
import { registerMCPBridgeTool } from "./mcp-bridge/mcp-tools.js";
import { PermissionManager } from "./permissions/permissions.js";
import { ContentFilter } from "./output-validator/content-filter.js";
import { CanaryTokenManager } from "./injection/canary.js";
import { PreWorkChecker } from "./standalone/pre-work-checker.js";
import { FeatureCreepDetector } from "./standalone/feature-creep-detector.js";
import { PatternRuleEngine } from "./standalone/pattern-rule-engine.js";
import { GitValidator } from "./standalone/git-validator.js";
import {
  initSession,
  recordRead,
  verifyRead,
  setScope,
  checkScope,
  recordAttempt,
  checkStrikes,
  resetStrikes,
  checkHalt,
  logViolation,
  getStatus,
} from "./tools.js";
import {
  createSessionStartHandler,
  createSessionShutdownHandler,
  createReadTrackingHandler,
  createPreEditHandler,
  createBashSafetyHandler,
  createInjectionDefenseHandler,
  createOutputValidationHandler,
  createPermissionHandler,
  type HandlerDeps,
} from "./handlers.js";
import {
  InitSessionParams,
  RecordReadParams,
  VerifyReadParams,
  SetScopeParams,
  CheckScopeParams,
  RecordAttemptParams,
  CheckStrikesParams,
  ResetStrikesParams,
  CheckHaltParams,
  LogViolationParams,
  StatusParams,
  PreWorkCheckParams,
  DetectCreepParams,
  CheckPatternParams,
  ValidateGitParams,
} from "./types.js";
import { GuardrailsPanel } from "./tui/guardrails-panel.js";

export default function piGuardrailsExtension(pi: ExtensionAPI) {
  // ===========================================================================
  // State initialization
  // ===========================================================================
  const config = loadConfig(process.cwd());

  fs.mkdirSync(getSessionsDir(), { recursive: true });

  const fileReadStore = new FileReadStore();
  const strikeCounter = new StrikeCounter(config.maxStrikes);
  const scopeValidator = new ScopeValidator();
  const haltChecker = new HaltChecker();
  const violationLog = new ViolationLog(getViolationsLogPath());
  const sessionStore = new SessionStore(config.maxStrikes);
  const mcpClient = new MCPClient();
  const permissionManager = new PermissionManager(config.toolPermissions);

  // Output security: content filter + canary tokens
  const contentFilter = config.outputValidation?.contentFilter
    ? new ContentFilter(config.outputValidation.contentFilter)
    : undefined;

  const canaryManager = config.canary
    ? new CanaryTokenManager({
        prefix: config.canary.prefix,
        tokenLength: config.canary.tokenLength,
      })
    : undefined;

  // GAP modules
  const preWorkChecker = new PreWorkChecker(violationLog, sessionStore);
  const featureCreepDetector = new FeatureCreepDetector();
  const patternRuleEngine = new PatternRuleEngine();
  const gitValidator = new GitValidator(config.gitPolicy);

  const deps: HandlerDeps = {
    sessionStore,
    fileReadStore,
    scopeValidator,
    strikeCounter,
    haltChecker,
    violationLog,
    mcpClient,
    config,
    permissionManager,
    contentFilter,
    canaryManager,
  };

  // ===========================================================================
  // Tool Registration
  // ===========================================================================

  pi.registerTool({
    name: "guardrail_init",
    label: "Guardrails Init",
    description:
      "Initialize a guardrails session. Sets up scope, strike tracking, and file read enforcement. Call this at the start of each session.",
    promptSnippet: "Initialize guardrails session",
    parameters: InitSessionParams,
    async execute(_id: string, params: any) {
      const result = initSession(sessionStore, mcpClient, params);
      if (config.mcpBinaryPath && !mcpClient.isConnected()) {
        const connected = await mcpClient.tryConnect(config.mcpBinaryPath).catch(() => false);
        if (connected && sessionStore.getState()) {
          sessionStore.setMcpConnected(config.mcpBinaryPath, true);
          result.mode = "mcp-bridge";
          result.mcpConnected = true;
          result.availableTools = ["guardrail_mcp", ...mcpClient.getTools()];
        }
      }
      return result;
    },
  });

  pi.registerTool({
    name: "guardrail_record_read",
    label: "Record File Read",
    description: "Mark a file as having been read by the agent. Required before editing (Law 1: Read Before Editing).",
    promptSnippet: "Record that a file was read",
    parameters: RecordReadParams,
    execute(_id: string, params: any) {
      return recordRead(fileReadStore, params);
    },
  });

  pi.registerTool({
    name: "guardrail_verify_read",
    label: "Verify File Read",
    description: "Check whether a file has been read before editing it. Enforces Law 1: Read Before Editing.",
    promptSnippet: "Check if a file was read",
    parameters: VerifyReadParams,
    execute(_id: string, params: any) {
      return verifyRead(fileReadStore, params);
    },
  });

  pi.registerTool({
    name: "guardrail_set_scope",
    label: "Set Scope",
    description: "Define which file paths the agent is authorized to operate on. Enforces Law 2: Stay in Scope.",
    promptSnippet: "Set authorized file scope",
    parameters: SetScopeParams,
    execute(_id: string, params: any) {
      return setScope(scopeValidator, sessionStore, params);
    },
  });

  pi.registerTool({
    name: "guardrail_check_scope",
    label: "Check Scope",
    description: "Check whether a file path is within the authorized scope for a given operation.",
    promptSnippet: "Check if path is in scope",
    parameters: CheckScopeParams,
    execute(_id: string, params: any) {
      return checkScope(scopeValidator, params);
    },
  });

  pi.registerTool({
    name: "guardrail_record_attempt",
    label: "Record Attempt",
    description:
      "Record a task attempt (success or failure). Consecutive failures trigger the Three Strikes rule. Enforces Law 4: Halt When Uncertain.",
    promptSnippet: "Record a task attempt result",
    parameters: RecordAttemptParams,
    execute(_id: string, params: any) {
      return recordAttempt(strikeCounter, sessionStore, params);
    },
  });

  pi.registerTool({
    name: "guardrail_check_strikes",
    label: "Check Strikes",
    description: "Check the strike count for a task. Max strikes triggers a halt recommendation.",
    promptSnippet: "Check strike count for a task",
    parameters: CheckStrikesParams,
    execute(_id: string, params: any) {
      return checkStrikes(strikeCounter, params);
    },
  });

  pi.registerTool({
    name: "guardrail_reset_strikes",
    label: "Reset Strikes",
    description: "Reset the strike count for a task after a successful resolution.",
    promptSnippet: "Reset strike count",
    parameters: ResetStrikesParams,
    execute(_id: string, params: any) {
      return resetStrikes(strikeCounter, sessionStore, params);
    },
  });

  pi.registerTool({
    name: "guardrail_check_halt",
    label: "Check Halt",
    description:
      "Evaluate whether an operation should be halted based on the Four Laws and Three Strikes rule.",
    promptSnippet: "Check if operation should halt",
    parameters: CheckHaltParams,
    execute(_id: string, params: any) {
      return checkHalt(haltChecker, params);
    },
  });

  pi.registerTool({
    name: "guardrail_log_violation",
    label: "Log Violation",
    description: "Log a guardrail violation with law, severity, and context details.",
    promptSnippet: "Log a guardrail violation",
    parameters: LogViolationParams,
    execute(_id: string, params: any) {
      return logViolation(violationLog, params);
    },
  });

  pi.registerTool({
    name: "guardrail_status",
    label: "Guardrails Status",
    description: "Get the current guardrails session status including scope, strikes, violations, and MCP connection state.",
    promptSnippet: "Get guardrails status",
    parameters: StatusParams,
    execute(_id: string, _params: any) {
      return getStatus(sessionStore, fileReadStore, strikeCounter, scopeValidator, violationLog, mcpClient);
    },
  });

  // MCP Bridge tool — proxies calls to Go MCP server
  registerMCPBridgeTool(pi, mcpClient);

  // GAP tools
  pi.registerTool({
    name: "guardrail_pre_work_check",
    label: "Pre-Work Check",
    description: "Generate a pre-work risk checklist from the violation log before starting a new task.",
    promptSnippet: "Run pre-work check",
    parameters: PreWorkCheckParams,
    execute(_id: string, params: any) {
      return preWorkChecker.generateChecklist(params.cwd || process.cwd());
    },
  });

  pi.registerTool({
    name: "guardrail_detect_creep",
    label: "Detect Feature Creep",
    description: "Compare modified files against authorized scope to detect feature creep.",
    promptSnippet: "Detect feature creep",
    parameters: DetectCreepParams,
    execute(_id: string, params: any) {
      return featureCreepDetector.detectCreep(params.scopePaths, params.modifiedFiles);
    },
  });

  pi.registerTool({
    name: "guardrail_check_pattern",
    label: "Check Pattern Rules",
    description: "Check code content against loaded prevention pattern rules from .guardrails/prevention-rules/pattern-rules.json.",
    promptSnippet: "Check code against pattern rules",
    parameters: CheckPatternParams,
    execute(_id: string, params: any) {
      return patternRuleEngine.checkPattern(params.code, params.filePath);
    },
  });

  pi.registerTool({
    name: "guardrail_validate_git",
    label: "Validate Git Operation",
    description: "Validate a git command against branch protection rules, commit format, and destructive operation policies.",
    promptSnippet: "Validate git operation",
    parameters: ValidateGitParams,
    execute(_id: string, params: any) {
      return gitValidator.validateGitOp(params.command);
    },
  });

  // ===========================================================================
  // Event Handlers
  // ===========================================================================

  pi.on("session_start", createSessionStartHandler(deps));
  pi.on("session_shutdown", createSessionShutdownHandler(deps));
  pi.on("tool_result", createReadTrackingHandler(deps));
  pi.on("tool_result", createOutputValidationHandler(deps));
  pi.on("tool_call", createPermissionHandler(deps));
  pi.on("tool_call", createPreEditHandler(deps));
  pi.on("tool_call", createBashSafetyHandler(deps));
  pi.on("tool_call", createInjectionDefenseHandler(deps));

  // ===========================================================================
  // Slash Command — /guardrails dashboard
  // ===========================================================================

  pi.registerCommand("guardrails", {
    description: "Open guardrails dashboard",
    handler: async (_args: any, ctx: any) => {
      if (!ctx?.hasUI) return;

      await ctx.ui.custom<void>(
        (tui: any, theme: any, _keybindings: any, done: () => void) => {
          return new GuardrailsPanel(tui, theme, done, {
            sessionStore,
            fileReadStore,
            scopeValidator,
            strikeCounter,
            haltChecker,
            violationLog,
            mcpConnected: mcpClient.isConnected(),
          });
        },
        { overlay: true },
      );
    },
  });
}
