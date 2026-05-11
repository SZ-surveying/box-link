#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALL_PATH="/usr/local/bin/box-link"

if [[ -x "$SCRIPT_DIR/box-link" ]]; then
  SOURCE_BIN="$SCRIPT_DIR/box-link"
elif [[ -x "$SCRIPT_DIR/../bin/box-link" ]]; then
  SOURCE_BIN="$SCRIPT_DIR/../bin/box-link"
else
  echo "could not find box-link binary next to install.sh or in ../bin" >&2
  exit 1
fi

install -m 0755 "$SOURCE_BIN" "$INSTALL_PATH"
printf 'installed to %s\n' "$INSTALL_PATH"
