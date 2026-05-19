import * as crypto from "node:crypto";

export interface CanaryToken {
  id: string;
  token: string;
  filePath: string;
  insertedAt: string;
  triggered: boolean;
  triggeredAt?: string;
  triggeredIn?: string;
}

export interface CanaryConfig {
  /** Token prefix for identification */
  prefix?: string;
  /** Length of random token suffix */
  tokenLength?: number;
}

const DEFAULT_PREFIX = "CANARY_";

export class CanaryTokenManager {
  private tokens = new Map<string, CanaryToken>();
  private prefix: string;
  private tokenLength: number;

  constructor(config?: CanaryConfig) {
    this.prefix = config?.prefix ?? DEFAULT_PREFIX;
    this.tokenLength = config?.tokenLength ?? 16;
  }

  insert(filePath: string): CanaryToken {
    const id = `canary-${crypto.randomUUID().substring(0, 8)}`;
    const token = `${this.prefix}${crypto.randomBytes(this.tokenLength).toString("hex").toUpperCase()}`;

    const canary: CanaryToken = {
      id,
      token,
      filePath,
      insertedAt: new Date().toISOString(),
      triggered: false,
    };

    this.tokens.set(token, canary);
    return canary;
  }

  check(text: string): CanaryToken[] {
    const triggered: CanaryToken[] = [];

    for (const [token, canary] of this.tokens) {
      if (text.includes(token) && !canary.triggered) {
        canary.triggered = true;
        canary.triggeredAt = new Date().toISOString();
        triggered.push(canary);
      }
    }

    return triggered;
  }

  getActive(): CanaryToken[] {
    return [...this.tokens.values()].filter((c) => !c.triggered);
  }

  getTriggered(): CanaryToken[] {
    return [...this.tokens.values()].filter((c) => c.triggered);
  }

  getAll(): CanaryToken[] {
    return [...this.tokens.values()];
  }

  remove(token: string): boolean {
    return this.tokens.delete(token);
  }

  generateInsertionComment(canary: CanaryToken): string {
    return `<!-- ${canary.token} -->`;
  }

  generateInsertionString(canary: CanaryToken): string {
    return canary.token;
  }
}
