---
type: reference
date: 2026-04-12
status: published
related:
  - demo.md
  - hook.md
  - ../getting-started/quickstart.md
  - ../guides/configuration.md
angela_mode: polish
---
# lore init

Initialize a Lore documentation repository in your project.

## Synopsis

```
lore init [flags]
```

## What Does This Do?

`lore init` prepares your project to start documenting the "why" behind your code changes. It creates the `.lore/` folder and installs the Git hook that triggers the question flow after each commit.

## Real World Scenario

> You just created a new Go project. You ran `git init`, wrote your first files, made your first commit. Now you want every future commit to carry its "why." One command:
>
> ```bash
> lore init
> ```
>
> From now on, every `git commit` triggers 3 questions. Your project has a memory.

![lore init](../assets/vhs/init.gif)
<!-- Generate: vhs assets/vhs/init.tape -->

## Prerequisites

- You must be inside a **Git repository** (a folder where you've run `git init`)
- That's it! No account, no API key, no internet connection needed

## What Happens When You Run It

```bash
cd my-project
lore init
```

Lore does 5 things:

1. **Creates the `.lore/` folder** — This is where all your documentation lives

```
.lore/
├── docs/          # Your documentation files go here
├── pending/       # Commits waiting to be documented
├── templates/     # Custom templates (optional, advanced)
├── store.db       # Smart index (auto-managed, ignore this)
└── README.md      # Explains Lore to anyone who clones your repo
```

2. **Creates `.lorerc`** — Shared settings file (committed to git, visible to your team)
3. **Creates `.lorerc.local`** — Personal settings file (gitignored, for API keys)
4. **Installs the Git hook** — A tiny script that triggers Lore after each commit
5. **Offers a demo** — Shows you how Lore works in ~45 seconds

> **Analogy:** The Git hook acts like a reminder that fires automatically after every commit — you never have to remember to document.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--no-demo` | bool | `false` | Skip the demo prompt after initialization |
| `--language` | string | `en` | Set interface language (`en` or `fr`) |
| `--quiet` | bool | `false` | No output except errors |
| `--verbose` | bool | `false` | Show detailed information |
| `--no-color` | bool | `false` | Disable colored output |

## Examples

### Basic Setup (Most Common)

```bash
cd my-project
lore init
# → ✓ Created .lore/ directory
# → ✓ Installed post-commit hook
# → ✓ Generated .lore/README.md
# → Would you like to see a demo? [Y/n]
```

### Silent Setup (CI/Scripts)

```bash
lore init --no-demo --quiet
# → No output, just sets everything up
```

### Already Initialized?

```bash
lore init
# → Already initialized (does nothing, no error)
# Safe to run multiple times!
```

## Common Questions

### "What is a Git hook?"

A Git hook is a script that runs automatically at certain points in your Git workflow. Lore uses a **post-commit hook** — it runs right after you type `git commit`. You don't need to understand hooks to use Lore; it handles everything for you.

### "Will this mess up my existing Git hooks?"

No. Lore uses special markers (`# LORE-START` / `# LORE-END`) in the hook file. If you already have hooks (like Husky or pre-commit), Lore adds its section without touching yours.

### "What if I'm not in a Git repository?"

```bash
lore init
# → Error: not a Git repository
# Fix: run "git init" first, then "lore init"
```

### "Can I undo this?"

Yes. Remove the `.lore/` folder with `rm -rf .lore` — your code and Git history are completely untouched.

## What Happens Next?

After `lore init`, the next time you run `git commit`, Lore will automatically ask you 3 questions:

1. **Type** — What kind of change? (feature, bugfix, decision, refactor, note)
2. **What** — Pre-filled from your commit message. Just press Enter.
3. **Why** — The important one! Why did you make this choice?

The "why" is captured in seconds and stored permanently alongside your code.

## Tips & Tricks

- **Safe to re-run:** `lore init` is idempotent — running it twice does nothing.
- **After cloning:** Team members should run `lore init` after cloning to install their local hook.
- **CI setup:** `lore init --no-demo --quiet` in pipelines to ensure `.lore/` exists.
- **Monorepo:** Run at the repo root. Documents capture full file paths.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success (or already initialized) |
| `1` | Not a Git repository |

## See Also

- [lore demo](demo.md) — Try Lore without initializing (safe preview)
- [lore hook](hook.md) — Manage the Git hook separately
- [Quickstart](../getting-started/quickstart.md) — Full 5-minute walkthrough
- [Configuration](../guides/configuration.md) — Customize Lore settings
