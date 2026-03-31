#!/bin/bash
# Stop hook — logs session end timestamp.
# No longer relies on a prompt hook for JSON verdicts.

LOG_DIR="$CLAUDE_PROJECT_DIR/.claude/logs"
VERDICT_LOG="$LOG_DIR/verdicts.jsonl"
TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")

mkdir -p "$LOG_DIR"

# Log session end
jq -n \
  --arg ts "$TIMESTAMP" \
  '{timestamp: $ts, event: "session_end"}' \
  >> "$VERDICT_LOG"

exit 0
