#!/usr/bin/env sh
# BitCLI Uninstaller for Linux and macOS
# Removes $HOME/.bitcli/ entirely — deletes the binary, all models, tools,
# backend, database, config, and chat history.
#
# Usage:
#   sh scripts/uninstall.sh

set -eu

BITCLI_HOME="${BITCLI_HOME:-$HOME/.bitcli}"

printf '\n  \033[1mBitCLI Uninstaller\033[0m\n'
printf '  ─────────────────────────────────────────────────────────\n'
printf '  This will permanently delete:\n'
printf '    \033[33m%s\033[0m\n' "$BITCLI_HOME"
printf '\n'
printf '  This includes the bitcli binary, all downloaded models,\n'
printf '  tools (cmake, clang, uv), the BitNet backend clone,\n'
printf '  database, config, and chat history.\n\n'

printf '  Type YES to confirm uninstall: '
read -r confirm

if [ "$confirm" != "YES" ]; then
    printf '  Uninstall cancelled.\n\n'
    exit 0
fi

printf '\n  Removing %s ...\n' "$BITCLI_HOME"

if [ -d "$BITCLI_HOME" ]; then
    rm -rf "$BITCLI_HOME"
    printf '  \033[32m✓\033[0m  Removed %s\n' "$BITCLI_HOME"
else
    printf '  \033[33m!\033[0m  %s does not exist — nothing to remove.\n' "$BITCLI_HOME"
fi

printf '\n  \033[32mBitCLI has been completely removed.\033[0m\n'
printf '  If you added env.sh to ~/.bashrc or ~/.zshrc, remove that line manually.\n\n'
