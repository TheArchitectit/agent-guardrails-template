import { Type, type TUnsafe } from "@sinclair/typebox";

function StringEnum<T extends readonly string[]>(
  values: T,
  options?: { description?: string; default?: T[number] },
): TUnsafe<T[number]> {
  return Type.Unsafe<T[number]>({
    type: "string",
    enum: [...values],
    ...(options?.description && { description: options.description }),
    ...(options?.default && { default: options.default }),
  });
}

// --- Tool Parameter Schemas ---

export const InitSessionParams = Type.Object({
  projectSlug: Type.String({ description: "Project identifier" }),
  agentType: Type.Optional(Type.String({ description: "e.g. pi, claude-code" })),
  scope: Type.Optional(Type.Array(Type.String(), { description: "Initial authorized file paths/globs" })),
  rules: Type.Optional(Type.Array(Type.String(), { description: "Enabled rule IDs" })),
});

export const RecordReadParams = Type.Object({
  filePath: Type.String({ description: "Path to the file that was read" }),
});

export const VerifyReadParams = Type.Object({
  filePath: Type.String({ description: "Path to check if it was read" }),
});

export const SetScopeParams = Type.Object({
  paths: Type.Array(Type.String(), { description: "Authorized file path prefixes" }),
  reason: Type.Optional(Type.String({ description: "Why this scope is set" })),
});

export const CheckScopeParams = Type.Object({
  filePath: Type.String({ description: "Path to check" }),
  operation: StringEnum(["read", "edit", "delete"] as const, { description: "Type of operation" }),
});

export const RecordAttemptParams = Type.Object({
  task: Type.String({ description: "Task identifier" }),
  success: Type.Boolean({ description: "Whether the attempt succeeded" }),
  error: Type.Optional(Type.String({ description: "Error message if failed" })),
});

export const CheckStrikesParams = Type.Object({
  task: Type.String({ description: "Task identifier" }),
});

export const ResetStrikesParams = Type.Object({
  task: Type.String({ description: "Task identifier" }),
});

export const CheckHaltParams = Type.Object({
  operation: Type.String({ description: "Operation being evaluated" }),
  filePath: Type.Optional(Type.String({ description: "File path if relevant" })),
  details: Type.Optional(Type.String({ description: "Additional context" })),
});

export const LogViolationParams = Type.Object({
  law: StringEnum(["read-before-edit", "stay-in-scope", "verify-before-commit", "halt-when-uncertain"] as const),
  severity: StringEnum(["warning", "critical"] as const),
  details: Type.String({ description: "What happened" }),
  filePath: Type.Optional(Type.String()),
  operation: Type.Optional(Type.String()),
});

export const StatusParams = Type.Object({});

export const McpBridgeParams = Type.Object({
  action: Type.String({ description: "MCP tool name to call (e.g. validate_bash)" }),
  params: Type.Optional(Type.Record(Type.String(), Type.Unknown(), { description: "Parameters for the MCP tool" })),
});

export const PreWorkCheckParams = Type.Object({
  cwd: Type.Optional(Type.String({ description: "Working directory for context" })),
});

export const DetectCreepParams = Type.Object({
  scopePaths: Type.Array(Type.String(), { description: "Authorized scope path prefixes" }),
  modifiedFiles: Type.Array(Type.String(), { description: "List of file paths that were modified" }),
});

export const CheckPatternParams = Type.Object({
  code: Type.String({ description: "Code content to check against pattern rules" }),
  filePath: Type.Optional(Type.String({ description: "File path for rule matching context" })),
});

export const ValidateGitParams = Type.Object({
  command: Type.String({ description: "Git command to validate" }),
});

// --- Core Types ---

export interface Attempt {
  success: boolean;
  error?: string;
  timestamp: string;
}

export interface Violation {
  id: string;
  law: string;
  severity: "warning" | "critical";
  details: string;
  filePath?: string;
  operation?: string;
  timestamp: string;
}

export interface HaltResult {
  shouldHalt: boolean;
  reasons: string[];
  severity: "none" | "warning" | "critical";
  suggestions: string[];
}

export interface CommandCheckResult {
  shouldHalt: boolean;
  reason?: string;
  category?: "destructive" | "elevated" | "network";
}

export interface PreWorkCheckResult {
  risks: { category: string; description: string; severity: "warning" | "critical" }[];
  recentViolations: number;
  checklist: string[];
}

export interface CreepResult {
  hasCreep: boolean;
  inScopeModified: string[];
  outOfScopeModified: string[];
  warnings: string[];
}

export interface PatternCheckResult {
  ruleId: string;
  description: string;
  match: string;
  severity: "warning" | "critical";
}

export interface GitValidationResult {
  allowed: boolean;
  reason?: string;
  category?: "protected-branch" | "commit-format" | "force-push" | "destructive";
}

export interface SessionState {
  id: string;
  projectSlug: string;
  createdAt: string;
  scope: {
    paths: string[];
    reason: string | null;
  };
  filesRead: Record<string, string>;
  strikes: Record<string, { attempts: Attempt[] }>;
  mcpEndpoint: string | null;
  mcpConnected: boolean;
}

export interface GuardrailsConfig {
  mcpBinaryPath: string;
  enabledRules: string[];
  autoRegister: boolean;
  defaultScope: string[];
  maxStrikes: number;
  statusBarEnabled: boolean;
  panelAutoOpen: boolean;
  toolPermissions?: {
    defaultLevel?: "auto" | "ask" | "blocked";
    tools?: Record<string, "auto" | "ask" | "blocked">;
  };
  injectionDefense?: {
    blockThreshold?: number;
    warnThreshold?: number;
    heuristicEnabled?: boolean;
  };
  outputValidation?: {
    enablePII?: boolean;
    autoRedact?: boolean;
    redactionText?: string;
    contentFilter?: {
      deniedTopics?: string[];
      allowedTopics?: string[];
      strictMode?: boolean;
      topicPatterns?: Record<string, string[]>;
    };
  };
  canary?: {
    prefix?: string;
    tokenLength?: number;
  };
  gitPolicy?: {
    protectedBranches?: string[];
    commitFormat?: string;
    requireAIAttribution?: boolean;
  };
}

export const DEFAULT_CONFIG: GuardrailsConfig = {
  mcpBinaryPath: "",
  enabledRules: ["four-laws", "three-strikes", "scope-validator"],
  autoRegister: true,
  defaultScope: [],
  maxStrikes: 3,
  statusBarEnabled: true,
  panelAutoOpen: false,
};
