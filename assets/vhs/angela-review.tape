# lore angela review — VHS Tape
# Generate: vhs assets/vhs/angela-review.tape
# Requires: scripts/setup-demo-project.sh (Ollama configured + running)
#
# Shows: list corpus → full AI review with real findings

Output docs/assets/vhs/angela-review.gif

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

# 1. Show corpus size
Type "lore list"
Enter
Sleep 3s

# 2. Full corpus review (local signals + AI via Ollama)
Type "lore angela review"
Enter
Sleep 25s
