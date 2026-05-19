import * as fs from "node:fs";
import * as path from "node:path";

interface LanguageRuleFile {
  language: string;
  version: string;
  detectors: string[];
  rules: Array<{
    id: string;
    name: string;
    pattern: string;
    severity: "warning" | "critical";
    message: string;
    filePatterns?: string[];
  }>;
}

export interface LanguageProfile {
  languages: string[];
  detectedBy: Record<string, string[]>;
  ruleCount: number;
}

const EXTENSION_MAP: Record<string, string> = {
  ".py": "python",
  ".js": "typescript",
  ".jsx": "typescript",
  ".ts": "typescript",
  ".tsx": "typescript",
  ".go": "go",
  ".rs": "rust",
};

const CONFIG_FILE_MAP: Record<string, string> = {
  "requirements.txt": "python",
  "pyproject.toml": "python",
  "setup.py": "python",
  "Pipfile": "python",
  "tsconfig.json": "typescript",
  "package.json": "typescript",
  "go.mod": "go",
  "go.sum": "go",
  "Cargo.toml": "rust",
  "Cargo.lock": "rust",
};

const LANGUAGE_DIRS: Record<string, string> = {
  python: "python",
  typescript: "typescript",
  go: "go",
  rust: "rust",
};

export class LanguageDetector {
  private detectedLanguages: string[] = [];
  private detectedBy: Record<string, string[]> = {};
  private scanned = false;

  detectLanguages(cwd: string): LanguageProfile {
    this.detectedLanguages = [];
    this.detectedBy = {};
    this.scanned = true;

    const langSet = new Set<string>();

    // Scan for config files (strong signal)
    for (const [file, lang] of Object.entries(CONFIG_FILE_MAP)) {
      if (fs.existsSync(path.join(cwd, file))) {
        langSet.add(lang);
        if (!this.detectedBy[lang]) this.detectedBy[lang] = [];
        this.detectedBy[lang].push(file);
      }
    }

    // Scan for language-specific directories
    const dirMap: Record<string, string> = {
      node_modules: "typescript",
      venv: "python",
      ".venv": "python",
      __pycache__: "python",
    };
    for (const [dir, lang] of Object.entries(dirMap)) {
      if (fs.existsSync(path.join(cwd, dir))) {
        langSet.add(lang);
        if (!this.detectedBy[lang]) this.detectedBy[lang] = [];
        this.detectedBy[lang].push(`${dir}/`);
      }
    }

    // Sample source files (walk up to 2 levels deep)
    this.scanDir(cwd, langSet, 0, 2);

    this.detectedLanguages = [...langSet].sort();

    // Count rules available for detected languages
    let ruleCount = 0;
    for (const lang of this.detectedLanguages) {
      const dirName = LANGUAGE_DIRS[lang];
      if (!dirName) continue;
      const rulePath = path.join(cwd, ".guardrails", "prevention-rules", "languages", `${dirName}.json`);
      try {
        if (fs.existsSync(rulePath)) {
          const raw = fs.readFileSync(rulePath, "utf-8");
          const parsed = JSON.parse(raw) as LanguageRuleFile;
          ruleCount += parsed.rules?.length ?? 0;
        }
      } catch {
        // Skip unreadable rule files
      }
    }

    return this.getProfile(ruleCount);
  }

  private scanDir(dir: string, langSet: Set<string>, depth: number, maxDepth: number): void {
    if (depth > maxDepth) return;
    let entries: fs.Dirent[];
    try {
      entries = fs.readdirSync(dir, { withFileTypes: true });
    } catch {
      return;
    }

    for (const entry of entries) {
      if (entry.name.startsWith(".") || entry.name === "node_modules" || entry.name === "vendor") continue;

      if (entry.isFile()) {
        const ext = path.extname(entry.name);
        const lang = EXTENSION_MAP[ext];
        if (lang) {
          langSet.add(lang);
          if (!this.detectedBy[lang]) this.detectedBy[lang] = [];
          this.detectedBy[lang].push(`*${ext}`);
        }
      } else if (entry.isDirectory()) {
        this.scanDir(path.join(dir, entry.name), langSet, depth + 1, maxDepth);
      }
    }
  }

  loadLanguageRules(cwd: string, languages?: string[]): { id: string; description: string; pattern: string; severity: "warning" | "critical"; filePatterns?: string[] }[] {
    const langs = languages ?? this.detectedLanguages;
    if (langs.length === 0) this.detectLanguages(cwd);

    const rules: { id: string; description: string; pattern: string; severity: "warning" | "critical"; filePatterns?: string[] }[] = [];

    for (const lang of langs) {
      const dirName = LANGUAGE_DIRS[lang];
      if (!dirName) continue;

      const rulePath = path.join(cwd, ".guardrails", "prevention-rules", "languages", `${dirName}.json`);
      try {
        if (!fs.existsSync(rulePath)) continue;
        const raw = fs.readFileSync(rulePath, "utf-8");
        const parsed = JSON.parse(raw) as LanguageRuleFile;

        for (const rule of parsed.rules ?? []) {
          rules.push({
            id: rule.id,
            description: rule.message ?? rule.name,
            pattern: rule.pattern,
            severity: rule.severity,
            filePatterns: rule.filePatterns,
          });
        }
      } catch {
        // Skip malformed rule files
      }
    }

    return rules;
  }

  getProfile(ruleCount?: number): LanguageProfile {
    return {
      languages: [...this.detectedLanguages],
      detectedBy: { ...this.detectedBy },
      ruleCount: ruleCount ?? 0,
    };
  }

  getDetectedLanguages(): string[] {
    return [...this.detectedLanguages];
  }

  isScanned(): boolean {
    return this.scanned;
  }
}
