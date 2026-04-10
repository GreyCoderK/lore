# lore angela draft

Analyse structurelle zéro-API de vos documents — pas d'internet requis.

## Synopsis

```
lore angela draft [fichier] [flags]
```

## Qu'est-ce que ça fait ?

`lore angela draft` c'est comme avoir un coach d'écriture qui relit votre document — sauf que ce coach travaille **entièrement hors ligne**. Il vérifie la structure, le style et la cohérence sans aucun appel réseau ni clé API.

> **Analogie :** Imaginez un correcteur orthographique, mais au lieu de vérifier l'orthographe, il vérifie : "Avez-vous expliqué *pourquoi* ? Avez-vous mentionné les alternatives ? Est-ce cohérent avec vos autres documents ?" Tout localement, tout gratuit.

## Scénario concret

> Avant de pousser votre PR, vous voulez vérifier que les 3 nouveaux documents sont bien structurés — sans dépenser de crédits API :
>
> ```bash
> lore angela draft --all
> # 2 docs à revoir, 1 est excellent
> ```
>
> Gratuit, hors ligne, instantané. Corrigez les problèmes, PUIS polissez avec l'IA.

![lore angela draft](../assets/vhs/angela-draft-polish.gif)
<!-- Generate: vhs assets/vhs/angela-draft-polish.tape -->

## Arguments

| Argument | Requis | Description |
|----------|--------|-------------|
| `fichier` | Non | Document spécifique à analyser (défaut : le plus récent) |

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--all` | bool | `false` | Analyser chaque document du corpus |
| `--verbose`, `-v` | bool | `false` | Avec `--all` : afficher chaque suggestion en détail (par défaut, seuls les avertissements sont affichés) |
| `--path` | string | `.lore/docs` | Chemin vers un répertoire markdown (mode autonome — pas de `lore init` requis) |

## Mode autonome

Angela peut analyser **n'importe quel répertoire de fichiers Markdown**, même sans `lore init` :

```bash
# Analyser les docs d'un projet externe
lore angela draft --all --path ./mon-projet/docs

# Un seul fichier dans un répertoire personnalisé
lore angela draft --path ./wiki api-guide.md
```

En mode autonome :
- Les fichiers **avec** front matter YAML reçoivent l'analyse complète (type, tags, scope)
- Les fichiers **sans** front matter reçoivent des métadonnées synthétiques (type=note, tags depuis le nom de fichier)
- Pas besoin de `.lorerc` — les valeurs par défaut s'appliquent

Ce mode rend Angela utilisable comme **gate de qualité en CI** sur n'importe quelle documentation Markdown. Voir le guide [Angela en CI](../guides/angela-ci.md).

## Ce qui est vérifié

| Catégorie | Ce que ça cherche | Exemple de finding |
|-----------|-------------------|-------------------|
| **Structure** | Sections manquantes (Why, Alternatives, Impact) | "Section 'Alternatives Considered' manquante" |
| **Style** | Voix passive, langage vague, problèmes de ton | "Voix passive excessive dans la section Why" |
| **Cohérence** | Contradictions ou connexions avec d'autres docs | "Lié : feature-add-auth-2026-02-15.md" |
| **Complétude** | Sections vides ou trop courtes | "La section Why ne fait que 5 mots — enrichissez" |

## Comprendre les sévérités

| Sévérité | Signification | Action |
|----------|---------------|--------|
| **error** | Quelque chose d'important manque | Corrigez avant de considérer le doc "terminé" |
| **warning** | Pourrait être mieux | Améliorez quand vous avez le temps |
| **info** | Informatif — connexions et contexte | Bon à savoir, pas d'action nécessaire |

## Sortie (document unique)

```bash
lore angela draft decision-database-2026-02-10.md
```

```
lore angela draft — decision-database-2026-02-10.md
  Reviewed by: Sialou + Doumbia  (relevance: 7)

  error    structure       Section "Impact" manquante — les décisions doivent décrire les conséquences
  warning  tone            "On a juste pris PostgreSQL" — évitez "juste", ça affaiblit la décision
  info     coherence       Lié : feature-user-model-2026-02-12.md (mentionne le même schéma)

