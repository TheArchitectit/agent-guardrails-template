---
id: language-detection
name: Language Detection
description: Auto-detects project languages and applies language-specific prevention rules
version: 1.0.0
tags: [safety, patterns, languages]
applies_to: [claude, cursor, opencode, openclaw, windsurf, copilot, pi]
author: TheArchitectit
tools: [Read, Grep, Glob]
globs: "**/*"
alwaysApply: false
---

# Language Detection

Automatically detect project languages and apply language-specific prevention rules.

## The Rule

**Know what language you're working in and apply its safety patterns.**

Language detection is determined by:
1. Config files (requirements.txt, go.mod, Cargo.toml, tsconfig.json, etc.)
2. Source file extensions (.py, .ts, .go, .rs)
3. Project structure indicators (node_modules/, venv/, etc.)

## Supported Languages

| Language | Config Signals | Rules |
|----------|---------------|-------|
| Python | requirements.txt, pyproject.toml, setup.py, Pipfile | eval/exec, subprocess shell=True, bare except, mutable defaults, pickle, hardcoded secrets, SQL injection, type: ignore |
| TypeScript/JS | tsconfig.json, package.json | any type, non-null assertion, eval/Function, innerHTML, console.log, document.write, hardcoded secrets |
| Go | go.mod, go.sum | ignored errors, panic, SQL concat, os/exec, Close without defer, goroutine without context |
| Rust | Cargo.toml, Cargo.lock | unsafe blocks, unwrap, panic!, todo!/unimplemented!, clone on large types, raw pointer deref |

## How It Works

1. **Detection**: Scans project for config files, source extensions, and directory indicators
2. **Rule Loading**: Loads language-specific patterns from `.guardrails/prevention-rules/languages/`
3. **Checking**: When `guardrail_check_pattern` is called, both generic and language-specific rules are applied
4. **File Filtering**: Rules are filtered by `filePatterns` — Python rules only fire on `.py` files, etc.

## Adding New Languages

Create a JSON file at `.guardrails/prevention-rules/languages/<language>.json`:

```json
{
  "$schema": "./language-rules.schema.json",
  "language": "java",
  "version": "1.0.0",
  "detectors": ["*.java", "pom.xml", "build.gradle"],
  "rules": [
    {
      "id": "java-reflection",
      "name": "Reflection usage",
      "pattern": "\\.getDeclaredMethod\\s*\\(",
      "severity": "warning",
      "message": "Reflection bypasses compile-time checking — verify usage is intentional",
      "filePatterns": ["\\.java$"]
    }
  ]
}
```

## When to Use

- Before writing code in any language — check which patterns to avoid
- After code generation — scan for language-specific antipatterns
- During code review — validate against language-specific safety rules

## Pi Enforcement

When using `@architectit/pi-guardrails`:

- `guardrail_detect_language` — Scan a project directory and return detected languages
- `guardrail_get_language_profile` — Get detected languages + available rules with descriptions
- `guardrail_check_pattern` — Check code content against all loaded rules (generic + language-specific)

Language rules are automatically loaded when `PatternRuleEngine.loadRules()` is called with a `LanguageDetector` configured. No manual setup needed — detection happens on first rule load.
