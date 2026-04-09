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
  # api_key: ""             # Clé API (préférer set-key ou env var LORE_AI_API_KEY)
  # endpoint: ""            # URL endpoint custom (pour Ollama, Groq, Together, etc.)
  # timeout: 60s            # Timeout pour les appels API IA

angela:
  mode: draft               # Mode par défaut : "draft" (zéro-API) ou "polish" (1 appel API)
  # max_tokens: 8192         # Optionnel : surcharge le max tokens auto-calculé (défaut : dynamique par mode)

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
  max_tokens: 8192
```

```yaml
# .lorerc.local — personnel (gitignore, chmod 600)
ai:
  api_key: "sk-ant-..."
```

Chaque membre de l'équipe stocke sa propre clé API. La config partagée définit le fournisseur et le modèle.

> **`angela.max_tokens`** — Quand défini, cette valeur remplace la limite auto-calculée. Par défaut, Angela calcule `max_tokens` dynamiquement selon la taille du document (word_count × 1.3 × 1.8, plafonné à 8192, plancher 512). Si vous définissez `angela.max_tokens: 10000` dans `.lorerc`, cette valeur est toujours utilisée. Augmentez-la si Angela avertit que « l'entrée dépasse la sortie max » ou si les réponses sont tronquées.

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

**Étape 1 — Obtenir une clé API :**

1. Allez sur [console.anthropic.com](https://console.anthropic.com) et inscrivez-vous (ou connectez-vous)
2. Naviguez vers **Settings → API Keys → Create Key**
3. Copiez la clé (commence par `sk-ant-...`)
4. Ajoutez des crédits : **Settings → Plans & Billing → Add Credits** (minimum $5)

> **Important :** Un abonnement chat Claude.ai (Pro, Team) n'inclut PAS de crédits API. L'API est un produit séparé facturé sur [console.anthropic.com](https://console.anthropic.com). Vous avez besoin de crédits même si vous payez pour Claude.ai.

**Étape 2 — Configurer Lore :**

```yaml
# .lorerc
ai:
  provider: "anthropic"
  model: "claude-sonnet-4-20250514"  # ou claude-haiku-4-5-20251001 (moins cher)
```

```bash
# Stocker la clé API dans le trousseau OS
lore config set-key anthropic
# → Entrez la clé API : sk-ant-...
```

**Étape 3 — Tester :**

```bash
lore angela draft --all                                  # gratuit, pas d'API
lore angela polish <votre-doc>.md --dry-run              # 1 appel API, aperçu seulement
lore angela review                                       # 1 appel API, analyse du corpus
```

| Élément | Détail |
|---------|--------|
| **Inscription** | [console.anthropic.com](https://console.anthropic.com) |
| **Clés API** | Settings → API Keys → Create Key |
| **Ajouter des crédits** | Settings → Plans & Billing → Add Credits ($5 minimum) |
| **Coût par polish** | ~$0.01–0.05 (Sonnet), ~$0.001 (Haiku) |
| **Endpoint** | `https://api.anthropic.com/v1/messages` (automatique) |
| **Modèles** | `claude-sonnet-4-20250514` (recommandé), `claude-haiku-4-5-20251001` (le moins cher) |

### OpenAI (GPT)

**Étape 1 — Obtenir une clé API :**

