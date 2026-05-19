import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import type { GuardrailsConfig } from "../types.js";
import { DEFAULT_CONFIG } from "../types.js";

export interface PolicyLayer {
  name: string;
  source: string;
  config: Partial<GuardrailsConfig>;
}

export class PolicyLoader {
  private layers: PolicyLayer[] = [];

  loadOrgPolicy(orgConfigPath?: string): PolicyLayer | null {
    const searchPaths = [
      orgConfigPath,
      path.join(os.homedir(), ".pi", "guardrails-org.json"),
      "/etc/pi-guardrails/org.json",
    ].filter(Boolean) as string[];

    for (const configPath of searchPaths) {
      try {
        if (!fs.existsSync(configPath)) continue;
        const raw = fs.readFileSync(configPath, "utf-8");
        const parsed = JSON.parse(raw);
        const layer: PolicyLayer = {
          name: "organization",
          source: configPath,
          config: parsed,
        };
        this.layers.push(layer);
        return layer;
      } catch {
        // Skip malformed config
      }
    }
    return null;
  }

  loadTeamPolicy(teamName?: string): PolicyLayer | null {
    const searchPaths = [
      teamName ? path.join(os.homedir(), ".pi", "teams", teamName, "guardrails.json") : null,
      path.join(os.homedir(), ".pi", "guardrails-team.json"),
    ].filter(Boolean) as string[];

    for (const configPath of searchPaths) {
      try {
        if (!fs.existsSync(configPath)) continue;
        const raw = fs.readFileSync(configPath, "utf-8");
        const parsed = JSON.parse(raw);
        const layer: PolicyLayer = {
          name: "team",
          source: configPath,
          config: parsed,
        };
        this.layers.push(layer);
        return layer;
      } catch {
        // Skip malformed config
      }
    }
    return null;
  }

  loadProjectPolicy(cwd: string): PolicyLayer | null {
    const candidates = [
      path.join(cwd, ".pi-guardrails.json"),
      path.join(cwd, ".pi", "guardrails.json"),
    ];

    for (const configPath of candidates) {
      try {
        if (!fs.existsSync(configPath)) continue;
        const raw = fs.readFileSync(configPath, "utf-8");
        const parsed = JSON.parse(raw);
        const layer: PolicyLayer = {
          name: "project",
          source: configPath,
          config: parsed,
        };
        this.layers.push(layer);
        return layer;
      } catch {
        // Skip malformed config
      }
    }
    return null;
  }

  loadAll(cwd: string, options?: { orgConfigPath?: string; teamName?: string }): void {
    this.loadOrgPolicy(options?.orgConfigPath);
    this.loadTeamPolicy(options?.teamName);
    this.loadProjectPolicy(cwd);
  }

  merge(base: GuardrailsConfig): GuardrailsConfig {
    let merged = { ...base };

    const order: Record<string, number> = { organization: 0, team: 1, project: 2 };
    const sorted = [...this.layers].sort((a, b) => (order[a.name] ?? 99) - (order[b.name] ?? 99));

    for (const layer of sorted) {
      merged = this.deepMerge(merged, layer.config);
    }

    return merged;
  }

  getLayers(): PolicyLayer[] {
    return [...this.layers];
  }

  clear(): void {
    this.layers = [];
  }

  private deepMerge<T extends Record<string, any>>(target: T, source: Partial<T>): T {
    const result = { ...target };
    for (const key of Object.keys(source) as (keyof T)[]) {
      const sv = source[key];
      if (sv !== undefined && sv !== null) {
        const tv = target[key];
        if (typeof sv === "object" && !Array.isArray(sv) && typeof tv === "object" && !Array.isArray(tv) && tv !== null) {
          (result as any)[key] = this.deepMerge(tv as any, sv as any);
        } else {
          (result as any)[key] = sv;
        }
      }
    }
    return result;
  }
}
