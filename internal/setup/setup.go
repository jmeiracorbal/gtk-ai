// Package setup installs gtk-ai into a Claude Code environment.
//
// Registration order:
//  1. Marketplace     → ~/.claude/settings.json (extraKnownMarketplaces)
//  2. Plugin install  → `claude plugin install -s user gtk-ai@gtk-ai`
//  3. Protocol doc    → ~/.claude/gtk-ai.md
//  4. CLAUDE.md       → append @gtk-ai.md
//
// Hooks are managed by Claude Code's plugin system via hooks/hooks.json.
// The plugin install triggers Claude Code to download the repo and activate
// the PostToolUse hook from hooks/hooks.json via ${CLAUDE_PLUGIN_ROOT}.
package setup

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed templates/gtk-ai.md
var protocolDoc []byte

const claudeMDReference = "@gtk-ai.md"

// Install configures gtk-ai in the local Claude Code environment.
// If dryRun is true, prints what would change without writing anything.
func Install(dryRun bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	protocolPath := filepath.Join(home, ".claude", "gtk-ai.md")
	claudeMDPath := filepath.Join(home, ".claude", "CLAUDE.md")

	if dryRun {
		fmt.Println("gtkai setup --dry-run (no changes will be written)")
	} else {
		fmt.Println("gtkai setup")
	}
	fmt.Println("───────────────────────────────────────")

	if err := previewOrRegisterMarketplace(settingsPath, dryRun); err != nil {
		return fmt.Errorf("settings.json: %w", err)
	}

	if err := previewOrInstallPlugin(dryRun); err != nil {
		return fmt.Errorf("plugin install: %w", err)
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

// ─── Marketplace registration ─────────────────────────────────────────────────

func previewOrRegisterMarketplace(settingsPath string, dryRun bool) error {
	config, err := readJSON(settingsPath)
	if err != nil {
		return err
	}

	var marketplaces map[string]json.RawMessage
	if raw, ok := config["extraKnownMarketplaces"]; ok {
		if err := json.Unmarshal(raw, &marketplaces); err != nil {
			marketplaces = make(map[string]json.RawMessage)
		}
	} else {
		marketplaces = make(map[string]json.RawMessage)
	}

	if _, ok := marketplaces["gtk-ai"]; ok {
		if dryRun {
			fmt.Println("[~/.claude/settings.json] — marketplace gtk-ai already registered")
		} else {
			fmt.Println("✓ ~/.claude/settings.json — marketplace already registered")
		}
		return nil
	}

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

	if dryRun {
		fmt.Println("[~/.claude/settings.json] — would register marketplace gtk-ai")
		return nil
	}

	if err := writeJSON(settingsPath, config); err != nil {
		return err
	}
	fmt.Println("✓ ~/.claude/settings.json — marketplace gtk-ai registered")
	return nil
}

// ─── Plugin install via claude CLI ───────────────────────────────────────────

func previewOrInstallPlugin(dryRun bool) error {
	if dryRun {
		fmt.Println("[claude plugin install] — would run: claude plugin install -s user gtk-ai@gtk-ai")
		return nil
	}

	// Check if already installed
	listOut, _ := exec.Command("claude", "plugin", "list").CombinedOutput()
	if strings.Contains(string(listOut), "gtk-ai@gtk-ai") {
		fmt.Println("✓ plugin gtk-ai — already installed")
		return nil
	}

	cmd := exec.Command("claude", "plugin", "install", "-s", "user", "gtk-ai@gtk-ai")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("claude plugin install failed: %w\n%s", err, string(out))
	}
	fmt.Println("✓ plugin gtk-ai installed via claude plugin install")
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
