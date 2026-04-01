# Manual GitHub Setup Guide

> **Internal document** — This is a setup guide for the maintainer, not a community file.

## 1. GitHub Discussions

### Activation

1. Go to **Settings > General > Features**
2. Check **Discussions**
3. Click **Save**

### Categories to create

| Category | Format | Description |
|---|---|---|
| **General** | Open-ended | General discussions about Lore |
| **Q&A** | Question / Answer | Questions with markable answers |
| **Ideas** | Open-ended | Feature suggestions and ideas |
| **Show & Tell** | Open-ended | Projects and repos using Lore |

## 2. GitHub Sponsors

### Activation

1. Go to [github.com/sponsors](https://github.com/sponsors) and sign up
2. Complete the profile with project description

### Sponsor Tiers

| Tier | Price | Benefit |
|---|---|---|
| **Merci / Thanks** | $1/mo | Thank you mention |
| **Supporter** | $5/mo | Name in README sponsors section |
| **Priority** | $25/mo | Priority support on issues |

### Goal

> "Next goal: full-time coverage on Lore"

### Verification

- `.github/FUNDING.yml` must contain: `github: greycoderk`
- Verify with: `cat .github/FUNDING.yml`

## 3. Social Preview

1. Go to **Settings > General**
2. Scroll to **Social preview**
3. Upload `assets/social-card.png` (1280x640px) — created in story 7e-1a

## 4. GitHub Pages (for story 7e-0b)

1. Go to **Settings > Pages**
2. Source: **Deploy from a branch**
3. Branch: `gh-pages` / `/ (root)`
4. Click **Save**
