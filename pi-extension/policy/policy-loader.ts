import * as fs from "node:fs";
import * as path from "node:path";
import type { GuardrailsConfig } from "../types.js";
import { DEFAULT_CONFIG } from "../types.js";

export interface PolicyLayer {
  name: string;
  source: string;
  config: Partial<GuardrailsConfig>;
}

export class PolicyLoader {
  private layers: PolicyLayer[] = [];

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

  loadOrgPolicy(orgConfigPath?: string): PolicyLayer | null {
    if (!orgConfigPath) return null;

    try {
      if (!fs.existsSync(orgConfigPath)) return null;
      const raw = fs.readFileSync(orgConfigPath, "utf-8");
      const parsed = JSON.parse(raw);
      const layer: PolicyLayer = {
        name: "organization",
        source: orgConfigPath,
        config: parsed,
      };
      this.layers.push(layer);
      return layer;
    } catch {
      return null;
    }
  }

  merge(base: GuardrailsConfig): GuardrailsConfig {
    // Merge order: base defaults -> org -> team -> project
    // Later layers override earlier ones
    let merged = { ...base };

    // Sort layers: organization first, then project
    const sorted = [...this.layers].sort((a, b) => {
      const order: Record<string, number> = { organization: 0, team: 1, project: 2 };
      return (order[a.name] ?? 99) - (order[b.name] ?? 99);
    });

    for (const layer of sorted) {
      merged = this.deepMerge(merged, layer.config);
    }

    return merged;
  }

  getLayers(): PolicyLayer[] {
    return [...this.layers];
  }

  private deepMerge<T extends Record<string, any>>(target: T, source: Partial<T>): T {
    const result = { ...target };
    for (const key of Object.keys(source) as (keyof T)[]) {
      const sv = source[key];
      const tv = target[key];
      if (sv !== undefined && sv !== null) {
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
