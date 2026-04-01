# Lore Demo — VHS Tape File
# Generate GIF: vhs assets/demo.tape
# Requires: VHS (go install github.com/charmbracelet/vhs@latest)

Output assets/demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 600
Set Padding 20
Set Theme "Dracula"

# Show a commit
Type "git commit -m 'Add rate limiting to API endpoints'"
Sleep 500ms
Enter
Sleep 1s

# Lore hook triggers
Type "# Lore post-commit hook triggers..."
Sleep 1s
Enter
Sleep 500ms

# Question 1: Type
Type "? Document type: feature"
Enter
Sleep 800ms

# Question 2: What
Type "? What changed: Added rate limiting middleware to all public API endpoints"
Enter
Sleep 800ms

# Question 3: Why
Type "? Why was this done: Production API was getting hammered by a single client doing 10K req/min, causing latency spikes for everyone"
Enter
Sleep 1s

# Doc generated
Type "# ✓ Document saved: .lore/docs/2026-03-15-add-rate-limiting.md"
Enter
Sleep 1.5s

# Show the doc
Type "lore show"
Enter
Sleep 500ms
Type "---"
Enter
Type "type: feature"
Enter
Type "date: 2026-03-15T14:30:00Z"
Enter
Type "commit: e4f5a6b"
Enter
Type "---"
Enter
Type "# Add rate limiting to API endpoints"
Enter
Type ""
Enter
Type "## Why"
Enter
Type "Production API was getting hammered by a single client"
Enter
Type "doing 10K req/min, causing latency spikes for everyone."
Enter
Sleep 2s

# End
Type "# 90 seconds. The 'why' is captured forever."
Enter
Sleep 2s
