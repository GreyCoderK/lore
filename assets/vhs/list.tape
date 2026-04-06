# lore list — VHS Tape
# Generate: vhs assets/vhs/list.tape
# Requires: scripts/setup-demo-project.sh

Output docs/assets/vhs/list.gif

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

# Full corpus
Type "lore list"
Enter
Sleep 2.5s

# Filter by type
Type "lore list --type decision"
Enter
Sleep 2.5s
