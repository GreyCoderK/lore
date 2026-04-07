# Lore Demo — Hero GIF
# Generate: vhs assets/demo.tape
# Requires: scripts/setup-demo-project.sh
#
# Shows the real workflow: commit → hook triggers → 3 questions → doc saved

Output docs/assets/demo.gif

Set Shell "bash"
Set FontSize 15
Set Width 1000
Set Height 600
Set Padding 16
Set Theme "Dracula"
Set TypingSpeed 50ms
Set CursorBlink false

# Setup invisible
Hide
Type "cd /tmp/lore-demo && clear"
Enter
Sleep 1s
Show

# 1. Real commit with code change
Type "echo '// graceful shutdown handler' >> main.go"
Enter
Sleep 300ms
Type "git add -A && git commit -m 'feat(server): add graceful shutdown'"
Enter
Sleep 2.5s

# 2. Hook triggers — type is pre-filled from "feat:" prefix
# Press Enter to confirm the pre-filled type "feature"
Sleep 500ms
Enter
Sleep 800ms

# 3. What — pre-filled from commit subject, press Enter
Enter
Sleep 800ms

# 4. Why — the real question, type the answer
Type "Clean shutdown prevents dropped connections during deploy"
Enter
Sleep 1.5s

# 5. Optional: Alternatives (Enter to skip)
Enter
Sleep 500ms

# 6. Optional: Impact (Enter to skip)
Enter
Sleep 1.5s

# 7. Show the captured doc
Type "lore show 'graceful'"
Enter
Sleep 3s
