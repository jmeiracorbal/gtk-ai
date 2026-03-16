#!/bin/sh
# gtk-ai installer
# Usage: curl -sSL https://raw.githubusercontent.com/jmeiracorbal/gtk-ai/main/install.sh | sh

set -e

REPO="jmeiracorbal/gtk-ai"
BINARY="gtkai"
INSTALL_DIR="${GTKAI_INSTALL_DIR:-$HOME/.local/bin}"
HOOK_DIR="$HOME/.claude/hooks"
SETTINGS="$HOME/.claude/settings.json"
TMP_DIR=$(mktemp -d)

# ── Colours ───────────────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
RESET='\033[0m'

info()    { printf "${BLUE}  →${RESET} %s\n" "$1"; }
success() { printf "${GREEN}  ✓${RESET} %s\n" "$1"; }
warn()    { printf "${YELLOW}  ⚠${RESET} %s\n" "$1"; }
error()   { printf "${RED}  ✗${RESET} %s\n" "$1" >&2; exit 1; }
header()  { printf "\n${BOLD}%s${RESET}\n" "$1"; }

# ── Banner ────────────────────────────────────────────────────────────────────

printf "${BOLD}"
cat <<'EOF'
   gtk-ai — Go Token Killer
   Claude Code token compression proxy
EOF
printf "${RESET}\n"

# ── Detect OS and architecture ────────────────────────────────────────────────

header "Detecting system"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       error "Unsupported architecture: $ARCH" ;;
esac

case "$OS" in
  linux|darwin) ;;
  *) error "Unsupported OS: $OS" ;;
esac

info "OS:   $OS"
info "Arch: $ARCH"

# ── Detect installation method ────────────────────────────────────────────────

header "Checking dependencies"

HAS_GO=false
HAS_CURL=false
HAS_WGET=false

command -v go    >/dev/null 2>&1 && HAS_GO=true    && success "Go found: $(go version | awk '{print $3}')"
command -v curl  >/dev/null 2>&1 && HAS_CURL=true  && success "curl found"
command -v wget  >/dev/null 2>&1 && HAS_WGET=true

# ── Download helper ───────────────────────────────────────────────────────────

download() {
  url="$1"
  dest="$2"
  if $HAS_CURL; then
    curl -sSL "$url" -o "$dest"
  elif $HAS_WGET; then
    wget -q "$url" -O "$dest"
  else
    error "Neither curl nor wget found. Install one and retry."
  fi
}

# ── Install binary ────────────────────────────────────────────────────────────

header "Installing $BINARY"

mkdir -p "$INSTALL_DIR"

# Try to download pre-built binary from GitHub releases first
RELEASE_URL="https://github.com/$REPO/releases/latest/download/${BINARY}-${OS}-${ARCH}"

if $HAS_CURL || $HAS_WGET; then
  info "Trying pre-built binary..."
  HTTP_CODE=0
  if $HAS_CURL; then
    HTTP_CODE=$(curl -sSL -o "$TMP_DIR/$BINARY" -w "%{http_code}" "$RELEASE_URL" 2>/dev/null || echo 0)
  fi

  if [ "$HTTP_CODE" = "200" ]; then
    chmod +x "$TMP_DIR/$BINARY"
    mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
    success "Binary downloaded from GitHub releases"
  else
    # Fall back to building from source
    info "No pre-built binary found — building from source"

    if ! $HAS_GO; then
      error "Go is required to build from source. Install it from https://go.dev/dl/ and retry."
    fi

    info "Cloning repository..."
    CLONE_DIR="$TMP_DIR/gtk-ai"
    if command -v git >/dev/null 2>&1; then
      git clone --depth 1 "https://github.com/$REPO.git" "$CLONE_DIR" >/dev/null 2>&1
    else
      error "git is required to build from source. Install it and retry."
    fi

    info "Building $BINARY..."
    cd "$CLONE_DIR"
    go build -o "$INSTALL_DIR/$BINARY" ./cmd/gtkai/ 2>/dev/null
    cd - >/dev/null
    success "Built from source"
  fi
fi

# Verify binary works
if ! "$INSTALL_DIR/$BINARY" version >/dev/null 2>&1; then
  error "Binary installed but failed to run. Check $INSTALL_DIR/$BINARY"
fi

INSTALLED_VERSION=$("$INSTALL_DIR/$BINARY" version | awk '{print $2}')
success "$BINARY $INSTALLED_VERSION installed to $INSTALL_DIR/$BINARY"

# ── Add to PATH ───────────────────────────────────────────────────────────────

