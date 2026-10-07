#!/bin/bash
# Pre-execution hook to ensure automatic prompt recording
# This intercepts stdin and logs it

# Pipe content to the logger
tee >(json_entry=$(printf '{"timestamp": "%s", "role": "user", "prompt": %s}\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$(jq -Rn --arg x "$(cat -)" '$x')") ; echo "$json_entry" >> "./prompt-logs/prompts.log.jsonl") | ./scripts/auto-record.sh
