#!/bin/sh
# Lore installer — downloads the latest release binary for your platform.
# Usage: curl -sSfL https://raw.githubusercontent.com/GreyCoderK/lore/main/install.sh | sh
set -e

REPO="GreyCoderK/lore"
BINARY="lore"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)  OS="Linux" ;;
  darwin) OS="Darwin" ;;
  *)
    echo "Error: unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="x86_64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Error: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# Determine latest version
VERSION=${LORE_VERSION:-$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)}
if [ -z "$VERSION" ]; then
  echo "Error: could not determine latest version." >&2
  echo "  This may be due to GitHub API rate limiting (60 requests/hour without auth)." >&2
  echo "  Workaround: set LORE_VERSION=v1.x.x manually." >&2
  exit 1
fi
# Validate version format to prevent injection via malformed tag names
case "$VERSION" in
  v[0-9]*) ;; # valid
  *)
    echo "Error: invalid version format: $VERSION" >&2
    exit 1
    ;;
esac

ARCHIVE="${BINARY}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

echo "Downloading ${BINARY} ${VERSION} for ${OS}/${ARCH}..."
curl -sSfL "$URL" -o "${WORK_DIR}/${ARCHIVE}"
curl -sSfL "$CHECKSUM_URL" -o "${WORK_DIR}/checksums.txt"

# Verify checksum
EXPECTED=$(grep "  ${ARCHIVE}$" "${WORK_DIR}/checksums.txt" | awk '{print $1}')
if [ -z "$EXPECTED" ]; then
  echo "Warning: archive ${ARCHIVE} not found in checksums.txt — skipping verification" >&2
fi
if [ -n "$EXPECTED" ]; then
  if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL=$(sha256sum "${WORK_DIR}/${ARCHIVE}" | awk '{print $1}')
  elif command -v shasum >/dev/null 2>&1; then
    ACTUAL=$(shasum -a 256 "${WORK_DIR}/${ARCHIVE}" | awk '{print $1}')
  else
    echo "Warning: no sha256 tool found, skipping checksum verification" >&2
    ACTUAL="$EXPECTED"
  fi
  if [ "$ACTUAL" != "$EXPECTED" ]; then
    echo "Error: checksum mismatch" >&2
    echo "  expected: $EXPECTED" >&2
    echo "  got:      $ACTUAL" >&2
    exit 1
  fi
  echo "Checksum verified."
fi

# Extract
tar xzf "${WORK_DIR}/${ARCHIVE}" -C "${WORK_DIR}"

# Install
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
  INSTALL_DIR="${HOME}/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi

install -m 755 "${WORK_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
echo "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"

# Warn if install dir is not in PATH
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo ""
    echo "Note: ${INSTALL_DIR} is not in your PATH."
    echo "  Add it with: export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac

echo "Run 'lore --help' to get started."
