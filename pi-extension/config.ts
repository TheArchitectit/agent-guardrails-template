import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import type { GuardrailsConfig } from "./types.js";
import { DEFAULT_CONFIG } from "./types.js";

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

export function loadConfig(): GuardrailsConfig {
  const configPath = getConfigPath();
  try {
    if (!fs.existsSync(configPath)) {
      return { ...DEFAULT_CONFIG };
    }
    const raw = fs.readFileSync(configPath, "utf-8");
    const parsed = JSON.parse(raw);
    return { ...DEFAULT_CONFIG, ...parsed };
  } catch {
    return { ...DEFAULT_CONFIG };
  }
}

export function ensureDirs(): void {
  const dirs = [EXTENSION_DIR, getSessionsDir()];
  for (const dir of dirs) {
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }
  }
}
