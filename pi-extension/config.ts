import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import type { GuardrailsConfig } from "./types.js";
import { DEFAULT_CONFIG } from "./types.js";
import { PolicyLoader } from "./policy/policy-loader.js";

const EXTENSION_DIR = path.join(os.homedir(), ".pi", "agent", "extensions", "pi-guardrails");

export function getStorageDir(): string {
  return EXTENSION_DIR;
}

export function getSessionsDir(): string {
  return path.join(EXTENSION_DIR, "sessions");
}

export function getViolationsLogPath(): string {
  return path.join(EXTENSION_DIR, "violations.jsonl");
}

export function getConfigPath(): string {
  return path.join(EXTENSION_DIR, "config.json");
}

export function getMcpApiKey(): string | undefined {
  return process.env.PI_GUARDRAILS_MCP_API_KEY;
}

export function loadConfig(cwd?: string): GuardrailsConfig {
  // Start with extension-level config
  const configPath = getConfigPath();
  let base: GuardrailsConfig;
  try {
    if (!fs.existsSync(configPath)) {
      base = { ...DEFAULT_CONFIG };
    } else {
      const raw = fs.readFileSync(configPath, "utf-8");
      const parsed = JSON.parse(raw);
      base = { ...DEFAULT_CONFIG, ...parsed };
    }
  } catch {
    base = { ...DEFAULT_CONFIG };
  }

  // Apply policy layers (org -> team -> project) if cwd is provided
  if (cwd) {
    const policyLoader = new PolicyLoader();
    policyLoader.loadAll(cwd, {
      orgConfigPath: process.env.PI_GUARDRAILS_ORG_CONFIG,
      teamName: process.env.PI_GUARDRAILS_TEAM,
    });
    base = policyLoader.merge(base);
  }

  return base;
}

export function ensureDirs(): void {
  const dirs = [EXTENSION_DIR, getSessionsDir()];
  for (const dir of dirs) {
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }
  }
}
