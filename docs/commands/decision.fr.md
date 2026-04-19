---
type: reference
date: 2026-04-12
status: published
related:
  - ../guides/contextual-detection.md
  - ../guides/configuration.md
  - status.md
angela_mode: polish
---
# lore decision

Comprendre comment Lore décide quels commits ont besoin de documentation.

## Synopsis

```
lore decision [flags]
```

## Qu'est-ce que ça fait ?

Lore ne demande pas de documenter chaque commit. Un fix de typo dans le README n'a pas besoin d'un "pourquoi." Mais ajouter un nouveau système d'authentification, si. Le **Decision Engine** détermine lequel est lequel.

`lore decision` affiche le scoring du moteur pour n'importe quel commit afin de comprendre et ajuster son comportement.

## Scénario concret

> Vous avez commité un fix de typo dans le README. Lore vous a posé les 5 questions. C'était excessif. Pourquoi ?
>
> ```bash
> lore decision --explain HEAD
> # Score : 72/100 — feat détecté dans le type de commit
> ```
>
> Votre message commençait par `feat:` au lieu de `docs:`. Le Decision Engine l'a scoré haut. Corrigez le préfixe et Lore auto-skip la prochaine fois.

![lore decision](../assets/vhs/decision.gif)
<!-- Generate: vhs assets/vhs/decision.tape -->

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--explain <ref>` | string | HEAD | Quel commit analyser |
| `--calibration` | bool | `false` | Afficher les métriques de précision du moteur |

## Comment le scoring fonctionne

Le moteur combine **5 signaux** pour un score de 0 à 100 :

```mermaid
graph LR
    A[Votre Commit] --> B["🏷️ Signal 1 : Type Conv (0-30)"]
    A --> C["📏 Signal 2 : Taille Diff (0-25)"]
    A --> D["🔍 Signal 3 : Contenu (0-20)"]
    A --> E["📁 Signal 4 : Fichiers (0-15)"]
    A --> F["📊 Signal 5 : Historique (0-10)"]
    B --> G[Score Total : 0-100]
    C --> G
    D --> G
    E --> G
    F --> G
    G --> H{Que se passe-t-il ?}
    H -->|"≥60"| I["📝 Questions complètes"]
    H -->|"35-59"| J["📋 Questions réduites"]
    H -->|"15-34"| K["🤔 Suggérer d'ignorer"]
    H -->|"<15"| L["⏭️ Auto-skip"]
```

### Les 5 signaux expliqués

| # | Signal | Ce qu'il regarde | Score haut = | Score bas = |
|---|--------|------------------|--------------|-------------|
| 1 | **Type Conv** | `feat:`, `fix:`, `docs:`... | `feat` ou `fix` = important | `docs`, `style`, `ci` = trivial |
| 2 | **Taille Diff** | Lignes ajoutées + supprimées | 10-500 lignes = optimal | Trop petit (1 ligne) ou trop gros (2000+) |
| 3 | **Contenu** | Mots-clés dans le diff | Auth, database, security = critique | Seulement des commentaires changés |
| 4 | **Fichiers** | Quels fichiers ont changé | Fichiers source `.go` = valeur haute | Tests, configs = plus bas |
| 5 | **Historique** | Taux de documentation passé | Ce scope est souvent documenté | Ce scope est rarement documenté |

### Score → Action

| Score | Action | Ce que vous voyez |
|-------|--------|-------------------|
| **≥ 60** | `ask-full` | Les 5 questions (Type, What, Why, Alternatives, Impact) |
| **35–59** | `ask-reduced` | 2 questions (Type, What) |
| **15–34** | `suggest-skip` | "Ignorer la documentation pour ce commit ? [O/n]" |
| **< 15** | `auto-skip` | Rien ne se passe (silencieux) |

## Sortie

```bash
lore decision --explain HEAD
```

```
Commit      e4f5a6b
Subject     feat(auth): add JWT middleware
Score       72/100
Action      ask-full
Confidence  95.0%

SIGNAL       SCORE  RAISON
conv-type    +15    feat → always_ask override
diff-size    +22    changement modéré (180 lignes ajoutées)
content      +18    mots-clés critiques : auth, middleware, token
files        +12    3 fichiers .go dans internal/ (haute valeur)
lks-history  +5     scope "auth" — 60% taux de documentation

