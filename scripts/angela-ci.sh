#!/usr/bin/env bash
# angela-ci.sh — Portable documentation quality gate
# Works with any CI system: GitHub Actions, GitLab CI, Jenkins, Bitbucket, etc.
#
# Usage:
#   ./scripts/angela-ci.sh [OPTIONS]
#
# Options:
#   --path <dir>        Path to markdown docs directory (default: ./docs)
#   --fail-on <level>   Fail on: error, warning, or none (default: error)
#   --install           Auto-install lore if not found
#   --version <ver>     Lore version to install (default: latest)
#   --quiet             Suppress human-readable output
#
# Exit codes:
#   0  All checks passed
#   1  Documentation issues found (at or above --fail-on level)
#   2  Script error (missing dependencies, bad arguments)
#
# Examples:
#   # GitHub Actions
#   - run: ./scripts/angela-ci.sh --path docs --fail-on warning
#
#   # GitLab CI
#   script:
#     - ./scripts/angela-ci.sh --path docs --install
#
#   # Jenkins
#   sh './scripts/angela-ci.sh --path docs --fail-on error'

set -euo pipefail

# --- Defaults ---
DOCS_PATH="./docs"
FAIL_ON="error"
AUTO_INSTALL=false
LORE_VERSION="latest"
QUIET=false

# --- Parse arguments ---
while [[ $# -gt 0 ]]; do
  case "$1" in
    --path)
      DOCS_PATH="$2"
      shift 2
      ;;
    --fail-on)
      FAIL_ON="$2"
      shift 2
      ;;
    --install)
      AUTO_INSTALL=true
      shift
      ;;
    --version)
      LORE_VERSION="$2"
      shift 2
      ;;
    --quiet)
      QUIET=true
      shift
      ;;
    -h|--help)
      head -30 "$0" | grep "^#" | sed 's/^# \?//'
      exit 0
      ;;
    *)
      echo "error: unknown option: $1" >&2
      exit 2
      ;;
  esac
done

# --- Validation ---
if [[ ! "$FAIL_ON" =~ ^(error|warning|none)$ ]]; then
  echo "error: --fail-on must be 'error', 'warning', or 'none'" >&2
  exit 2
fi

if [[ ! -d "$DOCS_PATH" ]]; then
  echo "error: docs directory not found: $DOCS_PATH" >&2
  exit 2
fi

# --- Ensure lore is available ---
install_lore() {
  local os arch url
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$arch" in
    x86_64)  arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *)
      echo "error: unsupported architecture: $arch" >&2
      exit 2
      ;;
  esac

  if [[ "$LORE_VERSION" == "latest" ]]; then
    url="https://github.com/GreyCoderK/lore/releases/latest/download/lore_${os}_${arch}.tar.gz"
  else
    url="https://github.com/GreyCoderK/lore/releases/download/${LORE_VERSION}/lore_${os}_${arch}.tar.gz"
  fi

  echo "Installing lore from $url ..." >&2
  local tmp
  tmp="$(mktemp -d)"
  trap "rm -rf '$tmp'" EXIT
  curl -sL "$url" -o "$tmp/lore.tar.gz"
  tar -xzf "$tmp/lore.tar.gz" -C "$tmp"
  LORE_BIN="$tmp/lore"
  chmod +x "$LORE_BIN"
  rm -f "$tmp/lore.tar.gz"
}

if command -v lore &>/dev/null; then
  LORE_BIN="lore"
elif [[ "$AUTO_INSTALL" == "true" ]]; then
  install_lore
else
  echo "error: 'lore' not found in PATH. Use --install to auto-install." >&2
  exit 2
fi

# --- Run Angela Draft ---
log() {
  if [[ "$QUIET" != "true" ]]; then
    echo "$@" >&2
  fi
}

log "angela-ci: analyzing $DOCS_PATH ..."

OUTPUT=$("$LORE_BIN" angela draft --all --path "$DOCS_PATH" 2>&1) || true

log "$OUTPUT"

# --- Count issues by severity ---
ERRORS=$(echo "$OUTPUT" | grep -c "^  error" || true)
WARNINGS=$(echo "$OUTPUT" | grep -c "^  warning" || true)

log ""
log "angela-ci: $ERRORS errors, $WARNINGS warnings"

# --- Determine exit code ---
EXIT_CODE=0

case "$FAIL_ON" in
  error)
    if [[ "$ERRORS" -gt 0 ]]; then
      EXIT_CODE=1
    fi
    ;;
  warning)
    if [[ "$ERRORS" -gt 0 ]] || [[ "$WARNINGS" -gt 0 ]]; then
      EXIT_CODE=1
    fi
    ;;
  none)
    EXIT_CODE=0
    ;;
esac

if [[ "$EXIT_CODE" -eq 0 ]]; then
  log "angela-ci: PASSED"
else
  log "angela-ci: FAILED (--fail-on=$FAIL_ON)"
fi

exit $EXIT_CODE
