// gtk-ai — Go Token Killer
// Claude Code token compression proxy. Modular, zero dependencies.
// Binary: gtkai
package main

import (
	"fmt"
	"os"

	"github.com/jmeiracorbal/gtk-ai/internal/hook"
	"github.com/jmeiracorbal/gtk-ai/modules/gain"

	// Register all built-in modules
	_ "github.com/jmeiracorbal/gtk-ai/modules/find"
	_ "github.com/jmeiracorbal/gtk-ai/modules/git"
	_ "github.com/jmeiracorbal/gtk-ai/modules/grep"
	_ "github.com/jmeiracorbal/gtk-ai/modules/ls"
)

const version = "0.1.2"

func usage() {
	fmt.Fprintf(os.Stderr, `gtkai %s — Go Token Killer

Usage:
  gtkai hook-post            PostToolUse hook — reads stdin, compresses Bash + MCP + Read output
  gtkai gain                 Show token savings analytics
  gtkai version              Print version

Claude Code integration:
  Register as PostToolUse hook in ~/.claude/settings.json:
    matcher: "Bash|mcp__.*|Read"
    command: "gtkai hook-post"

Environment:
  GTK_MCP_PASSTHROUGH_PATTERNS  Comma-separated MCP tool patterns to skip compression
                                 Example: hc_*,my_tool
`, version)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Printf("gtkai %s\n", version)

	case "hook-post":
		modified, err := hook.Run(os.Stdin, os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gtkai hook-post: %v\n", err)
			os.Exit(1)
		}
		_ = modified
		os.Exit(0)

	case "gain":
		t, err := gain.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "gtkai: cannot open gain db: %v\n", err)
			os.Exit(1)
		}
		defer t.Close()
		if err := gain.PrintSummary(t); err != nil {
			fmt.Fprintf(os.Stderr, "gtkai: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "gtkai: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}
