#!/usr/bin/env sh
# BitCLI Installer for Linux and macOS
# Usage (one-liner):
#   curl -fsSL https://raw.githubusercontent.com/ShubhamRaz/bitcli/main/scripts/install.sh | sh
#
# Everything is installed into $HOME/.bitcli/
# To uninstall completely: rm -rf "$HOME/.bitcli"

set -eu

BITCLI_VERSION="${BITCLI_VERSION:-latest}"
BITCLI_HOME="${BITCLI_HOME:-$HOME/.bitcli}"
BITCLI_BIN="$BITCLI_HOME/bin"
BINARY="$BITCLI_BIN/bitcli"
GITHUB_REPO="ShubhamRaz/bitcli"

# ── Helpers ──────────────────────────────────────────────────────────────────
step()   { printf '\n  \033[36m==>\033[0m %s\n' "$*"; }
ok()     { printf '  \033[32m✓\033[0m  %s\n' "$*"; }
warn()   { printf '  \033[33m!\033[0m  %s\n' "$*"; }
die()    { printf '\n  \033[31mERROR:\033[0m %s\n\n' "$*" >&2; exit 1; }

need_cmd() { command -v "$1" >/dev/null 2>&1 || die "$1 is required but not found. Please install it and re-run."; }

download() {
    local url="$1" dest="$2"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL --progress-bar -o "$dest" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -q --show-progress -O "$dest" "$url"
    else
        die "curl or wget is required to download files."
    fi
}

# ── Banner ───────────────────────────────────────────────────────────────────
printf '\n  \033[1mBitCLI Installer\033[0m\n'
printf '  ─────────────────────────────────────────────────────────\n'
printf '  Install directory: %s\n' "$BITCLI_HOME"
printf '  Everything stays in this folder. To uninstall:\n'
printf '    rm -rf "%s"\n' "$BITCLI_HOME"
printf '\n'

# ── Step 1: Prerequisites ────────────────────────────────────────────────────
step "Checking prerequisites"
need_cmd git

# ── Step 2: Create directory layout ─────────────────────────────────────────
step "Creating BitCLI home at $BITCLI_HOME"
mkdir -p "$BITCLI_BIN" "$BITCLI_HOME/models" "$BITCLI_HOME/tools"
ok "Directory layout created"

# ── Step 3: Detect platform ──────────────────────────────────────────────────
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux)  os_slug="linux" ;;
    Darwin) os_slug="darwin" ;;
    *)      die "Unsupported OS: $OS" ;;
esac

case "$ARCH" in
    x86_64|amd64) arch_slug="x86_64" ;;
    aarch64|arm64) arch_slug="arm64" ;;
    *) die "Unsupported architecture: $ARCH" ;;
esac

ASSET_NAME="bitcli-${os_slug}-${arch_slug}"

# ── Step 4: Download bitcli binary ───────────────────────────────────────────
step "Downloading BitCLI binary"

if [ "$BITCLI_VERSION" = "latest" ]; then
    API_URL="https://api.github.com/repos/$GITHUB_REPO/releases/latest"
    DOWNLOAD_URL=""
    if command -v curl >/dev/null 2>&1; then
        DOWNLOAD_URL=$(curl -fsSL "$API_URL" 2>/dev/null \
            | grep '"browser_download_url"' \
            | grep "$ASSET_NAME" \
            | head -1 \
            | sed 's/.*"browser_download_url": "\([^"]*\)".*/\1/')
    fi
else
    DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/$BITCLI_VERSION/$ASSET_NAME"
fi

if [ -n "$DOWNLOAD_URL" ]; then
    printf '  Downloading from %s...\n' "$DOWNLOAD_URL"
    download "$DOWNLOAD_URL" "$BINARY"
    chmod +x "$BINARY"
    ok "bitcli downloaded to $BINARY"
else
    # Fallback: build from source if Go is available
    warn "No pre-built binary found for $ASSET_NAME."
    if command -v go >/dev/null 2>&1; then
        printf '  Building from source (Go detected)...\n'
        TMP_SRC="$(mktemp -d)"
        git clone --depth=1 "https://github.com/$GITHUB_REPO.git" "$TMP_SRC" 2>&1
        (cd "$TMP_SRC" && go build -buildvcs=false -o "$BINARY" ./cmd/bitcli)
        rm -rf "$TMP_SRC"
        chmod +x "$BINARY"
        ok "bitcli built and installed"
    else
        die "No pre-built binary available and Go is not installed.\nInstall Go from https://go.dev/dl/ then re-run this installer."
    fi
fi

# ── Step 5: Activate PATH for this session ───────────────────────────────────
step "Configuring PATH for this session"
export PATH="$BITCLI_BIN:$PATH"
ok "PATH updated (current session)"

# ── Step 6: Run bitcli setup ─────────────────────────────────────────────────
step "Running bitcli setup (this may take several minutes)"
printf '\n'
"$BINARY" setup

# ── Step 7: Shell profile instructions ───────────────────────────────────────
printf '\n'
printf '  ─────────────────────────────────────────────────────────\n'
printf '  \033[32mBitCLI is installed!\033[0m\n\n'
printf '  To make BitCLI available in all future terminal sessions,\n'
printf '  add the following line to your ~/.bashrc or ~/.zshrc:\n\n'
printf '    \033[33m. "%s/env.sh"\033[0m\n\n' "$BITCLI_HOME"
printf '  To apply immediately in the current shell:\n'
printf '    . "%s/env.sh"\n\n' "$BITCLI_HOME"
printf '  To uninstall everything:\n'
printf '    rm -rf "%s"\n\n' "$BITCLI_HOME"