Prefill:
  What: "Add JWT middleware" (extrait du subject)
  Why:  — (pas de corps de commit, confidence 0.0)
```

### Ce que signifie "Confidence"

- **95-100 % :** Les 5 signaux calculés, store.db disponible. Très fiable.
- **80 % :** Store indisponible (Signal 5 manquant). Toujours bon.
- **< 80 % :** Certains signaux n'ont pas pu être calculés. Le score peut être imprécis.

### Ce que signifie "Prefill"

Lore **pré-remplit** les champs "What" et "Why" depuis votre message de commit :

- **What :** Extrait du subject du commit (première ligne)
- **Why :** Extrait du corps du commit (si présent)
- **Why Confidence :** 0.0 = pas de corps trouvé, 1.0 = justification claire dans le corps

> **Astuce :** Écrivez des messages de commit descriptifs et Lore pré-remplit vos réponses automatiquement.

## Overrides (contourner le scoring)

Forcez certains comportements dans `.lorerc` :

```yaml
decision:
  always_ask: [feat, breaking]          # Toujours poser toutes les questions pour ces types
  always_skip: [docs, style, ci, build] # Toujours ignorer ces types
  critical_scopes: [security, payments] # Toujours documenter ces zones de code
```

| Override | Effet | Exemple |
|----------|-------|---------|
| `always_ask` | Score ignoré → questions complètes | Chaque commit `feat:` est documenté |
| `always_skip` | Score ignoré → skip silencieux | Les changements `ci:` ne déclenchent jamais de questions |
| `critical_scopes` | Score boosté → questions complètes | Les changements dans `payments/` sont toujours documentés |

## Mode calibration

Après 20+ commits, le moteur apprend vos patterns. Vérifiez sa précision :

```bash
lore decision --calibration
```

Affiche les métriques de précision : taux de réussite, faux positifs (invité alors que ça ne l'aurait pas dû), et faux négatifs (ignoré alors que ça ne l'aurait pas dû).

## Ajuster les seuils

Si le moteur ignore trop de commits qui vous importent, abaissez les seuils :

```yaml
# .lorerc
decision:
  threshold_full: 50      # Défaut : 60. Plus bas = plus de questions complètes
  threshold_reduced: 25    # Défaut : 35. Plus bas = plus de questions réduites
  threshold_suggest: 10    # Défaut : 15. Plus bas = moins d'auto-skips
```

Si le moteur demande trop souvent, relevez-les.

## Tips & Tricks

- **"Pourquoi mon commit a été ignoré ?"** → `lore decision --explain <hash>` montre le détail du score.
- **Forcer la documentation :** `always_ask: [feat]` dans `.lorerc` pour toujours documenter les features.
- **Skip ponctuel :** Ajoutez `[doc-skip]` dans votre message de commit pour un skip unique.
- **Ajuster progressivement :** Commencez par les défauts, puis ajustez après 50+ commits via `--calibration`.
- **Le moteur apprend :** Après 20 commits, le Signal 5 (Historique) s'active et s'adapte à vos patterns.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Succès |
| `1` | Erreur |

## Questions fréquentes

### "Le score semble faux pour mon commit"

Lancez `lore decision --explain HEAD` pour voir quel signal a le plus contribué. Cause courante : le message commence par `feat:` au lieu de `docs:`, ce qui gonfle le score. Corrigez le préfixe et le moteur s'adapte.

### "Puis-je forcer la documentation pour un commit ?"

Oui. Ajoutez `[doc-skip]` dans votre message de commit pour un skip unique. Ou utilisez `always_ask`/`always_skip` dans `.lorerc` pour des overrides permanents par type.

### "Combien de temps avant que le moteur apprenne ?"

Le Signal 5 (Historique LKS) s'active après 20 commits. La précision s'améliore notablement après 50+. Jusqu'alors, les quatre autres signaux sont généralement suffisants.

## Voir aussi

- [Détection contextuelle](../guides/contextual-detection.md) — Règles qui s'exécutent *avant* le Decision Engine
- [Configuration](../guides/configuration.md) — Ajuster les seuils et overrides
- [lore status](status.md) — Santé documentaire globale
