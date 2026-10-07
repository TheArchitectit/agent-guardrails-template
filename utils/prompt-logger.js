'use strict';
// Prompt logger for traceability in the contest
// Logs all user prompts (and optional context) to a JSONL file for audit
// Usage: const logger = require('./utils/prompt-logger'); logger.logPrompt({ id, role, prompt, context, extra });

const fs = require('fs');
const path = require('path');

// Configurable via env vars; sensible defaults
const LOG_DIR = process.env.PROMPT_LOG_DIR || path.resolve(__dirname, '../../prompt-logs');
const LOG_FILE = process.env.PROMPT_LOG_FILE || path.join(LOG_DIR, 'prompts.log.jsonl');

function ensureDir(dir) {
  try {
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }
  } catch (e) {
    // If we can't create the directory, fail silently to avoid breaking prompts
    // but log the error to stdout for debugging during development
    // eslint-disable-next-line no-console
    console.error('prompt-logger: failed to ensure log dir', dir, e && e.message ? e.message : e);
  }
}
ensureDir(LOG_DIR);

// Create the log file if missing
try {
  if (!fs.existsSync(LOG_FILE)) {
    fs.writeFileSync(LOG_FILE, '', { flag: 'a' });
  }
} catch (e) {
  // Soft fail
  // eslint-disable-next-line no-console
  console.error('prompt-logger: could not initialize log file', e && e.message ? e.message : e);
}

function _appendLine(line) {
  try {
    fs.appendFileSync(LOG_FILE, line + '\n', 'utf8');
  } catch (e) {
    // Soft fail to avoid breaking the caller
    // eslint-disable-next-line no-console
    console.error('prompt-logger: failed to write log', e && e.message ? e.message : e);
  }
}

/**
 * Log a prompt entry
 * @param {Object} params
 * @param {string} [params.id] - optional id for correlation
 * @param {string} params.role - 'user' | 'system' | 'assistant' etc.
 * @param {string} params.prompt - the prompt text
 * @param {Object} [params.context] - any contextual metadata (optional)
 * @param {Object} [params.extra] - any extra fields (optional)
 */
function logPrompt({ id, role = 'user', prompt, context = {}, extra = {} }) {
  if (!prompt) return null;
  const entry = {
    id: id || `p-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    ts: new Date().toISOString(),
    role,
    prompt,
    context,
    extra
  };
  _appendLine(JSON.stringify(entry));
  return entry.id;
}

function readLastLines(n = 100) {
  try {
    const data = fs.readFileSync(LOG_FILE, 'utf8');
    const lines = data.trim().split('\n').filter(l => l.length > 0);
    return lines.slice(-n).map(l => {
      try { return JSON.parse(l); } catch(e) { return { raw: l }; }
    });
  } catch (e) {
    return [];
  }
}

module.exports = {
  logPrompt,
  getLogPath: () => LOG_FILE,
  readLastLines
};
