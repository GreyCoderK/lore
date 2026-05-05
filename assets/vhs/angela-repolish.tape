# lore angela re-polish — VHS Tape
# Generate: vhs assets/vhs/angela-repolish.tape
# Requires: scripts/setup-demo-project.sh + an AI provider configured
#           (Anthropic Claude Sonnet 4.6 in this recording).
#
# Shows: polish --yes (auto-accept the AI rewrite to get a clean baseline)
#        → cat the polished file
#        → manually append a new "## Alternatives" section
#        → polish --yes a second time → AI improves the new section too
#        → cat the final, idempotent result
#
# Key beat: the SECOND polish call produces a smaller diff than the first
# because the doc is already mostly clean. Demonstrates that polish is
# idempotent on already-polished content + handles incremental edits.

Output docs/assets/vhs/angela-repolish.gif

Set Shell "bash"
Set FontSize 14
Set Width 1100
Set Height 700
Set Padding 16
Set Theme "Dracula"
Set TypingSpeed 50ms
Set CursorBlink false

# Setup
Hide
Type "cd /tmp/lore-demo && clear"
Enter
Sleep 1s
Show

# 1. First polish (auto-accept to get a polished baseline)
Type "lore angela polish feature-token-bucket-rate-limiter-2026-05-05.md --yes"
Enter
Sleep 32s

# 2. Show the polished file
Type "cat .lore/docs/feature-token-bucket-rate-limiter-2026-05-05.md"
Enter
Sleep 4s

# 3. Add new content manually (simulate editing)
Type "echo '' >> .lore/docs/feature-token-bucket-rate-limiter-2026-05-05.md"
Enter
Sleep 200ms
Type "echo '## Alternatives' >> .lore/docs/feature-token-bucket-rate-limiter-2026-05-05.md"
Enter
Sleep 200ms
Type "echo 'Considered sliding window and leaky bucket.' >> .lore/docs/feature-token-bucket-rate-limiter-2026-05-05.md"
Enter
Sleep 500ms

# 4. Re-polish — AI improves the new section too (auto-accept)
Type "lore angela polish feature-token-bucket-rate-limiter-2026-05-05.md --yes"
Enter
Sleep 32s

# 5. Show final result
Type "cat .lore/docs/feature-token-bucket-rate-limiter-2026-05-05.md"
Enter
Sleep 4s
