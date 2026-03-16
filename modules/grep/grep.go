// Module grep: filters `grep` output for Claude Code.
// Truncates large result sets, summarizes by file.
package grep

import (
	"fmt"
	"strings"

	"github.com/jmeiracorbal/gtk-ai/internal/registry"
)

const maxLines = 200

func init() {
	registry.Register(&Module{})
}

type Module struct{}

func (m *Module) Name() string { return "grep" }

func (m *Module) Rewrite(args []string) ([]string, bool) {
	return nil, false
}

func (m *Module) FilterOutput(output string) string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	var matches []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			matches = append(matches, l)
		}
	}

	total := len(matches)
	if total == 0 {
		return "(no matches)\n"
	}

	// Count matches per file
	byFile := map[string]int{}
	for _, l := range matches {
		file := fileOf(l)
		byFile[file]++
	}

	truncated := false
	shown := matches
	if total > maxLines {
		shown = matches[:maxLines]
		truncated = true
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 %d matches in %d files\n\n", total, len(byFile)))

	for _, l := range shown {
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	if truncated {
		sb.WriteString(fmt.Sprintf("... +%d lines truncated\n", total-maxLines))
	}

	return sb.String()
}

func (m *Module) TokensBefore(output string) int {
	return registry.EstimateTokens(output)
}

func (m *Module) TokensAfter(filtered string) int {
	return registry.EstimateTokens(filtered)
}

// fileOf extracts the filename from a grep -n line (file:line:content).
func fileOf(line string) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}
