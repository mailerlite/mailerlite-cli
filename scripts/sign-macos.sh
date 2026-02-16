#!/bin/bash
set -euo pipefail

OS="$1"
BINARY="$2"
IS_SNAPSHOT="$3"

if [ "$OS" != "darwin" ]; then
  echo "skipping signing for $OS"
  exit 0
fi

quill sign-and-notarize "$BINARY" --dry-run="$IS_SNAPSHOT" --ad-hoc="$IS_SNAPSHOT"
