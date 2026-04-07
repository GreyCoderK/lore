# lore delete — VHS Tape
# Generate: vhs assets/vhs/delete.tape
# Requires: scripts/setup-demo-project.sh

Output docs/assets/vhs/delete.gif

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

# Delete a doc (with confirmation)
Type "lore delete feature-add-health-endpoint-2026-04-07.md --force"
Enter
Sleep 2s

# Verify it's gone
Type "lore list"
Enter
Sleep 2.5s
