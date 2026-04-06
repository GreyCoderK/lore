#!/bin/bash
# setup-demo-project.sh
# Cree un projet temoin Git + Lore pour les enregistrements VHS.
# Usage: bash scripts/setup-demo-project.sh
set -euo pipefail

DIR="/tmp/lore-demo"
echo ">>> Nettoyage de $DIR..."
rm -rf "$DIR"
mkdir -p "$DIR"
cd "$DIR"

# --- 1. Init git ---
git init -q
git config user.name "Demo Dev"
git config user.email "dev@example.com"

# --- 2. Fichiers Go minimaux ---
cat > main.go << 'GO'
package main

import "fmt"

func main() {
    fmt.Println("API Server starting on :8080")
}
GO

cat > handler.go << 'GO'
package main

import "net/http"

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("ok"))
}
GO

cat > README.md << 'MD'
# My API
A simple REST API with rate limiting, auth, and caching.
MD

git add -A && git commit -q -m "Initial project setup"

# --- 3. Init lore (sans demo interactive) ---
lore init --no-demo

# Desactiver le hook temporairement pour que les commits suivants
# ne declenchent pas le flow interactif pendant le setup
if [ -f .git/hooks/post-commit ]; then
    mv .git/hooks/post-commit .git/hooks/post-commit.bak
fi

# --- 4. Construire le corpus (5 docs avec contenu riche) ---
# On utilise les args positionnels (type, what, why) SANS --commit
# pour garantir un contenu propre dans les docs.

# Commit 2
echo '// rate limiting middleware' >> handler.go
git add -A && git commit -q -m "feat(api): add rate limiting middleware"

# Commit 3
echo '// JWT auth' >> handler.go
git add -A && git commit -q -m "feat(auth): add JWT authentication"

# Commit 4
echo '// Redis cache layer' >> handler.go
git add -A && git commit -q -m "feat(cache): add Redis caching layer"

# Commit 5
echo '// Prometheus metrics' >> handler.go
git add -A && git commit -q -m "feat(metrics): add Prometheus endpoint"

# Doc 1 — feature
lore new feature "Add health endpoint" "Load balancer needs a health check path"

# Doc 2 — feature
lore new feature "Add rate limiting" "API hammered by single client at 10K req/min"

# Doc 3 — decision
lore new decision "JWT over sessions" "Stateless auth scales horizontally without Redis"

# Doc 4 — decision
lore new decision "Redis over Memcached" "Need pub/sub for cache invalidation"

# Doc 5 — feature
lore new feature "Prometheus metrics endpoint" "SRE team needs /metrics for Grafana dashboards"

# --- 5. Cas speciaux pour les demos ---

# Fichier orphan .tmp pour demo doctor (doctor cherche *.tmp)
echo "temporary data" > .lore/docs/interrupted-write.tmp

# Commit non-documente + pending entry manuelle
echo '// structured logging' >> handler.go
git add -A && git commit -q -m "feat(log): add structured logging"
LAST_HASH=$(git rev-parse --short HEAD)
LAST_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

mkdir -p .lore/pending
cat > ".lore/pending/${LAST_HASH}.yaml" << YAML
commit: "${LAST_HASH}"
date: "${LAST_DATE}"
message: "feat(log): add structured logging"
status: "deferred"
reason: "non-tty"
answers:
  type: "feature"
  what: "add structured logging"
  why: ""
  alternatives: ""
  impact: ""
YAML

# Tag pour demo release (pointe 3 commits avant HEAD)
git tag v0.1.0 HEAD~3

# --- 6. Restaurer le hook pour le hero GIF ---
if [ -f .git/hooks/post-commit.bak ]; then
    mv .git/hooks/post-commit.bak .git/hooks/post-commit
fi

# --- Resume ---
echo ""
echo "=== Projet temoin pret ==="
echo "  Path:    $DIR"
echo "  Docs:    $(ls .lore/docs/*.md 2>/dev/null | wc -l | tr -d ' ')"
echo "  Commits: $(git rev-list --count HEAD)"
echo "  Tags:    $(git tag -l | tr '\n' ' ')"
PENDING_COUNT=$(ls .lore/pending/*.yaml 2>/dev/null | wc -l | tr -d ' ')
echo "  Pending: ${PENDING_COUNT} element(s)"
echo "  Doctor:  interrupted-write.tmp orphan present"
