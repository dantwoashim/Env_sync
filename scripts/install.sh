#!/usr/bin/env bash
# EnvSync installer for macOS and Linux
# Usage: curl -fsSL https://envsync.dev/install.sh | bash

set -euo pipefail

REPO="envsync/envsync"
INSTALL_DIR="/usr/local/bin"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  *) echo "Unsupported OS: $OS (use install.ps1 for Windows)"; exit 1 ;;
esac

echo "  ✦ Installing EnvSync for ${OS}/${ARCH}"

# Get latest version
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "  ✗ Failed to get latest version"
  exit 1
fi

echo "  ▸ Version: v${VERSION}"

# Download
FILENAME="envsync_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${FILENAME}"

echo "  ▸ Downloading ${FILENAME}..."
TMP=$(mktemp -d)
curl -fsSL "$URL" -o "${TMP}/${FILENAME}"

# Extract
tar -xzf "${TMP}/${FILENAME}" -C "$TMP"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP}/envsync" "${INSTALL_DIR}/envsync"
else
  echo "  ▸ Requires sudo to install to ${INSTALL_DIR}"
  sudo mv "${TMP}/envsync" "${INSTALL_DIR}/envsync"
fi

chmod +x "${INSTALL_DIR}/envsync"

# Cleanup
rm -rf "$TMP"

echo "  ✓ Installed envsync v${VERSION} to ${INSTALL_DIR}/envsync"
echo ""
echo "  Get started:"
echo "    envsync init"
echo ""