1. Allez sur [platform.openai.com/api-keys](https://platform.openai.com/api-keys) et inscrivez-vous (ou connectez-vous)
2. Cliquez **Create new secret key**, nommez-la (ex : "lore")
3. Copiez la clé (commence par `sk-...`)
4. Ajoutez des crédits : **Settings → Billing → Add payment method** puis **Add credits** ($5 minimum)

> **Note :** Un compte API OpenAI est séparé d'un abonnement ChatGPT. L'API utilise des crédits prépayés — pas de facturation récurrente sauf si vous activez l'auto-recharge.

**Étape 2 — Configurer Lore :**

```yaml
# .lorerc
ai:
  provider: "openai"
  model: "gpt-4o-mini"  # le moins cher, ou gpt-4o pour la meilleure qualité
```

```bash
# Stocker la clé API dans le trousseau OS
lore config set-key openai
# → Entrez la clé API : sk-...
```

**Étape 3 — Tester :**

```bash
lore angela polish <votre-doc>.md --dry-run              # aperçu des changements
lore angela review                                       # analyse du corpus
```

| Élément | Détail |
|---------|--------|
| **Inscription** | [platform.openai.com](https://platform.openai.com) |
| **Clés API** | [platform.openai.com/api-keys](https://platform.openai.com/api-keys) |
| **Ajouter des crédits** | Settings → Billing → Add credits ($5 minimum) |
| **Coût par polish** | ~$0.001 (gpt-4o-mini), ~$0.01–0.05 (gpt-4o) |
| **Endpoint** | `https://api.openai.com/v1/chat/completions` (automatique) |
| **Endpoint custom** | Définir `ai.endpoint` pour APIs compatibles (Azure OpenAI, Ollama — voir ci-dessous) |
| **Modèles** | `gpt-4o-mini` (le moins cher), `gpt-4o` (meilleure qualité), `gpt-4.1-mini`, `gpt-4.1` |

### Ollama (Local — Gratuit)

Tourne entièrement sur votre machine. Pas de clé API, pas de coût, aucune donnée envoyée.

**Étape 1 — Installer Ollama :**

=== "macOS"

    ```bash
    brew install ollama
    ```

=== "Linux"

    ```bash
    curl -fsSL https://ollama.com/install.sh | sh
    ```

=== "Windows"

    Téléchargez l'installateur depuis [ollama.com/download](https://ollama.com/download)

**Étape 2 — Télécharger un modèle et démarrer :**

```bash
ollama serve &            # démarrer le serveur (port 11434)
ollama pull llama3.2      # télécharger un modèle (~2Go)
```

Autres modèles recommandés :

| Modèle | Taille | Qualité | Vitesse |
|--------|--------|---------|---------|
| `llama3.2` | 2Go | Bon pour les docs courtes | Rapide |
| `llama3.1:8b` | 4.7Go | Meilleure qualité | Moyen |
| `llama3.1:70b` | 40Go | Proche de GPT-4o | Lent (nécessite 64Go RAM) |
| `mistral` | 4.1Go | Bon polyvalent | Rapide |
| `codellama` | 3.8Go | Meilleur pour les docs avec code | Rapide |
| `gemma2` | 5.4Go | Bon pour l'écriture technique | Moyen |

**Étape 3 — Configurer Lore :**

```yaml
# .lorerc
ai:
  provider: "ollama"
  model: "llama3.2"       # ou tout modèle de `ollama list`
```

Pas besoin de `lore config set-key` — Ollama n'a pas d'authentification.

**Étape 4 — Tester :**

```bash
ollama list                                               # vérifier que le modèle est installé
lore doctor --config                                     # vérifier que le provider est détecté
lore angela polish <votre-doc>.md --dry-run              # tester polish
lore angela review                                       # tester review
```

| Élément | Détail |
|---------|--------|
| **Télécharger** | [ollama.com/download](https://ollama.com/download) ou `brew install ollama` |
| **Coût** | Gratuit (tourne sur votre matériel) |
| **Endpoint** | `http://localhost:11434` (automatique) |
| **Parcourir les modèles** | [ollama.com/library](https://ollama.com/library) |
| **Lister les installés** | `ollama list` |
| **Télécharger un modèle** | `ollama pull <nom-du-modèle>` |

> **Conseil qualité :** Les petits modèles (llama3.2, phi3) peuvent halluciner ou produire du texte générique. Pour de meilleurs résultats, utilisez un modèle d'au moins 8B paramètres (llama3.1:8b, mistral) et écrivez des premiers brouillons détaillés avant de polir.

### Tester le code path OpenAI via Ollama (gratuit)

Ollama expose une API compatible OpenAI sur `/v1/chat/completions`. Cela permet de tester le provider `openai` sans payer de crédits OpenAI :

```yaml
# .lorerc.local
ai:
  provider: "openai"
  model: "llama3.2"
  endpoint: "http://localhost:11434/v1/chat/completions"
  api_key: "unused"       # Ollama ignore les clés API, mais le champ doit être non-vide
```

```bash
# Vérifier que ça marche
ollama serve &
lore angela polish <votre-doc>.md --dry-run
```

> **Note :** Cela fonctionne uniquement pour le provider `openai`. Le provider `anthropic` utilise un format de requête différent qu'Ollama ne supporte pas.

### Comparaison des providers

| | **Anthropic** | **OpenAI** | **Ollama** |
|---|---|---|---|
| **Qualité** | Meilleure pour les docs techniques | Très bonne | Dépend de la taille du modèle |
| **Coût** | ~$0.01–0.05/appel | ~$0.001–0.01/appel | Gratuit |
| **Vie privée** | Données envoyées à l'API | Données envoyées à l'API | 100% local |
| **Temps de setup** | 5 min (inscription + crédits) | 5 min (inscription + crédits) | 2 min (installer + pull) |
| **Hors ligne** | Non | Non | Oui |
| **Vitesse** | Rapide (~3s) | Rapide (~3s) | Dépend du matériel (5-30s) |
| **Inscription** | [console.anthropic.com](https://console.anthropic.com) | [platform.openai.com](https://platform.openai.com) | Pas de compte nécessaire |

### Pas d'IA ? Pas de problème

`lore angela draft` et `lore angela draft --all` fonctionnent **100% hors ligne** sans aucune configuration. Ils analysent la structure des documents, les sections manquantes, la cohérence de style et les références croisées — tout localement.

Pour polish/review sans credits API, voir le [workflow manuel via le chat Claude.ai](../faq.fr.md#jai-un-abonnement-claudeai-mais-pas-de-crédits-api-puis-je-utiliser-angela) dans la FAQ.

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
