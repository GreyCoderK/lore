---
type: feature
date: "2026-04-09"
commit: c23555b486b6ecc7bdf2467995780b59c0bd30a0
branch: main
status: draft
generated_by: hook
---
# Angela Enhancement Sprint — quality scoring, audience rewrite, auto-mode, preflight, multi-pass, post-processing, multi-provider CI, i18n complet, documentation bilingue

## What
Angela Enhancement Sprint — quality scoring, audience rewrite, auto-mode, preflight, multi-pass, post-processing, multi-provider CI, i18n complet, documentation bilingue

## Why
Angela etait un prototype minimal : l'utilisateur attendait 2 minutes sans feedback, recevait 46 hunks sans contexte, perdait du contenu sans avertissement, et ne pouvait pas estimer le cout avant un appel API. Ce sprint transforme Angela en assistant complet avec verification pre-appel (abort si trop gros, estimation cout), feedback temps reel (spinner, token stats, vitesse), triage intelligent (auto-mode classifie les hunks, warnings avant suppression), reecriture ciblee par audience (--for CTO, equipe commerciale), score qualite avant/apres, et support multi-provider en CI (Anthropic, OpenAI, Ollama, Groq, Together, Mistral). La documentation bilingue EN/FR et le CHANGELOG ont ete mis a jour pour couvrir toutes les nouvelles features, et Angela a ete integree dans notre propre CI comme quality gate sur le site de documentation.
