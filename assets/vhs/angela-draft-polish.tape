# lore angela draft + polish — VHS Tape
# Generate: vhs assets/vhs/angela-draft-polish.tape
# Requires: scripts/setup-demo-project.sh (Ollama configured + running)
#
# Shows: draft (local) → polish stops at first hunk prompt

Output docs/assets/vhs/angela-draft-polish.gif

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

# 1. Draft — free local analysis
Type "lore angela draft decision-session-based-auth-over-jwt-2026-04-07.md"
Enter
Sleep 5s

# 2. Polish — interactive hunk review
Type "lore angela polish decision-session-based-auth-over-jwt-2026-04-07.md"
Enter
Sleep 30s

# First hunk prompt visible — pause to show the interactive review
Sleep 5s
