#!/bin/sh
# gtkai PostToolUse hook for Claude Code.
# Handles both Bash output compression and MCP passthrough in one pass.
# Install: register in ~/.claude/settings.json under PostToolUse.

command -v gtkai >/dev/null 2>&1 || exit 0

exec gtkai hook-post
