# lore angela re-polish — VHS Tape
# Generate: vhs assets/vhs/angela-repolish.tape
# Requires: scripts/setup-demo-project.sh (Ollama configured + running)
#
# Shows: polish --yes → view result → manual edit → re-polish --yes → view final

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
Type "lore angela polish feature-token-bucket-rate-limiter-2026-04-07.md --yes"
Enter
Sleep 30s

# 2. Show the polished file
Type "cat .lore/docs/feature-token-bucket-rate-limiter-2026-04-07.md"
Enter
Sleep 4s

# 3. Add new content manually (simulate editing)
Type "echo '' >> .lore/docs/feature-token-bucket-rate-limiter-2026-04-07.md"
Enter
Sleep 200ms
Type "echo '## Alternatives' >> .lore/docs/feature-token-bucket-rate-limiter-2026-04-07.md"
Enter
Sleep 200ms
Type "echo 'Considered sliding window and leaky bucket.' >> .lore/docs/feature-token-bucket-rate-limiter-2026-04-07.md"
Enter
Sleep 500ms

# 4. Re-polish — AI improves the new section too (auto-accept)
Type "lore angela polish feature-token-bucket-rate-limiter-2026-04-07.md --yes"
Enter
Sleep 30s

# 5. Show final result
Type "cat .lore/docs/feature-token-bucket-rate-limiter-2026-04-07.md"
Enter
Sleep 4s
