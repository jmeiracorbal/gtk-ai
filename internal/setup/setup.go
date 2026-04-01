// Package setup installs gtk-ai into a Claude Code environment.
//
// Registration order:
//  1. Hook script       → ~/.claude/hooks/gtkai-post-tool-use.sh
//  2. PostToolUse       → ~/.claude/settings.json
//  3. Plugin registry   → ~/.claude/settings.json (enabledPlugins + extraKnownMarketplaces)
//  4. Protocol doc      → ~/.claude/gtk-ai.md
//  5. CLAUDE.md         → append @gtk-ai.md
package setup

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed hooks/gtkai-post-tool-use.sh
var hookScript []byte

//go:embed templates/gtk-ai.md
var protocolDoc []byte

const (
	claudeMDReference = "@gtk-ai.md"
	hookMatcher       = "Bash|mcp__.*|Read"
	hookScriptName    = "gtkai-post-tool-use.sh"
)

// Install configures gtk-ai in the local Claude Code environment.
// If dryRun is true, prints what would change without writing anything.
func Install(dryRun bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}

	hooksDir := filepath.Join(home, ".claude", "hooks")
	hookPath := filepath.Join(hooksDir, hookScriptName)
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	protocolPath := filepath.Join(home, ".claude", "gtk-ai.md")
	claudeMDPath := filepath.Join(home, ".claude", "CLAUDE.md")

	if dryRun {
		fmt.Println("gtkai setup --dry-run (no changes will be written)")
	} else {
		fmt.Println("gtkai setup")
	}
	fmt.Println("───────────────────────────────────────")

	if err := previewOrWriteHook(hookPath, dryRun); err != nil {
		return fmt.Errorf("hook: %w", err)
	}

	if err := previewOrInjectSettingsAndPlugin(settingsPath, hookPath, dryRun); err != nil {
		return fmt.Errorf("settings.json: %w", err)
	}

	if err := previewOrWriteProtocol(protocolPath, dryRun); err != nil {
		return fmt.Errorf("gtk-ai.md: %w", err)
	}

	if err := previewOrInjectCLAUDEMD(claudeMDPath, dryRun); err != nil {
		return fmt.Errorf("CLAUDE.md: %w", err)
	}

	if !dryRun {
		fmt.Println("\nDone. Restart Claude Code to activate gtk-ai.")
	}
	return nil
}

// ─── Hook script ──────────────────────────────────────────────────────────────

func previewOrWriteHook(path string, dryRun bool) error {
	if dryRun {
		fmt.Printf("[~/.claude/hooks/] — would write:\n  %s\n", hookScriptName)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, hookScript, 0755); err != nil {
		return err
	}
	fmt.Printf("✓ hook written to %s\n", path)
	return nil
}

// ─── settings.json (hook + plugin registry) ──────────────────────────────────

func previewOrInjectSettingsAndPlugin(settingsPath, hookPath string, dryRun bool) error {
	config, err := readJSON(settingsPath)
	if err != nil {
		return err
	}

	changed := false

	// PostToolUse hook
	hooks, err := unmarshalHooks(config)
	if err != nil {
		return err
	}
	if !hookAlreadyRegistered(hooks, hookPath) {
		if err := appendPostToolUseHook(hooks, hookPath); err != nil {
			return err
		}
		encoded, err := json.Marshal(hooks)
		if err != nil {
			return err
		}
		config["hooks"] = encoded
		changed = true
	}

	// enabledPlugins
	var enabledPlugins map[string]json.RawMessage
	if raw, ok := config["enabledPlugins"]; ok {
		if err := json.Unmarshal(raw, &enabledPlugins); err != nil {
			enabledPlugins = make(map[string]json.RawMessage)
		}
	} else {
		enabledPlugins = make(map[string]json.RawMessage)
	}
	if _, ok := enabledPlugins["gtk-ai@gtk-ai"]; !ok {
		enabledPlugins["gtk-ai@gtk-ai"] = json.RawMessage("true")
		raw, _ := json.Marshal(enabledPlugins)
		config["enabledPlugins"] = raw
		changed = true
	}

	// extraKnownMarketplaces
	var marketplaces map[string]json.RawMessage
	if raw, ok := config["extraKnownMarketplaces"]; ok {
		if err := json.Unmarshal(raw, &marketplaces); err != nil {
			marketplaces = make(map[string]json.RawMessage)
		}
	} else {
		marketplaces = make(map[string]json.RawMessage)
	}
	if _, ok := marketplaces["gtk-ai"]; !ok {
		entry := map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "jmeiracorbal/gtk-ai",
			},
		}
		entryRaw, _ := json.Marshal(entry)
		marketplaces["gtk-ai"] = entryRaw
		mRaw, _ := json.Marshal(marketplaces)
		config["extraKnownMarketplaces"] = mRaw
		changed = true
	}

	if !changed {
		if dryRun {
			fmt.Println("\n[~/.claude/settings.json] — already up to date")
		} else {
			fmt.Println("✓ ~/.claude/settings.json — already up to date")
		}
		return nil
	}

	if dryRun {
		out, _ := json.MarshalIndent(config, "", "  ")
		fmt.Printf("\n[~/.claude/settings.json]\n%s\n", string(out))
		return nil
	}

	if err := writeJSON(settingsPath, config); err != nil {
		return err
	}
	fmt.Println("✓ ~/.claude/settings.json updated (hook + plugin registry)")
	return nil
}

