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

Nécessite Go 1.21+ :

```bash
go install github.com/greycoderk/lore@latest
```

## Binaires pré-compilés

```bash
curl -sSL https://github.com/GreyCoderK/lore/releases/latest/download/install.sh | sh
```

Ou télécharger depuis [GitHub Releases](https://github.com/GreyCoderK/lore/releases).

## Vérifier l'installation

```bash
lore --version
# lore v1.0.0 (abc1234)
```

## Dépannage

### macOS : "impossible d'ouvrir car le développeur ne peut pas être vérifié"

```bash
xattr -d com.apple.quarantine $(which lore)
```

### Linux : `lore: command not found`

Assurez-vous que `$GOPATH/bin` (généralement `~/go/bin`) est dans votre `PATH` :

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Windows (WSL)

Lore fonctionne sous Windows Subsystem for Linux.

## Étapes suivantes

- [Quickstart](quickstart.md) — Démarrez en 5 minutes
- [Shell Completions](completions.md) — Completion pour votre shell
