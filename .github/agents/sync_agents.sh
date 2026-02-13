#!/usr/bin/env sh

set -eu

DEST_DIR=$(cd "$(dirname "$0")" && pwd)
SRC_DIR="$HOME/projects/trakhimenok/ai/.github/agents/"

rsync -a --exclude "README.md" "$SRC_DIR" "$DEST_DIR"

echo "Copied agents/ into $DEST_DIR (excluding README.md)."
