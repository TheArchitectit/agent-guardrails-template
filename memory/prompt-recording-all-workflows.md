---
name: prompt-recording-all-workflows
description: Document how to record prompts across all workflows in the contest
metadata:
  type: reference
---

Overview:
- Introduce a central prompt-logger that is used by all workflows to record prompts and responses.

Guiding principles:
- Redact sensitive content before logging using PROMPT_LOG_MASK.
- Log both prompts and model responses with a turn index and roles (user/assistant).
- Enable export to CSV for auditability.

Implementation sketch:
- Implement a lightweight prompt-logger module (JS/TS) that exposes: logTurn({turn, role, text}), and getLogs() to retrieve entries.
- Instrument all prompts in workflows by wrapping around the call that emits prompts and responses.
- Persist logs to a file under prompt-logs/prompts.log.jsonl.
- Optional: periodically export last N prompts to CSV via prompt-logger-export.js.

Usage example:
- const logger = require('./utils/prompt-logger');
- logger.logTurn({ turn: 1, role: 'user', text: 'Hello' });
- logger.logTurn({ turn: 2, role: 'assistant', text: 'Hi there' });

Notes:
- Ensure log file permissions are secure and ensure redaction is enforced.
