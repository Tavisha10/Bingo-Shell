#!/bin/bash
# myshell installer
# usage: bash install.sh

set -e

BINARY_NAME="myshell"
INSTALL_DIR="/usr/local/bin"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo ""
echo "  myshell installer"
echo "  ─────────────────"
echo ""

# check go is installed
if ! command -v go &>/dev/null; then
  echo "  ✗ Go is not installed. Please install Go first:"
  echo "    https://golang.org/dl/"
  exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo "  ✓ Go found: $GO_VERSION"

# build the binary
echo "  → Building $BINARY_NAME..."
cd "$SOURCE_DIR"

if ! go build -o "$BINARY_NAME" ./...; then
  echo "  ✗ Build failed. Fix errors above and retry."
  exit 1
fi
echo "  ✓ Build successful"

# install to /usr/local/bin
echo "  → Installing to $INSTALL_DIR/$BINARY_NAME..."

if [ -w "$INSTALL_DIR" ]; then
  mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
else
  echo "  (requires sudo for $INSTALL_DIR)"
  sudo mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
fi

chmod +x "$INSTALL_DIR/$BINARY_NAME"
echo "  ✓ Installed to $INSTALL_DIR/$BINARY_NAME"

# create config dir
CONFIG_DIR="$HOME/.config/myshell"
mkdir -p "$CONFIG_DIR/hooks"
echo "  ✓ Config directory: $CONFIG_DIR"

# verify installation
if command -v "$BINARY_NAME" &>/dev/null; then
  echo ""
  echo "  ✓ Installation complete!"
  echo ""
  echo "  Run your shell:"
  echo "    $BINARY_NAME"
  echo ""
  echo "  To set as your default shell:"
  echo "    echo \"\$(which $BINARY_NAME)\" | sudo tee -a /etc/shells"
  echo "    chsh -s \"\$(which $BINARY_NAME)\""
  echo ""
else
  echo ""
  echo "  ✗ Installation check failed — $BINARY_NAME not found in PATH"
  echo "  Make sure $INSTALL_DIR is in your PATH"
  echo "    export PATH=\"\$PATH:$INSTALL_DIR\""
fi