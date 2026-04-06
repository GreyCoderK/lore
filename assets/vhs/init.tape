# lore init — VHS Tape
# Generate: vhs assets/vhs/init.tape
# Does NOT use demo project — creates its own temp repo

Output docs/assets/vhs/init.gif

Set Shell "bash"
Set FontSize 15
Set Width 1000
Set Height 500
Set Padding 16
Set Theme "Dracula"
Set TypingSpeed 60ms
Set CursorBlink false

# Setup invisible — fresh git repo
Hide
Type "rm -rf /tmp/lore-init-demo && mkdir /tmp/lore-init-demo && cd /tmp/lore-init-demo && git init -q && clear"
Enter
Sleep 1s
Show

# Run lore init
Type "lore init --no-demo"
Enter
Sleep 2.5s

# Show what was created
Type "ls .lore/"
Enter
Sleep 1.5s

Type "cat .git/hooks/post-commit | head -3"
Enter
Sleep 2s
