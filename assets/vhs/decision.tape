# lore decision — VHS Tape
# Generate: vhs assets/vhs/decision.tape
# Requires: scripts/setup-demo-project.sh

Output docs/assets/vhs/decision.gif

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

# Explain scoring for HEAD commit
Type "lore decision --explain HEAD"
Enter
Sleep 3s

# Show calibration
Type "lore decision --calibration"
Enter
Sleep 3s
