#!/bin/bash
# Pre-Commit Hook - ensure prompt-logger hook presence in flow
if ! grep -R "prompt-logger" -n .; then
  echo "[GUARDRAILS] Warning: prompt-logger integration not detected in this repo commit." >&2
fi
