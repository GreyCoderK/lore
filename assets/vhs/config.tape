# lore config — VHS Tape
# Generate: vhs assets/vhs/config.tape
# Requires: scripts/setup-demo-project.sh (with Ollama configured)

Output docs/assets/vhs/config.gif

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

# Show current config (includes ai.provider: "ollama")
Type "cat .lorerc"
Enter
Sleep 3.5s

# Show lore doctor --config for resolved values
Type "lore doctor --config"
Enter
Sleep 2.5s
