#!/usr/bin/env sh
# Clone the official Microsoft BitNet repository into the managed BitCLI backend path.
set -eu
BITCLI_HOME="${BITCLI_HOME:-$HOME/.bitcli}"
TARGET="$BITCLI_HOME/backends/bitnet/current"
mkdir -p "$(dirname "$TARGET")"
if [ ! -d "$TARGET/.git" ]; then
  git clone --recursive https://github.com/microsoft/BitNet.git "$TARGET"
else
  git -C "$TARGET" pull --ff-only
  git -C "$TARGET" submodule update --init --recursive
fi
echo "BitNet backend ready at $TARGET"

