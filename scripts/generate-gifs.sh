#!/bin/bash
# generate-gifs.sh
# Regenere tous les GIFs VHS pour le projet Lore.
# Usage: bash scripts/generate-gifs.sh [tape-name]
#   Sans argument: regenere tout
#   Avec argument:  regenere un seul (ex: bash scripts/generate-gifs.sh demo)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(dirname "$SCRIPT_DIR")"
cd "$ROOT"

# Verifier VHS
if ! command -v vhs &>/dev/null; then
    echo "ERREUR: VHS n'est pas installe."
    echo "  go install github.com/charmbracelet/vhs@latest"
    exit 1
fi

# Chrome path pour macOS (VHS utilise un navigateur headless)
if [ -z "${VHS_CHROME_PATH:-}" ]; then
    if [ -x "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" ]; then
        export VHS_CHROME_PATH="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
    fi
fi

# Reconstruire le projet temoin
echo ">>> Setup du projet temoin..."
bash scripts/setup-demo-project.sh
echo ""

# Si un argument est passe, ne generer que ce tape
if [ "${1:-}" != "" ]; then
    TAPE="$1"
    if [ -f "assets/${TAPE}.tape" ]; then
        echo ">>> assets/${TAPE}.tape..."
        vhs "assets/${TAPE}.tape"
    elif [ -f "assets/vhs/${TAPE}.tape" ]; then
        echo ">>> assets/vhs/${TAPE}.tape..."
        vhs "assets/vhs/${TAPE}.tape"
    else
        echo "ERREUR: Tape '${TAPE}' introuvable dans assets/ ou assets/vhs/"
        exit 1
    fi
    echo "Done."
    exit 0
fi

# Generer tous les tapes
TOTAL=0
FAILED=0

# Hero GIF
if [ -f assets/demo.tape ]; then
    echo ">>> assets/demo.tape..."
    if vhs assets/demo.tape; then
        ((TOTAL++))
    else
        echo "    ECHEC: demo.tape"
        ((FAILED++))
    fi
fi

# Tapes individuels
for tape in assets/vhs/*.tape; do
    [ -f "$tape" ] || continue
    echo ">>> $(basename "$tape")..."
    if vhs "$tape"; then
        ((TOTAL++))
    else
        echo "    ECHEC: $(basename "$tape")"
        ((FAILED++))
    fi
done

echo ""
echo "=== Generation terminee ==="
echo "  Reussis: ${TOTAL}"
echo "  Echoues: ${FAILED}"
echo ""

# Creer le dossier de sortie si necessaire
mkdir -p docs/assets/vhs

if [ -f docs/assets/demo.gif ]; then
    echo "  docs/assets/demo.gif ($(du -h docs/assets/demo.gif | cut -f1))"
fi
for gif in docs/assets/vhs/*.gif; do
    [ -f "$gif" ] || continue
    echo "  ${gif} ($(du -h "$gif" | cut -f1))"
done
