# lore hook — VHS Tape
# Generate: vhs assets/vhs/hook.tape
# Requires: scripts/setup-demo-project.sh

Output docs/assets/vhs/hook.gif

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

# Uninstall hook
Type "lore hook uninstall"
Enter
Sleep 1.5s

# Reinstall
Type "lore hook install"
Enter
Sleep 1.5s

# Verify
Type "cat .git/hooks/post-commit | head -3"
Enter
Sleep 2s
