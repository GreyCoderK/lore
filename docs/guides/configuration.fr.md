# Configuration

Lore utilise un système de configuration en cascade.

## Fichiers de configuration

| Fichier | Rôle | Git |
|---------|------|-----|
| `.lorerc` | Configuration partagée du projet | Committé |
| `.lorerc.local` | Surcharges personnelles (clés API) | Gitignore (chmod 600) |
| `LORE_*` env vars | Surcharges CI/automation | — |
| `--language` flag | Surcharge CLI | — |

**Ordre de résolution** (priorité décroissante) : flags CLI > env vars > `.lorerc.local` > `.lorerc` > défauts.

## Référence complète

```yaml
# .lorerc — config partagée
language: "fr"              # "en" ou "fr" — langue de l'interface

ai:
  provider: ""              # "anthropic", "openai", "ollama", ou "" (zéro-API)
  model: ""                 # Nom du modèle

angela:
  mode: draft               # Mode par défaut : "draft" (zéro-API) ou "polish" (1 appel API)
  max_tokens: 2000

hooks:
  post_commit: true          # Activer le hook post-commit
  star_prompt: true          # Afficher le prompt star
  star_prompt_after: 5       # Afficher le prompt star après N commits documentés (0 = désactivé)
  amend_prompt: true         # Demander "Documenter ce changement ?" lors de git commit --amend

notification:
  mode: auto                 # auto, terminal, dialog, notify, silent
  disabled_envs: []          # Environnements à ignorer (ex. ["vim"])
  amend: true                # Activer les notifications pour les amend commits

decision:
  threshold_full: 60         # Score >= 60 : questionnaire complet
  threshold_reduced: 35      # Score 35-59 : questions réduites
  threshold_suggest: 15      # Score 15-34 : suggestion de skip (confirmation)
  always_ask: [feat, breaking]  # Toujours demander pour ces types de commit
  always_skip: [docs, style, ci, build]  # Auto-skip pour ces types
  learning: true             # Activer l'apprentissage LKS
  learning_min_commits: 20   # Minimum de commits avant activation du learning

templates:
  dir: .lore/templates

output:
  format: markdown
  dir: .lore/docs
```

## Surcharges personnelles

```yaml
# .lorerc.local — personnel, gitignore, chmod 600
ai:
  provider: "anthropic"
  model: "claude-sonnet-4-20250514"
  api_key: "sk-ant-..."     # Stockée ici ou dans le trousseau OS
```

## Variables d'environnement

| Variable | Équivalent |
|----------|------------|
| `LORE_LANGUAGE` | `language` |
| `LORE_AI_PROVIDER` | `ai.provider` |
| `LORE_AI_API_KEY` | `ai.api_key` |

## Valider la configuration

```bash
lore doctor --config
```

Vérifie les fautes de frappe, clés inconnues, et suggère des corrections via distance de Levenshtein.

## Configurations typiques

### Développeur solo (minimaliste)

```yaml
# .lorerc — juste l'essentiel
hooks:
  post_commit: true
output:
  dir: .lore/docs
```

Pas d'IA, pas de config de langue. Anglais par défaut, mode zéro-API. Simplicité maximale.

### Projet open source

```yaml
# .lorerc — committé dans le repo
language: "en"
hooks:
  post_commit: true
  star_prompt_after: 5
decision:
  always_ask: [feat, breaking]
  always_skip: [docs, style, ci]
output:
  dir: .lore/docs
```

Le prompt star encourage les contributeurs à mettre une étoile. Le Decision Engine ignore les commits triviaux automatiquement.

### Équipe avec IA

```yaml
# .lorerc — config partagée (committée)
language: "en"
ai:
  provider: "anthropic"
  model: "claude-sonnet-4-20250514"
hooks:
  post_commit: true
angela:
  mode: draft
  max_tokens: 2000
```

```yaml
# .lorerc.local — personnel (gitignore, chmod 600)
ai:
  api_key: "sk-ant-..."
```

Chaque membre de l'équipe stocke sa propre clé API. La config partagée définit le fournisseur et le modèle.

### Projet bilingue (FR/EN)

```yaml
# .lorerc
language: "fr"
hooks:
  post_commit: true
```

Tous les messages UI, prompts, badges et messages de renforcement passent en français. "Lore" devient "L'or."

## Configuration du fournisseur IA

Les commandes `polish` et `review` d'Angela nécessitent un fournisseur IA. Trois fournisseurs sont supportés, chacun avec des compromis différents.

### Anthropic (Claude)

Meilleure qualité pour la documentation technique. Nécessite des crédits API achetés séparément d'un abonnement chat Claude.ai.

```yaml
# .lorerc
ai:
  provider: "anthropic"
  model: "claude-sonnet-4-20250514"  # ou claude-haiku-4-5-20251001 (moins cher)
```

```bash
# Stocker la clé API dans le trousseau OS
lore config set-key anthropic
```

