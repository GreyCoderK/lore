# LORE-START
if ! command -v lore >/dev/null 2>&1; then
  REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
  if [ -x "$REPO_ROOT/lore" ]; then
    echo "lore: found ./lore but it is not in your PATH — hook skipped." >&2
    echo "" >&2
    echo "  Add it to your PATH:" >&2
    echo "    bash:       echo 'export PATH=\"\$PATH:$REPO_ROOT\"' >> ~/.bashrc && source ~/.bashrc" >&2
    echo "    zsh:        echo 'export PATH=\"\$PATH:$REPO_ROOT\"' >> ~/.zshrc && source ~/.zshrc" >&2
    echo "    fish:       fish_add_path $REPO_ROOT" >&2
    echo "    PowerShell: [Environment]::SetEnvironmentVariable('PATH', \$env:PATH + ';$REPO_ROOT', 'User')" >&2
  else
    echo "lore: command not found — hook skipped." >&2
    echo "" >&2
    echo "  Install lore using one of:" >&2
    echo "    go install github.com/greycoderk/lore@latest" >&2
    echo "    brew install greycoderk/lore/lore" >&2
    echo "" >&2
    echo "  Or add it to your PATH manually:" >&2
    echo "    bash:       echo 'export PATH=\"\$PATH:/path/to/lore\"' >> ~/.bashrc && source ~/.bashrc" >&2
    echo "    zsh:        echo 'export PATH=\"\$PATH:/path/to/lore\"' >> ~/.zshrc && source ~/.zshrc" >&2
    echo "    fish:       fish_add_path /path/to/lore" >&2
    echo "    PowerShell: [Environment]::SetEnvironmentVariable('PATH', \$env:PATH + ';C:\\path\\to\\lore', 'User')" >&2
  fi
  exit 0
fi
exec lore _hook-post-commit < /dev/tty
# LORE-END
