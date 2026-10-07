# AGENT GUARDRAILS

This document defines guardrails for agent behavior across workflows, intended to be integrated in the contest environment.

Planned extension: central prompt-logging wrapper to record user prompts and assistant responses at all workflow junctions.

Usage notes:
- The guardrails expect a central prompt-logging module that can be invoked around any workflow step that emits prompts or responses.
- Hooks may be wired into .claude/hooks or into the orchestration logic within scripts/* where prompts are generated.
- Ensure privacy: apply redaction via PROMPT_LOG_MASK, and provide an option to log raw prompts in controlled environments.
