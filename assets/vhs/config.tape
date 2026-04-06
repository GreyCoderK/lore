# lore config — VHS Tape
# Generate: vhs assets/vhs/config.tape
# Requires: scripts/setup-demo-project.sh

Output docs/assets/vhs/config.gif

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

# Show current config
Type "cat .lorerc"
Enter
Sleep 2.5s

# List stored API keys
Type "lore config list-keys"
Enter
Sleep 2s