> **Important :** Un abonnement chat Claude.ai (Pro, Team) n'inclut PAS de crédits API. L'API est facturée séparément sur [console.anthropic.com](https://console.anthropic.com) → Plans & Billing. Minimum $5 de crédits.

| Élément | Détail |
|---------|--------|
| **Obtenir une clé** | [console.anthropic.com](https://console.anthropic.com) → API Keys |
| **Coût par appel** | ~$0.01–0.05 (Sonnet), ~$0.001 (Haiku) |
| **Endpoint** | `https://api.anthropic.com/v1/messages` (défaut, automatique) |
| **Format** | Spécifique Anthropic (non compatible OpenAI) |

### OpenAI (GPT)

```yaml
# .lorerc
ai:
  provider: "openai"
  model: "gpt-4o-mini"  # le moins cher, ou gpt-4o pour la meilleure qualité
```

```bash
lore config set-key openai
```

| Élément | Détail |
|---------|--------|
| **Obtenir une clé** | [platform.openai.com](https://platform.openai.com) → API Keys |
| **Coût par appel** | ~$0.001 (gpt-4o-mini), ~$0.01 (gpt-4o) |
| **Endpoint** | `https://api.openai.com/v1/chat/completions` (défaut) |
| **Endpoint custom** | Définir `ai.endpoint` pour APIs compatibles (ex : Ollama en mode OpenAI, Azure OpenAI — non testé, contributions bienvenues) |

### Ollama (Local — Gratuit)

Tourne entièrement sur votre machine. Pas de clé API, pas de coût, aucune donnée envoyée.

```yaml
# .lorerc
ai:
  provider: "ollama"
  model: "llama3.2"  # ou tout modèle : mistral, codellama, gemma2, etc.
```

```bash
# Installer et démarrer Ollama
brew install ollama    # macOS
ollama serve &         # démarrer le serveur
ollama pull llama3.2   # télécharger un modèle
```

Pas besoin de `lore config set-key` — Ollama n'a pas d'authentification.

| Élément | Détail |
|---------|--------|
| **Installer** | [ollama.com](https://ollama.com) ou `brew install ollama` |
| **Coût** | Gratuit (tourne sur votre matériel) |
| **Endpoint** | `http://localhost:11434` (défaut, automatique) |
| **Modèles** | Tout modèle Ollama — `ollama list` pour voir les installés |

### Tester le provider OpenAI sans crédits OpenAI

Ollama expose une API compatible OpenAI. Vous pouvez valider le provider `openai` contre Ollama :

```yaml
# .lorerc
ai:
  provider: "openai"
  model: "llama3.2"
  endpoint: "http://localhost:11434/v1/chat/completions"
```

```yaml
# .lorerc.local
ai:
  api_key: "any-value"  # Ollama ignore les clés API
```

> **Note :** Cela fonctionne uniquement pour le provider `openai`. Le provider `anthropic` utilise un format de requête différent qu'Ollama ne supporte pas.

### Comparaison des providers

| | **Anthropic** | **OpenAI** | **Ollama** |
|---|---|---|---|
| **Qualité** | Meilleure pour les docs techniques | Très bonne | Dépend du modèle |
| **Coût** | ~$0.01–0.05/appel | ~$0.001–0.01/appel | Gratuit |
| **Vie privée** | Données envoyées à l'API | Données envoyées à l'API | 100% local |
| **Setup** | Clé API + crédits | Clé API + crédits | Installer + pull modèle |
| **Hors ligne** | Non | Non | Oui |
| **Vitesse** | Rapide | Rapide | Dépend du matériel |

### Pas d'IA ? Pas de problème

`lore angela draft` et `lore angela draft --all` fonctionnent **100% hors ligne** sans aucune configuration. Ils analysent la structure des documents, les sections manquantes, la cohérence de style et les références croisées — tout localement.

Pour polish/review sans crédits API, voir le [workflow manuel via le chat Claude.ai](../faq.fr.md#jai-un-abonnement-claudeai-mais-pas-de-crédits-api-puis-je-utiliser-angela) dans la FAQ.

## Dépannage

### "Mon changement de config n'a aucun effet"

Vérifiez l'ordre de la cascade — une source de priorité plus élevée peut surcharger votre changement :

```
Flag CLI (--language fr)     ← priorité la plus haute
  ↓
Environnement (LORE_LANGUAGE)
  ↓
.lorerc.local
  ↓
.lorerc                      ← vous avez édité ici
  ↓
Défauts                      ← priorité la plus basse
```

Lancez `lore doctor --config` pour voir la configuration résolue.

### "Avertissement de clé inconnue"

```bash
lore doctor --config
# ✗ clé inconnue "ai.providr" — vouliez-vous dire "ai.provider" ?
```

Lore utilise la distance de Levenshtein pour suggérer des corrections de fautes de frappe.

## Voir aussi

- [`lore config`](../commands/config.md) — Voir et modifier la config
- [`lore doctor --config`](../commands/doctor.md) — Valider la config
