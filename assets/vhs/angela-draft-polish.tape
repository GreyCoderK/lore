# lore angela draft + polish — VHS Tape
# Generate: vhs assets/vhs/angela-draft-polish.tape
# Requires: scripts/setup-demo-project.sh + an AI provider configured
#           (Anthropic Claude Sonnet 4.6 in this recording).
#
# Shows the full polish flow as it ships in v1.2.x:
#   1. lore angela draft  → free, offline, structural analysis with persona signals
#   2. lore angela polish → preflight (tokens / personas / quality / cost),
#                           AI call with token + speed stats,
#                           interactive hunk review with [y/n/b/q],
#                           hallucination warnings,
#                           quality-score delta before → after.

Output docs/assets/vhs/angela-draft-polish.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 760
Set Padding 16
Set Theme "Dracula"
Set TypingSpeed 45ms
Set CursorBlink false
Set PlaybackSpeed 1.0

# ─── Setup (hidden) ─────────────────────────────────────────────────────────
Hide
Type "cd /tmp/lore-demo && clear"
Enter
Sleep 800ms
Show

# ─── 1. Draft — free, 100% offline structural analysis ──────────────────────
# Persona signals visible: which lenses fire, how many findings each.
Type "lore angela draft decision-session-based-auth-over-jwt-2026-05-05.md"
Enter
Sleep 6s

# Pause on the draft output so viewers can read the persona attribution
# (📖 Affoué, ✏️ Salou, 🔍 Kouamé…) and the structural findings.
Sleep 3s

# Quick clear so the polish output starts on a fresh screen.
Type "clear"
Enter
Sleep 500ms

# ─── 2. Polish — interactive AI rewrite with diff review ────────────────────
# Showcases:
#   [1/3] Reading…       → tokens / personas / quality / cost preflight
#   [2/3] Calling AI…    → spinner with elapsed/total, then tok/s + cost actual
#   [3/3] Computing diff → quality delta + hunk count
#                          interactive [y/n/b/q] prompt per hunk
#                          ⚠ hallucination warnings if the model added
#                            unsupported claims (lore guards against this).
Type "lore angela polish decision-session-based-auth-over-jwt-2026-05-05.md"
Enter

# Stage 1 (Reading…) renders almost instantly — wait for the AI call.
Sleep 4s

# Stage 2 (Calling AI). Sonnet 4.6 on this 2.5K-token doc = ~22-26s.
# Tape sleeps 30s for headroom + buffer to render the response stats line.
Sleep 30s

# Stage 3 (Computing diff) — quality delta line + first [y/n/b/q] prompt.
Sleep 4s

# Accept the polish (single-hunk on a small doc → one prompt).
# The [y/n/b/q] prompt is line-buffered, so we need Enter after the choice.
Type "y"
Enter
Sleep 4s

# ─── 3. Final state ─────────────────────────────────────────────────────────
# Show that the file on disk now reflects the polished version.
Type "head -40 .lore/docs/decision-session-based-auth-over-jwt-2026-05-05.md"
Enter
Sleep 6s
