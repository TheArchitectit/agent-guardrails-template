#!/usr/bin/env node

/**
 * pi-guardrails installer
 *
 * Copies the npm package contents to ~/.pi/agent/extensions/pi-guardrails.
 *
 * Usage:
 *   npx @thearchitectit/pi-guardrails          # Install or update extension
 *   npx @thearchitectit/pi-guardrails --remove  # Remove the extension
 *   npx @thearchitectit/pi-guardrails --help     # Show this help
 */

import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const PACKAGE_DIR = path.dirname(__filename);
const EXTENSION_DIR = path.join(os.homedir(), ".pi", "agent", "extensions", "pi-guardrails");

const pkg = JSON.parse(fs.readFileSync(path.join(PACKAGE_DIR, "package.json"), "utf-8"));
const VERSION = pkg.version;

const args = process.argv.slice(2);
const isRemove = args.includes("--remove") || args.includes("-r");
const isHelp = args.includes("--help") || args.includes("-h");

if (isHelp) {
  console.log(`
pi-guardrails v${VERSION} - Four Laws guardrails enforcement for pi coding agent

Usage:
  npx @thearchitectit/pi-guardrails          Install or update extension
  npx @thearchitectit/pi-guardrails --remove  Remove the extension
  npx @thearchitectit/pi-guardrails --help     Show this help

Extension directory: ${EXTENSION_DIR}
`);
  process.exit(0);
}

// ─── Extension remove ────────────────────────────────────────────────

if (isRemove) {
  if (fs.existsSync(EXTENSION_DIR)) {
    fs.rmSync(EXTENSION_DIR, { recursive: true });
    console.log("Removed pi-guardrails from " + EXTENSION_DIR);
  } else {
    console.log("pi-guardrails is not installed");
  }
  process.exit(0);
}

// Already running from the extension dir (e.g. local dev)
if (path.resolve(PACKAGE_DIR) === path.resolve(EXTENSION_DIR)) {
  console.log(`Already installed at ${EXTENSION_DIR} (v${VERSION})`);
  process.exit(0);
}

const isUpdate = fs.existsSync(EXTENSION_DIR);

// Clean slate for updates so removed files don't linger between versions
if (isUpdate) {
  fs.rmSync(EXTENSION_DIR, { recursive: true });
}

const SKIP = new Set([".git", "node_modules", ".DS_Store"]);

function copyDir(src, dest) {
  fs.mkdirSync(dest, { recursive: true });
  for (const entry of fs.readdirSync(src, { withFileTypes: true })) {
    if (SKIP.has(entry.name)) continue;
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);
    if (entry.isDirectory()) {
      copyDir(srcPath, destPath);
    } else {
      fs.copyFileSync(srcPath, destPath);
    }
  }
}

copyDir(PACKAGE_DIR, EXTENSION_DIR);

// Ensure sessions subdirectory exists
fs.mkdirSync(path.join(EXTENSION_DIR, "sessions"), { recursive: true });

const action = isUpdate ? "Updated" : "Installed";
console.log(`${action} pi-guardrails v${VERSION} → ${EXTENSION_DIR}

Tools:    guardrail_init, guardrail_record_read, guardrail_verify_read,
          guardrail_set_scope, guardrail_check_scope, guardrail_record_attempt,
          guardrail_check_strikes, guardrail_reset_strikes, guardrail_check_halt,
          guardrail_log_violation, guardrail_status, guardrail_mcp
Commands: /guardrails (Sprint 1)
Docs:     ${EXTENSION_DIR}/README.md`);
