# Lore Interactive — Type Selector Demo
# Generate: vhs assets/vhs/interactive.tape
# Requires: scripts/setup-demo-project.sh
#
# Shows the interactive type selector with arrow keys

Output docs/assets/vhs/interactive.gif

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

# 1. Start lore new (fully interactive)
Type "lore new"
Enter
Sleep 1.5s

# 2. Type selector appears — navigate with arrow keys
# Down arrow to "decision"
Down
Sleep 400ms
Down
Sleep 400ms
# Press Enter to select "decision"
Enter
Sleep 800ms

# 3. What
Type "Switch from REST to gRPC for internal services"
Enter
Sleep 800ms

# 4. Why
Type "REST overhead adds 12ms per hop in the microservices chain"
Enter
Sleep 800ms

# 5. Alternatives (optional)
Type "GraphQL federation, keep REST with HTTP/2"
Enter
Sleep 500ms

# 6. Impact (optional)
Type "All internal clients need protobuf migration"
Enter
Sleep 2s
