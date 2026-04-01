# Installation

## Homebrew (macOS / Linux)

```bash
brew install GreyCoderK/tap/lore
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

### Windows

Lore fonctionne **nativement sous Windows** — pas besoin de WSL.

**Chocolatey (recommandé) :**
```powershell
choco install lore
```

**Go install :**
```powershell
go install github.com/greycoderk/lore@latest
```

**Binaire pré-compilé :** Téléchargez le `.zip` depuis [GitHub Releases](https://github.com/GreyCoderK/lore/releases), extrayez `lore.exe`, et ajoutez le dossier à votre `PATH`.

**Astuce PowerShell :** En cas d'erreur "execution policy" :
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

> WSL fonctionne aussi si vous préférez un environnement Unix sous Windows.

## Plateformes supportées

| Plateforme | Architectures | Formats de packages | Testé en CI |
|------------|--------------|---------------------|-------------|
| **macOS** | amd64 (Intel), arm64 (Apple Silicon) | Homebrew, tar.gz, Go, curl | Oui (`macos-latest`) |
| **Linux** | amd64, arm64 | Homebrew, Snap, deb, rpm, apk, tar.gz, Go, curl | Oui (`ubuntu-latest`) |
| **Windows** | amd64 | Chocolatey, zip, Go | Oui (`windows-latest`) |

### Canaux de distribution

| Canal | Commande | Plateformes |
|-------|----------|-------------|
| **Homebrew** | `brew install GreyCoderK/tap/lore` | macOS, Linux |
| **Snap** | `sudo snap install lore --classic` | Linux |
| **Chocolatey** | `choco install lore` | Windows |
| **Go** | `go install github.com/greycoderk/lore@latest` | Toutes (Go 1.21+) |
| **curl** | `curl -sSfL .../install.sh \| sh` | macOS, Linux |
| **deb** | `sudo dpkg -i lore_*.deb` | Debian, Ubuntu |
| **rpm** | `sudo rpm -i lore_*.rpm` | Fedora, RHEL, CentOS |
| **apk** | `apk add --allow-untrusted lore_*.apk` | Alpine Linux |
| **Binaire** | Télécharger depuis [GitHub Releases](https://github.com/GreyCoderK/lore/releases) | Toutes |


## Étapes suivantes

- [Quickstart](quickstart.md) — Démarrez en 5 minutes
- [Shell Completions](completions.md) — Complétion pour votre shell
