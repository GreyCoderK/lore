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

# --- 5. Vrais fichiers Go pour demos angela (2 commits ~100 lignes) ---

cat > auth.go << 'GO'
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Session represents an authenticated user session.
type Session struct {
	UserID    string
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
	IP        string
}

// SessionStore manages active sessions with thread-safe access.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	ttl      time.Duration
}

// NewSessionStore creates a store with the given session TTL.
func NewSessionStore(ttl time.Duration) *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
		ttl:      ttl,
	}
}

// Create generates a new session for the given user.
func (s *SessionStore) Create(userID, ip string) (*Session, error) {
	token, err := generateToken(32)
	if err != nil {
		return nil, fmt.Errorf("session: generate token: %w", err)
	}
	now := time.Now()
	session := &Session{
		UserID:    userID,
		Token:     token,
		CreatedAt: now,
		ExpiresAt: now.Add(s.ttl),
		IP:        ip,
	}
	s.mu.Lock()
	s.sessions[token] = session
	s.mu.Unlock()
	return session, nil
}

// Validate checks if a token is valid and not expired.
func (s *SessionStore) Validate(token string) (*Session, error) {
	s.mu.RLock()
	session, ok := s.sessions[token]
	s.mu.RUnlock()
	if !ok {
		return nil, errors.New("session: not found")
	}
	if time.Now().After(session.ExpiresAt) {
		s.Revoke(token)
		return nil, errors.New("session: expired")
	}
	return session, nil
}

// Revoke removes a session by token.
func (s *SessionStore) Revoke(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// Cleanup removes all expired sessions. Call periodically.
func (s *SessionStore) Cleanup() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	now := time.Now()
	for token, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, token)
			removed++
		}
	}
	return removed
}

// AuthMiddleware extracts the Bearer token and validates the session.
func AuthMiddleware(store *SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(header, "Bearer ")
			session, err := store.Validate(token)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), "session", session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func generateToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
GO

git add -A && git commit -q -m "feat(auth): add session-based authentication with middleware"
lore new decision "Session-based auth over JWT" "Need server-side session revocation for compliance — JWT tokens cannot be invalidated before expiry which violates SOC2 requirements"

cat > ratelimit.go << 'GO'
package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a token bucket algorithm per client IP.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    int           // tokens per interval
	burst   int           // max tokens
	interval time.Duration
	cleanup  time.Duration // how often to purge stale buckets
}

type bucket struct {
	tokens    int
	lastFill  time.Time
}

// NewRateLimiter creates a limiter allowing `rate` requests per `interval`
// with a burst capacity of `burst`.
func NewRateLimiter(rate, burst int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		burst:    burst,
		interval: interval,
		cleanup:  5 * time.Minute,
	}
	go rl.purgeLoop()
	return rl
}

// Allow checks if the request from the given IP should be allowed.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[ip]
	if !ok {
		b = &bucket{tokens: rl.burst, lastFill: time.Now()}
		rl.buckets[ip] = b
	}

	// Refill tokens based on elapsed time
	elapsed := time.Since(b.lastFill)
	refill := int(elapsed / rl.interval) * rl.rate
	if refill > 0 {
		b.tokens += refill
		if b.tokens > rl.burst {
			b.tokens = rl.burst
		}
		b.lastFill = time.Now()
	}

	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

// RemainingTokens returns the current token count for an IP.
func (rl *RateLimiter) RemainingTokens(ip string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.buckets[ip]
	if !ok {
		return rl.burst
	}
	return b.tokens
}

// RateLimitMiddleware rejects requests that exceed the rate limit.
func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if !rl.Allow(ip) {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rl.interval.Seconds())))
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			w.Header().Set("X-RateLimit-Remaining",
				fmt.Sprintf("%d", rl.RemainingTokens(ip)))
			next.ServeHTTP(w, r)
		})
	}
}

// purgeLoop periodically removes stale buckets to prevent memory leaks.
func (rl *RateLimiter) purgeLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.cleanup)
		for ip, b := range rl.buckets {
			if b.lastFill.Before(cutoff) {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}
GO

git add -A && git commit -q -m "feat(api): add token bucket rate limiter with per-IP tracking"
lore new feature "Token bucket rate limiter" "Previous naive rate limiter counted requests globally — one heavy client starved everyone. Token bucket with per-IP tracking isolates clients fairly"

# --- 6. Configurer Ollama pour demos angela ---
# Replace the empty ai: section created by lore init
sed -i '' 's/provider: ""/provider: "ollama"/' .lorerc
sed -i '' 's/model: ""/model: "llama3.2"/' .lorerc

# --- 7. Cas speciaux pour les demos ---

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

# --- 8. Restaurer le hook pour le hero GIF ---
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
