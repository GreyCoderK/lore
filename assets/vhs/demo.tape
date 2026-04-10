# Lore — Landing Page Demo
# Generate: vhs assets/vhs/demo.tape
# Shows the 30-second pitch: init → commit → capture the why
#
# Output lands at docs/assets/demo.gif (referenced from docs/index.md
# and docs/index.fr.md).

Output docs/assets/demo.gif

Set Shell "bash"
Set FontSize 15
Set Width 1000
Set Height 600
Set Padding 16
Set Theme "Dracula"
Set TypingSpeed 55ms
Set CursorBlink false

# Setup invisible — fresh git repo with a tiny change staged
Hide
Type "rm -rf /tmp/lore-demo-landing && mkdir /tmp/lore-demo-landing && cd /tmp/lore-demo-landing && git init -q && git config user.email demo@lore.dev && git config user.name Demo && clear"
Enter
Sleep 500ms
Show

# Step 1 — Initialize Lore
Type "lore init --no-demo"
Enter
Sleep 2s

# Step 2 — Stage a change
Type "echo 'package main' > main.go && git add main.go"
Enter
Sleep 1s

# Step 3 — Commit triggers the interactive flow
Type "git commit -m 'feat: initial scaffold'"
Enter
Sleep 1500ms

# Answer Type (default: feature)
Enter
Sleep 600ms

# Answer What (default from subject)
Enter
Sleep 600ms

# Answer Why — the heart of Lore
Type "Because the project needs a starting point."
Enter
Sleep 1500ms

# Step 4 — Show the captured document
Type "ls .lore/docs/"
Enter
Sleep 2500ms

# Cleanup invisible
Hide
Type "cd / && rm -rf /tmp/lore-demo-landing"
Enter
