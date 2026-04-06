# Lore Demo — Hero GIF
# Generate: vhs assets/demo.tape
# Requires: scripts/setup-demo-project.sh
#
# Shows the core workflow: commit → document → retrieve

Output docs/assets/demo.gif

Set Shell "bash"
Set FontSize 15
Set Width 1000
Set Height 550
Set Padding 16
Set Theme "Dracula"
Set TypingSpeed 60ms
Set CursorBlink false

# Setup invisible — projet temoin deja pret
Hide
Type "cd /tmp/lore-demo && clear"
Enter
Sleep 1s
Show

# 1. Un vrai commit
Type "echo '// graceful shutdown' >> main.go"
Enter
Sleep 300ms
Type "git add -A && git commit -m 'feat(server): add graceful shutdown'"
Enter
Sleep 1.5s

# 2. Documenter avec lore new (3 args positionnels)
Type "lore new feature 'Add graceful shutdown' 'Clean shutdown prevents dropped connections during deploy'"
Enter
Sleep 3s

# 3. Retrouver la decision
Type "lore show 'graceful'"
Enter
Sleep 3s
