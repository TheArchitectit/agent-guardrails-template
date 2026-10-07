# CLAUDE Guardrails Template (Contest Integration)

This repository includes guardrails for agent orchestration and prompts logging. The changes introduce a central prompt logger that records prompts and responses across workflows to enable auditability in contests.

How to use:
- Import and invoke the logger in your workflow steps around user prompts and assistant responses.
- Use logTurn() to log each turn with a turn index and role.
- Optionally export recent prompts to CSV via the prompt-logger-export utility.

Environment variables:
- PROMPT_LOG_DIR: Directory to store prompt logs (default: ./prompt-logs)
- PROMPT_LOG_FILE: Log file name (default: prompts.log.jsonl)
- PROMPT_LOG_MASK: Regex pattern string to redact sensitive content before logging

Usage example in code:
- const promptLogger = require('./utils/prompt-logger');
- promptLogger.logTurn({ turn: 1, role: 'user', text: 'Hello' });
- promptLogger.logTurn({ turn: 2, role: 'assistant', text: 'Hi there' });

For details, see memory entry: prompt-recording-all-workflows.md.