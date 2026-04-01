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

## Data & Privacy

### Where is my data stored?

All data lives in `.lore/` inside your repository. Nothing is sent anywhere unless you explicitly use Angela Polish with an AI provider.

### Can I delete all Lore data?

`rm -rf .lore/` removes everything. Your Git history and code are untouched.

### What license is Lore under?

AGPL-3.0. A commercial license is available for proprietary use.
