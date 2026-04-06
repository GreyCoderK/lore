# lore status — VHS Tape
# Generate: vhs assets/vhs/status-badge.tape
# Requires: scripts/setup-demo-project.sh

Output docs/assets/vhs/status-badge.gif

Set Shell "bash"
Set FontSize 15
Set Width 1000
Set Height 500
Set Padding 16
Set Theme "Dracula"
Set TypingSpeed 60ms
Set CursorBlink false

Hide
Type "cd /tmp/lore-demo && clear"
Enter
Sleep 1s
Show

# Dashboard
Type "lore status"
Enter
Sleep 3s

# Badge markdown
Type "lore status --badge"
Enter
Sleep 2.5s
