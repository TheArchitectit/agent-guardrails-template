import * as path from "node:path";

const MAX_ENTRIES = 2000;

export class FileReadStore {
  private reads = new Map<string, string>();

  record(filePath: string): void {
    const normalized = path.resolve(filePath);
    if (this.reads.size >= MAX_ENTRIES && !this.reads.has(normalized)) {
      const firstKey = this.reads.keys().next().value;
      if (firstKey !== undefined) this.reads.delete(firstKey);
    }
    this.reads.set(normalized, new Date().toISOString());
  }

  wasRead(filePath: string): boolean {
    return this.reads.has(path.resolve(filePath));
  }

  getReadAt(filePath: string): string | null {
    return this.reads.get(path.resolve(filePath)) ?? null;
  }

  clear(): void {
    this.reads.clear();
  }

  get size(): number {
    return this.reads.size;
  }

  toJSON(): Record<string, string> {
    return Object.fromEntries(this.reads);
  }

  static fromJSON(data: Record<string, string>): FileReadStore {
    const store = new FileReadStore();
    for (const [fp, ts] of Object.entries(data)) {
      store.reads.set(fp, ts);
    }
    return store;
  }
}
