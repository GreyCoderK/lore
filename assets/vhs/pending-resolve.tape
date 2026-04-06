# lore pending — VHS Tape
# Generate: vhs assets/vhs/pending-resolve.tape
# Requires: scripts/setup-demo-project.sh (creates 1 pending entry)

Output docs/assets/vhs/pending-resolve.gif

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

# List pending items
Type "lore pending"
Enter
Sleep 2s

# Resolve it — lore will ask the remaining questions
Type "lore pending resolve 1"
Enter
Sleep 1.5s

# Answer Why (type and what are pre-filled from pending record)
Type "Debugging production issues requires structured JSON logs"
Enter
Sleep 2s
