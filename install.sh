
#!/usr/bin/env sh
# DevFlow CLI installer
# Usage: curl -fsSL https://raw.githubusercontent.com/vidipsingh/devflow-backend/main/install.sh | sh

set -e

REPO="vidipsingh/devflow-backend"
BINARY="devflow"
INSTALL_DIR="/usr/local/bin"

# ── Detect OS and arch ────────────────────────────────────────────────────────
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux)
    case "$ARCH" in
      x86_64)  ASSET="devflow-linux-amd64" ;;
      aarch64) ASSET="devflow-linux-arm64" ;;
      *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
    esac
    ;;
  Darwin)
    case "$ARCH" in
      x86_64) ASSET="devflow-darwin-amd64" ;;
      arm64)  ASSET="devflow-darwin-arm64" ;;
      *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
    esac
    ;;
  *)
    echo "Unsupported OS: $OS"
    echo "For Windows, download devflow-windows-amd64.exe from:"
    echo "  https://github.com/$REPO/releases/latest"
    exit 1
    ;;
esac

# ── Get latest release tag ────────────────────────────────────────────────────
echo "Fetching latest DevFlow CLI release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' \
  | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
  echo "Could not determine latest release. Check https://github.com/$REPO/releases"
  exit 1
fi

echo "Installing DevFlow CLI $LATEST ($ASSET)..."

# ── Download binary ───────────────────────────────────────────────────────────
TMP="$(mktemp)"
curl -fsSL "https://github.com/$REPO/releases/download/$LATEST/$ASSET" -o "$TMP"
chmod +x "$TMP"

# ── Install ───────────────────────────────────────────────────────────────────
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP" "$INSTALL_DIR/$BINARY"
else
  echo "Installing to $INSTALL_DIR (may require sudo password)..."
  sudo mv "$TMP" "$INSTALL_DIR/$BINARY"
fi

# ── Verify ────────────────────────────────────────────────────────────────────
echo ""
echo "✓ DevFlow CLI installed at $INSTALL_DIR/$BINARY"
echo ""
echo "Get started:"
echo "  devflow init"
echo ""
