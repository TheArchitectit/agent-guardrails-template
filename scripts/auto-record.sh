#!/usr/bin/env bash
# Record every user prompt automatically
# Append the content to the log file in a standardized format

LOG_FILE="./prompt-logs/prompts.log.jsonl"
mkdir -p "$(dirname "$LOG_FILE")"

# Read the entire stdin (the prompt)
PROMPT=$(cat)

# Create a small JSON entry
LOG_ENTRY=$(printf '{"timestamp": "%s", "role": "user", "prompt": %s}\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$(jq -R . <<<"$PROMPT")")

# Append to log
echo "$LOG_ENTRY" >> "$LOG_FILE"
