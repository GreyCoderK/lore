# lore doctor — VHS Tape
# Generate: vhs assets/vhs/doctor-fix.tape
# Requires: scripts/setup-demo-project.sh (creates orphan .tmp)

Output docs/assets/vhs/doctor-fix.gif

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

# Diagnose
Type "lore doctor"
Enter
Sleep 2.5s

# Fix
Type "lore doctor --fix"
Enter
Sleep 2.5s

# Verify clean
Type "lore doctor"
Enter
Sleep 2s