header "Configuring PATH"

add_to_path() {
  shell_rc="$1"
  if [ -f "$shell_rc" ]; then
    if ! grep -q "$INSTALL_DIR" "$shell_rc" 2>/dev/null; then
      printf '\n# gtk-ai\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >> "$shell_rc"
      success "Added $INSTALL_DIR to PATH in $shell_rc"
    else
      info "$INSTALL_DIR already in $shell_rc"
    fi
  fi
}

case "$SHELL" in
  */zsh)  add_to_path "$HOME/.zshrc"  ;;
  */bash) add_to_path "$HOME/.bashrc" ;;
  *)      add_to_path "$HOME/.profile" ;;
esac

# Ensure binary is reachable right now
export PATH="$INSTALL_DIR:$PATH"

# ── Install Claude Code hook ──────────────────────────────────────────────────

header "Installing Claude Code hook"

mkdir -p "$HOOK_DIR"
HOOK_FILE="$HOOK_DIR/gtkai-post-tool-use.sh"

cat > "$HOOK_FILE" <<'HOOK'
#!/bin/sh
# gtkai PostToolUse hook for Claude Code.
command -v gtkai >/dev/null 2>&1 || exit 0
exec gtkai hook-post
HOOK

chmod +x "$HOOK_FILE"
success "Hook installed: $HOOK_FILE"

# ── Patch ~/.claude/settings.json ─────────────────────────────────────────────

header "Configuring Claude Code"

if ! command -v python3 >/dev/null 2>&1; then
  warn "python3 not found — skipping settings.json patch"
  warn "Add the hook manually to $SETTINGS (see README)"
else
  python3 - "$SETTINGS" "$HOOK_FILE" <<'PYEOF'
import json, sys, os

settings_path = sys.argv[1]
hook_path     = sys.argv[2]

# Load or create settings
if os.path.exists(settings_path):
  try:
    data = json.loads(open(settings_path).read())
  except (json.JSONDecodeError, OSError):
    data = {}
else:
  data = {}

data.setdefault("hooks", {})
data["hooks"].setdefault("PostToolUse", [])

hook_entry = {
  "matcher": "Bash|mcp__.*",
  "hooks": [{"type": "command", "command": hook_path}]
}

# Check if already registered
already = any(
  any(h.get("command") == hook_path for h in e.get("hooks", []))
  for e in data["hooks"]["PostToolUse"]
)

if not already:
  data["hooks"]["PostToolUse"].append(hook_entry)
  os.makedirs(os.path.dirname(settings_path), exist_ok=True)
  open(settings_path, "w").write(json.dumps(data, indent=2) + "\n")
  print("patched")
else:
  print("already_present")
PYEOF

  PATCH_RESULT=$(python3 - "$SETTINGS" "$HOOK_FILE" <<'PYEOF'
import json, sys, os
settings_path = sys.argv[1]
hook_path     = sys.argv[2]
if os.path.exists(settings_path):
  try:
    data = json.loads(open(settings_path).read())
  except: data = {}
else:
  data = {}
data.setdefault("hooks", {})
data["hooks"].setdefault("PostToolUse", [])
hook_entry = {"matcher": "Bash|mcp__.*", "hooks": [{"type": "command", "command": hook_path}]}
already = any(any(h.get("command") == hook_path for h in e.get("hooks", [])) for e in data["hooks"]["PostToolUse"])
if not already:
  data["hooks"]["PostToolUse"].append(hook_entry)
  os.makedirs(os.path.dirname(settings_path), exist_ok=True)
  open(settings_path, "w").write(json.dumps(data, indent=2) + "\n")
print("ok" if not already else "skip")
PYEOF
)

  if [ "$PATCH_RESULT" = "ok" ]; then
    success "Hook registered in $SETTINGS"
  else
    info "Hook already registered in $SETTINGS"
  fi
fi

# ── RTK warning ───────────────────────────────────────────────────────────────

if command -v rtk >/dev/null 2>&1; then
  warn "RTK is installed. To avoid conflicts, remove its hooks from $SETTINGS"
  warn "Look for entries referencing rtk-rewrite.sh or rtk-post-tool-use.sh"
fi

# ── Cleanup ───────────────────────────────────────────────────────────────────

rm -rf "$TMP_DIR"

# ── Done ──────────────────────────────────────────────────────────────────────

printf "\n${BOLD}${GREEN}gtk-ai installed successfully!${RESET}\n\n"
info "Restart Claude Code to activate the hook"
info "Run 'gtkai gain' to track token savings"
printf "\n"
