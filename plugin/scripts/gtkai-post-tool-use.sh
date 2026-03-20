#!/bin/sh
# gtkai PostToolUse hook for Claude Code.
# Handles both Bash output compression and MCP passthrough in one pass.
# Requires gtkai in PATH — install via: curl -sSL https://raw.githubusercontent.com/jmeiracorbal/gtk-ai/main/install.sh | sh

command -v gtkai >/dev/null 2>&1 || exit 0

exec gtkai hook-post
