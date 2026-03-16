// Module find: filters `find` output for Claude Code.
// Strips noise, summarizes large result sets, groups by extension.
package find

import (
	"fmt"
	"strings"

	"github.com/jmeiracorbal/gtk-ai/internal/registry"
)

const maxLines = 100

func init() {
	registry.Register(&Module{})
}

type Module struct{}

func (m *Module) Name() string { return "find" }

func (m *Module) Rewrite(args []string) ([]string, bool) {
	return nil, false
}

func (m *Module) FilterOutput(output string) string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	var paths []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			paths = append(paths, l)
		}
	}

	total := len(paths)
	if total == 0 {
		return "(no results)\n"
	}

	// Group by extension for summary
	byExt := map[string]int{}
	for _, p := range paths {
		byExt[extOf(p)]++
	}

	truncated := false
	shown := paths
	if total > maxLines {
		shown = paths[:maxLines]
		truncated = true
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📁 %d results", total))
	if truncated {
		sb.WriteString(fmt.Sprintf(" (showing %d)", maxLines))
	}
	sb.WriteString("\n")

	// Extension summary
	if len(byExt) > 0 {
		parts := make([]string, 0, len(byExt))
		for ext, n := range byExt {
			if ext == "" {
				ext = "no-ext"
			}
			parts = append(parts, fmt.Sprintf(".%s(%d)", ext, n))
		}
		sb.WriteString(fmt.Sprintf("ext: %s\n", strings.Join(parts, " ")))
	}

	sb.WriteString("\n")
	for _, p := range shown {
		sb.WriteString(p)
		sb.WriteString("\n")
	}
	if truncated {
		sb.WriteString(fmt.Sprintf("... +%d more\n", total-maxLines))
	}

	return sb.String()
}

func (m *Module) TokensBefore(output string) int {
	return registry.EstimateTokens(output)
}

func (m *Module) TokensAfter(filtered string) int {
	return registry.EstimateTokens(filtered)
}

func extOf(path string) string {
	// Use last component
	slash := strings.LastIndex(path, "/")
	name := path
	if slash >= 0 {
		name = path[slash+1:]
	}
	dot := strings.LastIndex(name, ".")
	if dot < 0 || dot == len(name)-1 {
		return ""
	}
	return name[dot+1:]
}
