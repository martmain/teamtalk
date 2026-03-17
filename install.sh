#!/bin/sh
set -e

REPO="Higangssh/teamtalk"
BINARY="teamtalk"
INSTALL_DIR="/usr/local/bin"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Try go install first
if command -v go >/dev/null 2>&1; then
    echo "Installing via go install..."
    go install "github.com/$REPO@latest"
    echo "✅ Installed! Run: teamtalk --demo"
    exit 0
fi

echo "Go not found. Install Go first: https://go.dev/dl/"
echo "Then run: go install github.com/$REPO@latest"
exit 1
