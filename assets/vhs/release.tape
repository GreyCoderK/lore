# lore release — VHS Tape
# Generate: vhs assets/vhs/release.tape
# Requires: scripts/setup-demo-project.sh (creates v0.1.0 tag)

Output docs/assets/vhs/release.gif

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

# Show tags
Type "git tag -l"
Enter
Sleep 1s

# Generate release notes
Type "lore release --version v0.1.0"
Enter
Sleep 3.5s
