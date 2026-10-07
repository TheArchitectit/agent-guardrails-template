#!/bin/bash
# Post-Execution Hook - Emit prompt logging events

PROMPT_JSON=$1

# Forward the prompt JSON to stdout for now; in production would call the logger API
echo "[PROMPT-LOG] ${PROMPT_JSON}"
exit 0
