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

Necessite Go 1.21+ :

```bash
go install github.com/greycoderk/lore@latest
```

## Binaires pre-compiles

```bash
curl -sSL https://github.com/GreyCoderK/lore/releases/latest/download/install.sh | sh
```

Ou telecharger depuis [GitHub Releases](https://github.com/GreyCoderK/lore/releases).

## Verifier l'installation

```bash
lore --version
# lore v1.0.0 (abc1234)
```

## Depannage

### macOS : "impossible d'ouvrir car le developpeur ne peut pas etre verifie"

```bash
xattr -d com.apple.quarantine $(which lore)
```

### Linux : `lore: command not found`

Assurez-vous que `$GOPATH/bin` (generalement `~/go/bin`) est dans votre `PATH` :

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Windows (WSL)

Lore fonctionne sous Windows Subsystem for Linux.

## Etapes suivantes

- [Quickstart](quickstart.md) — Demarrez en 5 minutes
- [Shell Completions](completions.md) — Completion pour votre shell