3 suggestions
```

## Sortie (corpus entier `--all`)

```bash
lore angela draft --all
```

Par défaut, Angela affiche la ligne de résumé pour chaque document et le
détail inline de chaque **avertissement** (les blockers à corriger en priorité) :

```
lore angela draft --all — 12 documents

  B    review   decision-database-2026-02-10.md      3 suggestions (2 avertissements)
         warning  structure      Section "Impact" manquante
         warning  completeness   La section "Why" ne fait que 5 mots
  C    review   feature-rate-limit-2026-03-16.md      1 suggestions
  A    ok       refactor-extract-auth-2026-03-01.md

2/12 documents nécessitent une attention. 4 suggestions au total.
Utilisez --verbose (-v) pour voir toutes les suggestions en détail.
```

### `--verbose` / `-v`

Pour voir toutes les suggestions (info, warning, error), passez `-v` :

```bash
lore angela draft --all -v
```

```
  B    review   decision-database-2026-02-10.md      3 suggestions (2 avertissements)
         warning  structure      Section "Impact" manquante
         warning  completeness   La section "Why" ne fait que 5 mots
         info     coherence      Lié : feature-user-model-2026-02-12.md
```

## Flux

```mermaid
graph TD
    A[lore angela draft] --> B{--all ?}
    B -->|Non| C[Charger un document]
    B -->|Oui| D[Charger tout le corpus]
    C --> E[Parser front matter + contenu]
    D --> E
    E --> F[Vérifier structure : sections présentes ?]
    F --> G[Vérifier style : ton, clarté, longueur]
    G --> H[Vérifier cohérence : références croisées]
    H --> I[Scorer avec les personas]
    I --> J[Générer suggestions]
    J --> K[Afficher rapport]
```

## Questions fréquentes

### "Ai-je besoin d'une clé API ?"

**Non.** `angela draft` est 100% local. Utilise des règles et heuristiques intégrées. C'est un linter sophistiqué pour la documentation.

### "Différence entre `draft` et `polish` ?"

| | `angela draft` | `angela polish` |
|---|---|---|
| **Réseau** | Aucun (hors ligne) | 1 appel API |
| **Coût** | Gratuit | Utilise des crédits API |
| **Ce qu'il fait** | Signale les problèmes | Réécrit le document |
| **Sortie** | Liste de suggestions | Diff interactif |

> **Bonne pratique :** Toujours `draft` d'abord (gratuit), corriger les problèmes faciles, puis `polish` (coûte des crédits).

### "C'est quoi les 'personas' ?"

Angela utilise 6 relecteurs virtuels avec des perspectives différentes. Les 3 meilleurs sont activés selon le type de document et son contenu :

| Persona | Icône | Focus |
|---------|-------|-------|
| **Affoue** (Storyteller) | 📖 | Clarté narrative, sections "Why" |
| **Sialou** (Tech Writer) | ✏️ | Précision technique, structure |
| **Kouame** (QA Reviewer) | 🔍 | Critères de validation, cas limites |
| **Doumbia** (Architect) | 🏗️ | Compromis, conception système |
| **Gougou** (UX Designer) | 🎨 | Empathie utilisateur, accessibilité |
| **Beda** (Business Analyst) | 📊 | Valeur business, exigences |

Chaque persona exécute des vérifications locales et produit des suggestions typées. Par exemple, Affoue vérifie que la section "Why" raconte une histoire plutôt que de lister des bullets. Kouame vérifie que les affirmations ont des critères de vérification.

## Tips & Tricks

- **Avant chaque PR :** `lore angela draft --all` pour attraper les problèmes de qualité.
- **`draft` avant `polish` :** Corrigez les problèmes gratuits d'abord, puis dépensez les crédits.
- **Le score est relatif :** 7/10 c'est bien, 9/10 c'est excellent. Ne visez pas 10/10 sur chaque doc.
- **Personnalisez les règles :** Dans `.lorerc` sous `angela.style_guide`.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Succès (même si suggestions trouvées) |
| `1` | Erreur (`.lore/` non trouvé, fichier non trouvé) |

## Voir aussi

- [lore angela polish](angela-polish.md) — Réécriture assistée par IA (étape suivante)
- [lore angela review](angela-review.md) — Revue de cohérence corpus
- [Types de documents](../guides/document-types.md) — Quelles sections chaque type attend
