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

### Windows

Lore runs **natively on Windows** — no WSL required.

**Chocolatey (recommended):**
```powershell
choco install lore
```

**Go install:**
```powershell
go install github.com/greycoderk/lore@latest
```

**Pre-built binary:** Download the `.zip` from [GitHub Releases](https://github.com/GreyCoderK/lore/releases), extract `lore.exe`, and add the folder to your `PATH`.

**PowerShell tip:** If you get an "execution policy" error, run:
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

> WSL also works if you prefer a Unix environment on Windows.

## Supported Platforms

| Platform | Architectures | Package formats | Tested in CI |
|----------|--------------|-----------------|-------------|
| **macOS** | amd64 (Intel), arm64 (Apple Silicon) | Homebrew, tar.gz, Go, curl | Yes (`macos-latest`) |
| **Linux** | amd64, arm64 | Homebrew, Snap, deb, rpm, apk, tar.gz, Go, curl | Yes (`ubuntu-latest`) |
| **Windows** | amd64 | Chocolatey, zip, Go | Yes (`windows-latest`) |

### Distribution Channels

| Channel | Command | Platforms |
|---------|---------|-----------|
| **Homebrew** | `brew install GreyCoderK/lore/lore` | macOS, Linux |
| **Snap** | `sudo snap install lore --classic` | Linux |
| **Chocolatey** | `choco install lore` | Windows |
| **Go** | `go install github.com/greycoderk/lore@latest` | All (requires Go 1.21+) |
| **curl** | `curl -sSfL .../install.sh \| sh` | macOS, Linux |
| **deb** | `sudo dpkg -i lore_*.deb` | Debian, Ubuntu |
| **rpm** | `sudo rpm -i lore_*.rpm` | Fedora, RHEL, CentOS |
| **apk** | `apk add --allow-untrusted lore_*.apk` | Alpine Linux |
| **Binary** | Download from [GitHub Releases](https://github.com/GreyCoderK/lore/releases) | All |

## Next Steps

- [Quickstart](quickstart.md) — Get started in 5 minutes
- [Shell Completions](completions.md) — Tab completion for your shell
