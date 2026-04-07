# FAQ

## General

### What is Lore?

A CLI tool that captures the *why* behind your code changes at commit-time. Three questions, ninety seconds, a Markdown document forever.

### Does Lore require an internet connection?

No. Everything works offline by default. AI features (Angela) are opt-in and require an API key.

### What languages does Lore support?

The CLI interface is bilingual: English and French. Set `language: "fr"` in `.lorerc` for French.

### Does Lore work with any Git hosting?

Yes. Lore operates locally via Git hooks. It works with GitHub, GitLab, Bitbucket, or any Git host.

## Usage

### Can I skip documentation for a commit?

Yes. Press Ctrl+C during the questions — partial answers are saved to pending. Or add `[doc-skip]` to your commit message.

### What happens during merge commits?

Lore skips merge commits automatically — no documentation needed.

### What happens in CI or non-TTY environments?

Commits are deferred to pending silently. In VS Code terminals, Lore sends a notification. Use `lore pending resolve` later.

### Why do I get a dialog instead of interactive questions?

Git redirects stdin to `/dev/null` for hooks. The Lore hook reconnects stdin from the terminal so the questions can be asked interactively.

**Fix:** Reinstall the hook:

```bash
lore hook uninstall
lore hook install
```

Verify the hook contains the stdin redirect:

=== "macOS / Linux"

    ```bash
    grep "dev/tty" .git/hooks/post-commit
    # Should show: exec lore _hook-post-commit < /dev/tty
    ```

    The hook uses `< /dev/tty` to reconnect stdin from the terminal. In environments where `/dev/tty` is unavailable (CI, Docker, pipes), commits are silently deferred to pending.

=== "Windows (Git Bash)"

    ```bash
    grep "dev/tty" .git/hooks/post-commit
    # Same mechanism — Git Bash (MSYS2) provides /dev/tty
    ```

    Windows uses Git Bash for hooks, which provides `/dev/tty` like Unix systems. If you use PowerShell or CMD directly, commits will be deferred to pending.

> **Note:** In environments where the terminal is unavailable (CI, Docker, pipes), commits are always deferred to pending — this is by design. See [Contextual Detection](guides/contextual-detection.md) for details.

### How does Lore handle notifications on different platforms?

When a commit is deferred (non-TTY), Lore can send a notification depending on `notification.mode` in `.lorerc`:

| Platform | `dialog` mode | `notify` mode |
|----------|--------------|---------------|
| **macOS** | AppleScript dialog (`osascript`) | `terminal-notifier` (if installed) or `display notification` |
| **Linux** | `zenity`, `kdialog`, or `yad` (whichever is available) | `notify-send` |
| **Windows** | PowerShell `System.Windows.Forms` balloon | PowerShell balloon notification |

API key storage also varies by platform:

| Platform | Keychain backend |
|----------|-----------------|
| **macOS** | System Keychain (`security` CLI) |
| **Linux** | `secret-tool` (GNOME Keyring / KWallet) |
| **Windows** | Windows Credential Manager (fallback to config) |

### Can I document old commits retroactively?

Yes: `lore new --commit abc1234`

### How do I undo a documented commit?

`lore delete <filename>` with confirmation.

## AI (Angela)

### Is AI required?

No. Lore works fully without AI. Angela is opt-in.

### What AI providers are supported?

Anthropic (Claude), OpenAI (GPT), and Ollama (local models).

### What does Angela Draft do without an API?

Local structural analysis: missing sections, style guide compliance, related documents, coherence checks. Zero network calls.

### What does Angela Polish do?

Sends your document to the AI provider for rewriting. Shows an interactive diff — accept or reject each change individually.

### I have a Claude.ai subscription but no API credits. Can I use Angela?

`angela draft` works **100% offline** — no API needed. For the polish/review features, you have two free options:

**Option 1: Ollama (local, free)**

```bash
brew install ollama
ollama pull llama3.2
```

```yaml
# .lorerc
ai:
  provider: "ollama"
  model: "llama3.2"
```

**Option 2: Manual polish via Claude.ai chat**

1. Run `lore angela draft <filename>` to get structural suggestions
2. Copy your document content into Claude.ai with this prompt:

```
Improve this technical decision document. Keep the Markdown format
with sections: What, Why, Alternatives, Impact. Be concise and technical:

[paste document content]
```

3. Paste the improved version back into your file

> **Note:** Claude.ai (chat subscription) and the Anthropic API are separate products with separate billing. The API requires credits purchased at [console.anthropic.com](https://console.anthropic.com).

## Data & Privacy

### Where is my data stored?

All data lives in `.lore/` inside your repository. Nothing is sent anywhere unless you explicitly use Angela Polish with an AI provider.

### Can I delete all Lore data?

`rm -rf .lore/` removes everything. Your Git history and code are untouched.

### What license is Lore under?

AGPL-3.0. A commercial license is available for proprietary use.

## Power User

### How do I tune the Decision Engine?

Edit `.lorerc`:

```yaml
decision:
  threshold_full: 50      # Lower = more full questions (default: 60)
  always_ask: [feat, breaking, security]
  always_skip: [docs, style, ci, build, chore]
```

Run `lore decision --explain HEAD` to see the scoring for any commit.

### Can I use Lore in a monorepo?

Yes. `lore init` at the repo root. Documents capture the full path of changed files. Use `lore show --type decision` + keyword search to find decisions per service.

### Can I use a custom AI model with Ollama?

```yaml
# .lorerc
ai:
  provider: "ollama"
  model: "llama3"          # Any model available in your Ollama instance
  endpoint: "http://localhost:11434"
```

No API key needed. The model runs entirely on your machine.

### How do I write a custom style guide for Angela?

Add a `style_guide` section in `.lorerc`:

```yaml
angela:
  style_guide:
    tone: "technical but approachable"
    max_length: 500
    required_sections: ["Why", "Alternatives Considered"]
    avoid: ["passive voice", "jargon without explanation"]
```

Angela Draft and Polish will check against these rules.

### Can I export my corpus?

Your corpus IS the export — it's Markdown files in `.lore/docs/`. Copy them anywhere. They're self-contained with YAML front matter. No proprietary format, no lock-in.

### How do I migrate from ADRs to Lore?

You don't — they're complementary. Keep your ADRs for big architectural decisions. Use Lore for the daily "why" behind every commit. Over time, your ADRs become the summaries and your Lore corpus becomes the detailed history.

### Can I use Lore in CI/CD?

```bash
# Fail build if pending docs exist
[ $(lore pending --quiet | wc -l) -eq 0 ] || exit 1

# Fail build if corpus is unhealthy
[ $(lore doctor --quiet) -eq 0 ] || exit 1

# Generate coverage badge
lore status --badge >> $GITHUB_STEP_SUMMARY
```

### How do I handle merge conflicts in `.lore/docs/`?

Rare — each commit creates a unique filename. If it happens, resolve like any Markdown conflict. Then `lore doctor --fix` to rebuild the index.

### What's the performance impact of the post-commit hook?

Negligible. The Decision Engine scores in ~0.4ms. The entire hook (including question rendering) adds < 100ms to a commit when auto-skipped. When you answer questions, the time is your typing speed.

### How do I disable Lore temporarily?

```bash
# Skip one commit
git commit -m "chore: deps [doc-skip]"

# Disable the hook entirely
lore hook uninstall

# Re-enable later
lore hook install
```

---

**Question not listed?** [Ask on GitHub Discussions Q&A](https://github.com/greycoderk/lore/discussions/categories/q-a)
