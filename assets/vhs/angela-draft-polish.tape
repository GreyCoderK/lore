# lore angela draft — VHS Tape
# Generate: vhs assets/vhs/angela-draft-polish.tape
# Requires: scripts/setup-demo-project.sh

Output docs/assets/vhs/angela-draft-polish.gif

Set Shell "bash"
Set FontSize 15
Set Width 1000
Set Height 550
Set Padding 16
Set Theme "Dracula"
Set TypingSpeed 60ms
Set CursorBlink false

Hide
Type "cd /tmp/lore-demo && clear"
Enter
Sleep 1s
Show

# List decisions to pick one
Type "lore list --type decision"
Enter
Sleep 2s

# Run draft analysis (zero API — local only)
Type "lore angela draft .lore/docs/decision-jwt-over-sessions-2026-04-06.md"
Enter
Sleep 3.5s
