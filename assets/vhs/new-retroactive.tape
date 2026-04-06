# lore new --commit — VHS Tape
# Generate: vhs assets/vhs/new-retroactive.tape
# Requires: scripts/setup-demo-project.sh

Output docs/assets/vhs/new-retroactive.gif

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

# Show recent commits
Type "git log --oneline -4"
Enter
Sleep 1.5s

# Document the undocumented commit retroactively
Type "lore new feature 'Add structured logging' 'Debugging production issues requires structured JSON logs'"
Enter
Sleep 3s
