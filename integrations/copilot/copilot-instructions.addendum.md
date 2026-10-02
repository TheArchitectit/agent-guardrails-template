## Guardrails MCP (advisory)
These are requests to Copilot, not enforcement. Copilot can ignore them.
- Before editing files, call `guardrail_pre_work_check` and `guardrail_validate_scope` for the target paths.
- Before running a shell command, call `guardrail_validate_bash`. If the verdict is FAIL, UNKNOWN or ERROR, stop and ask the user.
- Before committing or pushing, call `guardrail_validate_commit` / `guardrail_validate_push`.
- Treat UNKNOWN as a stop, never as a pass.
- Never print, log or commit secrets or the MCP API key.
