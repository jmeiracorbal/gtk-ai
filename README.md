# gtk-ai

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go&logoColor=white)
![Version](https://img.shields.io/badge/version-0.1.0-blue?style=flat)
![License](https://img.shields.io/badge/license-Apache%202.0-blue?style=flat)
![Claude Code](https://img.shields.io/badge/Claude%20Code-hook%20compatible-blueviolet?style=flat)
![Build](https://img.shields.io/badge/build-passing-brightgreen?style=flat)

**Go Token Killer**: modular token compression proxy for Claude Code. Single binary, zero dependencies, plug-and-play.

gtkai sits between Claude Code and your shell as a `PostToolUse` hook. It intercepts command output before it reaches the agent, compresses it to the relevant parts, and records the savings. The less Claude reads, the less you pay.

```
Claude → Bash("find . -name '*.rs'")
              ↓ executes
         84 results, full paths (raw output)
              ↓ PostToolUse → gtkai hook-post
         📁 84 results / ext: .rs(84) / ...10 paths shown
              ↓
         Claude receives compressed output (same information, fewer tokens)
```

## Why Go

- **Single binary**: `go build` produces one static executable, no runtime required
- **Instant startup**: hooks run on every tool call; Go's startup time is negligible
- **Easy distribution**: `brew install`, `curl | sh`, or drop the binary anywhere in PATH
- **Modular by design**: each command is an independent Go package that registers itself at init time

## Benchmark

Measured against raw shell output on a real Rust codebase (76 files):

| Command | Raw output | gtkai | Savings |
|---|---|---|---|
| `find . -name "*.rs"` | ~4,200 tokens | ~300 tokens | **~93%** |
| `ls src/` | ~800 tokens | ~120 tokens | **~85%** |
| `git log --oneline` | ~1,200 tokens | ~180 tokens | **~85%** |
| `grep -rn "savings" src/` | ~3,100 tokens | ~340 tokens | **~89%** |

Track your own savings at any time:

```bash
gtkai gain
```

```
gtkai Token Savings
════════════════════════════════════════
Total commands: 261
Tokens in:      265.4K
Tokens out:     66.8K
Tokens saved:   198.8K (74.9%)
Efficiency:     ██████████████████░░ 74.9%
```

## Quickstart

### 1. Install

**Option A: One-line installer (recommended)**

```bash
curl -sSL https://raw.githubusercontent.com/jmeiracorbal/gtk-ai/main/install.sh | sh
```

Detects your OS and architecture, downloads the binary, installs the hook, and patches `~/.claude/settings.json` automatically.

**Option B: Claude Code plugin**

```bash
claude plugin marketplace add jmeiracorbal/gtk-ai
claude plugin install gtk-ai@gtk-ai
```

Registers the PostToolUse hook automatically. Requires `gtkai` in PATH — download the binary for your platform from [GitHub Releases](https://github.com/jmeiracorbal/gtk-ai/releases) and place it in `~/.local/bin/`.

**Option C: Manual install** (requires Go 1.22+):

```bash
git clone https://github.com/jmeiracorbal/gtk-ai
cd gtk-ai
go build -o ~/.local/bin/gtkai ./cmd/gtkai/
```

Verify:

```bash
gtkai version
# gtkai 0.1.0
```

### 2. Register the hook in Claude Code

Add to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Bash|mcp__.*",
        "hooks": [
          {
            "type": "command",
            "command": "~/.claude/hooks/gtkai-post-tool-use.sh"
          }
        ]
      }
    ]
  }
}
```

Copy the hook script:

```bash
cp hooks/gtkai-post-tool-use.sh ~/.claude/hooks/
chmod +x ~/.claude/hooks/gtkai-post-tool-use.sh
```

Restart Claude Code. Done.

## Modules

Each module handles one command. All built-in modules ship with the binary.

| Module | Command | What it compresses |
|---|---|---|
| `find` | `find` | Truncates large result sets, groups by extension, shows summary |
| `ls` | `ls` | Groups files by extension, separates dirs |
| `git` | `git diff/log/status/branch` | Limits diff lines, truncates log, formats status |
| `grep` | `grep` | Caps results, shows match count per file |
| `gain` | — | SQLite analytics: tracks tokens in/out per command |

## Adding a module

1. Create `modules/mycommand/mycommand.go`
2. Implement the `Module` interface
3. Register at `init()` time
4. Import in `cmd/gtkai/main.go`

```go
// modules/mycommand/mycommand.go
package mycommand

import "github.com/jmeiracorbal/gtk-ai/internal/registry"

func init() { registry.Register(&Module{}) }

type Module struct{}

func (m *Module) Name() string                        { return "mycommand" }
func (m *Module) Rewrite(args []string) ([]string, bool) { return nil, false }
func (m *Module) FilterOutput(output string) string   { /* compress here */ return output }
func (m *Module) TokensBefore(output string) int      { return registry.EstimateTokens(output) }
func (m *Module) TokensAfter(filtered string) int     { return registry.EstimateTokens(filtered) }
```

```go
// cmd/gtkai/main.go — add one import
_ "github.com/jmeiracorbal/gtk-ai/modules/mycommand"
```

No other changes needed.

## MCP passthrough

By default, gtkai compresses all `mcp__*` tool responses above 3,000 chars. To exempt specific tools, set `GTK_MCP_PASSTHROUGH_PATTERNS` in your shell configuration file (`.zshrc`, `.bashrc`, `.profile`, or whichever your shell uses):

```sh
export GTK_MCP_PASSTHROUGH_PATTERNS="my_tool_*,other_tool"
```

Pattern syntax: exact name or glob prefix (`prefix_*`).

To identify which tools to exempt, check the tool names returned by your MCP servers — any tool whose output should reach the agent unmodified (e.g. structured symbol data, memory results) should be listed here.

## Commands

```
gtkai hook-post     PostToolUse hook — reads stdin JSON, writes compressed output
gtkai gain          Token savings analytics
gtkai version       Print version
```

## Architecture

```
gtk-ai/
├── cmd/gtkai/              # binary entry point
├── internal/
│   ├── registry/           # Module interface + Register() + EstimateTokens()
│   └── hook/               # unified PostToolUse handler (Bash + MCP)
└── modules/
    ├── find/               # find output filter
    ├── ls/                 # ls output filter
    ├── git/                # git subcommand filters
    ├── grep/               # grep output filter
    └── gain/               # SQLite token savings analytics
```

The `registry` package is the only shared dependency between modules. Modules never import each other.

## Works well with

**[hybrid-coco](https://github.com/jmeiracorbal/hybrid-coco)**: local code intelligence for Claude Code. While gtkai compresses shell output, hybrid-coco replaces file reads and grep with index queries on code navigation. Both operate independently via separate hooks and complement each other.

## License

Apache 2.0, see [LICENSE](LICENSE). Attribution required on redistribution.
