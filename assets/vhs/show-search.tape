# lore show — VHS Tape
# Generate: vhs assets/vhs/show-search.tape
# Requires: scripts/setup-demo-project.sh

Output docs/assets/vhs/show-search.gif

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

# Search for a decision
Type "lore show 'JWT'"
Enter
Sleep 3s

# Search for a feature
Type "lore show 'rate'"
Enter
Sleep 3s