func unmarshalHooks(config map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	hooks := make(map[string]json.RawMessage)
	if raw, ok := config["hooks"]; ok {
		if err := json.Unmarshal(raw, &hooks); err != nil {
			return nil, err
		}
	}
	return hooks, nil
}

func hookAlreadyRegistered(hooks map[string]json.RawMessage, hookPath string) bool {
	raw, ok := hooks["PostToolUse"]
	if !ok {
		return false
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return false
	}
	for _, entry := range entries {
		hooksRaw, ok := entry["hooks"]
		if !ok {
			continue
		}
		var hs []map[string]json.RawMessage
		if err := json.Unmarshal(hooksRaw, &hs); err != nil {
			continue
		}
		for _, h := range hs {
			cmdRaw, ok := h["command"]
			if !ok {
				continue
			}
			var cmd string
			if err := json.Unmarshal(cmdRaw, &cmd); err == nil && cmd == hookPath {
				return true
			}
		}
	}
	return false
}

func appendPostToolUseHook(hooks map[string]json.RawMessage, hookPath string) error {
	entry := map[string]interface{}{
		"matcher": hookMatcher,
		"hooks": []map[string]interface{}{
			{"type": "command", "command": hookPath},
		},
	}

	var existing []interface{}
	if raw, ok := hooks["PostToolUse"]; ok {
		if err := json.Unmarshal(raw, &existing); err != nil {
			existing = nil
		}
	}
	existing = append(existing, entry)

	raw, err := json.Marshal(existing)
	if err != nil {
		return err
	}
	hooks["PostToolUse"] = raw
	return nil
}

// ─── gtk-ai.md ────────────────────────────────────────────────────────────────

func previewOrWriteProtocol(path string, dryRun bool) error {
	if dryRun {
		fmt.Printf("\n[~/.claude/gtk-ai.md]\n%s\n", string(protocolDoc))
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, protocolDoc, 0644); err != nil {
		return err
	}
	fmt.Println("✓ ~/.claude/gtk-ai.md written")
	return nil
}

// ─── CLAUDE.md ────────────────────────────────────────────────────────────────

func previewOrInjectCLAUDEMD(path string, dryRun bool) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		data = []byte{}
	} else if err != nil {
		return err
	}

	content := string(data)
	if strings.Contains(content, claudeMDReference) {
		if dryRun {
			fmt.Printf("\n[~/.claude/CLAUDE.md] — already contains %s, no change needed\n", claudeMDReference)
		} else {
			fmt.Println("✓ ~/.claude/CLAUDE.md — already up to date")
		}
		return nil
	}

	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	newContent := content + claudeMDReference + "\n"

	if dryRun {
		fmt.Printf("\n[~/.claude/CLAUDE.md] — would append:\n  %s\n", claudeMDReference)
		return nil
	}

	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return err
	}
	fmt.Println("✓ ~/.claude/CLAUDE.md updated")
	return nil
}

// ─── JSON helpers ─────────────────────────────────────────────────────────────

func readJSON(path string) (map[string]json.RawMessage, error) {
	config := make(map[string]json.RawMessage)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return config, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return config, nil
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return config, nil
}

func writeJSON(path string, config map[string]json.RawMessage) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0644)
}
