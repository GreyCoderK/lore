---
type: guide
date: 2026-04-12
status: published
related:
  - guides/contextual-detection.md
  - guides/configuration.md
  - commands/angela-polish.md
---
# FAQ

## General

### What is lore?

A CLI tool that captures the *why* behind your code changes at commit-time. Three questions, ninety seconds, a Markdown document that lasts forever.

### Does lore require an internet connection?

No. Everything works offline by default. AI features (Angela) are opt-in and require an API key.

### What languages does lore support?

The CLI interface is bilingual: English and French. Set `language: "fr"` in `.lorerc` to switch to French.

### Does lore work with any Git hosting?

Yes. lore operates locally via Git hooks and works with GitHub, GitLab, Bitbucket, or any Git host.

## Usage

### Can I skip documentation for a commit?

Yes. Press Ctrl+C during the questions — partial answers are saved to pending. Alternatively, add `[doc-skip]` to your commit message for a silent skip.

### What happens during merge commits?

lore skips merge commits automatically — no documentation needed.

### What happens in CI or non-TTY environments?

Commits are deferred to pending silently. In VS Code terminals, lore sends a notification. Use `lore pending resolve` later.

### Why do I get a dialog instead of interactive questions?

Git redirects stdin to `/dev/null` for hooks. The lore hook reconnects stdin from the terminal so questions can be asked interactively.

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

### How does lore handle notifications on different platforms?

When a commit is deferred (non-TTY), lore sends a notification based on `notification.mode` in `.lorerc`:

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

No. lore works fully without AI. Angela is opt-in.

### What AI providers are supported?

| Provider | Setup | Cost | Best for |
|----------|-------|------|----------|
| **Anthropic** (Claude) | [console.anthropic.com](https://console.anthropic.com) → API Key + $5 credits | ~$0.01–0.05/call | Best quality technical docs |
| **OpenAI** (GPT) | [platform.openai.com](https://platform.openai.com) → API Key + $5 credits | ~$0.001–0.01/call | Good quality, cheapest cloud option |
| **Ollama** (local) | [ollama.com/download](https://ollama.com/download) + `ollama pull llama3.2` | Free | Privacy, offline, no account needed |

See the [AI Provider Setup guide](guides/configuration.md#ai-provider-setup) for step-by-step instructions with download links.

### What does Angela draft do without an API?

Local structural analysis: missing sections, style guide compliance, related documents, and coherence checks. Zero network calls.

### What does Angela polish do?

Sends your document to the AI provider for rewriting and shows an interactive diff — accept or reject each change individually. The output is expected to include Mermaid diagrams, tables, and concrete details.

> **Quality depends on your model.** Small local models (llama3.2) may produce generic text. Larger models (Claude Sonnet, GPT-4o, llama3.1:70b) produce much better results with diagrams and tables. See the [quality warning](commands/angela-polish.md#ai-quality-warning) in the polish docs.

### I have a Claude.ai subscription but no API credits. Can I use Angela?

`angela draft` works **100% offline** — no API needed. For polish/review features, two free options are available:

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

All data lives in `.lore/` inside your repository. Nothing is sent anywhere unless you explicitly use Angela polish with an AI provider.

### Can I delete all lore data?

`rm -rf .lore/` removes everything. Your Git history and code remain untouched.

### What license is lore under?

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

### Can I use lore in a monorepo?

Yes. Run `lore init` at the repo root. Documents capture the full path of changed files. Use `lore show --type decision` with keyword search to find decisions per service.

### Can I use a custom AI model with Ollama?

```yaml
# .lorerc
ai:
  provider: "ollama"
  model: "llama3"          # Any model available in your Ollama instance
  endpoint: "http://localhost:11434"
```

No API key needed — the model runs entirely on your machine.

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

Your corpus is the export — Markdown files in `.lore/docs/`. Copy them anywhere. They are self-contained with YAML front matter. No proprietary format, no lock-in.

### How do I migrate from ADRs to lore?

You don't — they are complementary. Keep your ADRs for big architectural decisions. Use lore for the daily "why" behind every commit. Over time, your ADRs become the summaries and your lore corpus becomes the detailed history.

### Can I use lore in CI/CD?

```bash
# Fail build if pending docs exist
[ $(lore pending --quiet | wc -l) -eq 0 ] || exit 1

# Fail build if corpus is unhealthy
[ $(lore doctor --quiet) -eq 0 ] || exit 1

# Generate coverage badge
lore status --badge >> $GITHUB_STEP_SUMMARY
```

### How do I handle merge conflicts in `.lore/docs/`?

This is rare — each commit generates a unique filename. If it happens, resolve it like any Markdown conflict, then run `lore doctor --fix` to rebuild the index.

### What is the performance impact of the post-commit hook?

Negligible. The Decision Engine scores in ~0.4ms. The entire hook (including question rendering) adds less than 100ms when auto-skipped. When you answer questions, the time is bounded by your typing speed.

### How do I disable lore temporarily?

```bash
# Skip one commit
git commit -m "chore: deps [doc-skip]"

# Disable the hook entirely
lore hook uninstall

# Re-enable later
lore hook install
```

### Why don't I see the lore logo in notifications on macOS?

macOS `display notification` (osascript) does not support custom icons. Install `terminal-notifier` to enable the lore logo:

```bash
brew install terminal-notifier
```

lore auto-installs it if Homebrew is available. After installation, the logo appears in all toast notifications. Dialogs (the interactive question flow) already show the logo without `terminal-notifier`.

### The amend workflow isn't asking me any questions

Two common causes:

1. **You are running an old version of lore.** Check `lore --version`. The git hook uses `lore` from your `PATH` — make sure it resolves to the latest version. Update via `brew upgrade lore` or `go install github.com/greycoderk/lore@latest`.

2. **The terminal is not interactive.** Some IDEs and CI environments do not provide a TTY. lore then defers the commit to `.lore/pending/` automatically. Check with `lore pending` and resolve with `lore pending resolve`.

---

**Question not listed?** [Ask on GitHub Discussions Q&A](https://github.com/greycoderk/lore/discussions/categories/q-a)
