import type { Attempt } from "../types.js";

export class StrikeCounter {
  private strikes = new Map<string, Attempt[]>();
  private maxStrikes: number;

  constructor(maxStrikes = 3) {
    this.maxStrikes = maxStrikes;
  }

  recordAttempt(task: string, success: boolean, error?: string): { strikeCount: number; maxReached: boolean } {
    let attempts = this.strikes.get(task);
    if (!attempts) {
      attempts = [];
      this.strikes.set(task, attempts);
    }

    attempts.push({ success, error, timestamp: new Date().toISOString() });

    const failedAttempts = attempts.filter((a) => !a.success);
    const consecutiveFailures = this.countConsecutiveFailures(attempts);
    const strikeCount = consecutiveFailures;

    return { strikeCount, maxReached: strikeCount >= this.maxStrikes };
  }

  getStrikes(task: string): { strikeCount: number; maxReached: boolean; details: Attempt[] } {
    const attempts = this.strikes.get(task) ?? [];
    const consecutiveFailures = this.countConsecutiveFailures(attempts);
    return {
      strikeCount: consecutiveFailures,
      maxReached: consecutiveFailures >= this.maxStrikes,
      details: attempts,
    };
  }

  reset(task: string): boolean {
    return this.strikes.delete(task);
  }

  getAllStrikes(): Map<string, Attempt[]> {
    return this.strikes;
  }

  getMaxStrikes(): number {
    return this.maxStrikes;
  }

  private countConsecutiveFailures(attempts: Attempt[]): number {
    let count = 0;
    for (let i = attempts.length - 1; i >= 0; i--) {
      if (!attempts[i].success) {
        count++;
      } else {
        break;
      }
    }
    return count;
  }

  toJSON(): Record<string, { attempts: Attempt[] }> {
    const obj: Record<string, { attempts: Attempt[] }> = {};
    for (const [task, attempts] of this.strikes) {
      obj[task] = { attempts };
    }
    return obj;
  }

  static fromJSON(data: Record<string, { attempts: Attempt[] }>, maxStrikes = 3): StrikeCounter {
    const counter = new StrikeCounter(maxStrikes);
    for (const [task, val] of Object.entries(data)) {
      counter.strikes.set(task, val.attempts);
    }
    return counter;
  }
}
