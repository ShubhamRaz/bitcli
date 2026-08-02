#!/usr/bin/env sh
# BitCLI Installer for Linux and macOS
# Usage:  curl -fsSL https://raw.githubusercontent.com/ShubhamRaz/bitcli/main/scripts/install.sh | sh
#
# Requires: git, go (https://go.dev/dl/)
# Installs into: $HOME/.bitcli/
# To uninstall:  rm -rf "$HOME/.bitcli"

set -eu

BITCLI_HOME="${BITCLI_HOME:-$HOME/.bitcli}"
BITCLI_BIN="$BITCLI_HOME/bin"
BINARY="$BITCLI_BIN/bitcli"
GITHUB_REPO="ShubhamRaz/bitcli"

# ── Helpers ──────────────────────────────────────────────────────────────────
step()   { printf '\n  \033[36m==>\033[0m %s\n' "$*"; }
ok()     { printf '  \033[32m[OK]\033[0m %s\n' "$*"; }
warn()   { printf '  \033[33m[!]\033[0m  %s\n' "$*"; }
die()    { printf '\n  \033[31mERROR:\033[0m %s\n\n' "$*" >&2; exit 1; }

need_cmd() {
    command -v "$1" >/dev/null 2>&1 || die "$1 is required but not found. Please install it and re-run."
}

# ── Banner ───────────────────────────────────────────────────────────────────
printf '\n  \033[1mBitCLI Installer (build from source)\033[0m\n'
printf '  -------------------------------------------------------\n'
printf '  Install dir : %s\n' "$BITCLI_HOME"
printf '  Uninstall   : rm -rf "%s"\n' "$BITCLI_HOME"

# ── Step 1: Check prerequisites ──────────────────────────────────────────────
step "Checking prerequisites"
need_cmd git
ok "git found"
need_cmd go
ok "go found ($(go version))"

# ── Step 2: Create directory layout ──────────────────────────────────────────
step "Creating directory layout"
mkdir -p "$BITCLI_BIN" "$BITCLI_HOME/models" "$BITCLI_HOME/tools"
ok "Created $BITCLI_HOME"

# ── Step 3: Clone source ─────────────────────────────────────────────────────
step "Cloning source from GitHub"
TMP_SRC="$(mktemp -d)"
git clone --depth=1 "https://github.com/$GITHUB_REPO.git" "$TMP_SRC"
ok "Cloned to $TMP_SRC"

# ── Step 4: Build binary ─────────────────────────────────────────────────────
step "Building bitcli (~30 seconds)"
(cd "$TMP_SRC" && go build -buildvcs=false -o "$BINARY" ./cmd/bitcli)
chmod +x "$BINARY"
rm -rf "$TMP_SRC"
ok "Built bitcli at $BINARY"

# ── Step 5: Update PATH for this session ─────────────────────────────────────
step "Updating PATH for this session"
export PATH="$BITCLI_BIN:$PATH"
ok "PATH updated"

# ── Step 6: Run bitcli setup ─────────────────────────────────────────────────
step "Running bitcli setup"
printf '  Downloads cmake, clang, uv, and clones the BitNet backend.\n'
printf '  May take several minutes on first run.\n\n'
"$BINARY" setup

# ── Done ─────────────────────────────────────────────────────────────────────
printf '\n  -------------------------------------------------------\n'
printf '  \033[32mBitCLI installed successfully!\033[0m\n\n'
printf '  Quick start:\n'
printf '    \033[33mbitcli doctor\033[0m\n'
printf '    \033[33mbitcli pull microsoft/BitNet-b1.58-2B-4T\033[0m\n'
printf '    \033[33mbitcli run --prompt "Hello!"\033[0m\n\n'
printf '  Permanent PATH — add to ~/.bashrc or ~/.zshrc:\n'
printf '    \033[33m. "%s/env.sh"\033[0m\n\n' "$BITCLI_HOME"
printf '  Uninstall: rm -rf "%s"\n\n' "$BITCLI_HOME"
