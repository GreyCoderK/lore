# Installation

## Homebrew (macOS / Linux)

```bash
brew install GreyCoderK/lore/lore
```

## Snap (Linux)

```bash
sudo snap install lore --classic
```

## Go Install

Requires Go 1.21+:

```bash
go install github.com/greycoderk/lore@latest
```

## Pre-built Binaries

```bash
curl -sSL https://github.com/GreyCoderK/lore/releases/latest/download/install.sh | sh
```

Or download directly from [GitHub Releases](https://github.com/GreyCoderK/lore/releases).

## Verify Installation

```bash
lore --version
# lore v1.0.0 (abc1234)
```

## Troubleshooting

### macOS: "cannot be opened because the developer cannot be verified"

```bash
xattr -d com.apple.quarantine $(which lore)
```

### Linux: `lore: command not found`

Make sure `$GOPATH/bin` (usually `~/go/bin`) is in your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Add this to your `~/.bashrc` or `~/.zshrc` to make it permanent.

### Windows (WSL)

Lore works in Windows Subsystem for Linux. Install via Go or download the Linux binary.

## Next Steps

- [Quickstart](quickstart.md) — Get started in 5 minutes
- [Shell Completions](completions.md) — Tab completion for your shell
