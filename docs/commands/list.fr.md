# lore list

Lister tous les documents du corpus avec leurs métadonnées.

## Synopsis

```
lore list [flags]
```

## Qu'est-ce que ça fait ?

Affiche un tableau de chaque document dans `.lore/docs/`, trié par date (plus récent d'abord). C'est la table des matières du corpus de votre projet.

> **Analogie :** `lore list` est l'index d'un livre — tous les chapitres d'un coup avec leurs dates et types. `lore show` lit un chapitre spécifique.

## Scénario concret

> Jour de release. Vous devez voir tout ce qui a été documenté depuis le dernier tag :
>
> ```bash
> lore list --type decision
> ```
>
> 12 décisions, triées par date. Vous savez exactement ce qui a changé et pourquoi.

![lore list](../assets/vhs/list.gif)
<!-- Generate: vhs assets/vhs/list.tape -->

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--type <type>` | string | — | Afficher uniquement ce type (`decision`, `feature`, `bugfix`, `refactor`, `note`, `release`) |
| `--quiet` | bool | `false` | Sortie tab-séparée pour le scripting |

## Sortie

```bash
lore list
```

```
TYPE       SLUG                                  DATE        TAGS
decision   database-selection-2026-02-10          2026-02-10  2 tags
feature    add-jwt-auth-2026-02-15                2026-02-15  3 tags
feature    add-rate-limiting-2026-03-16           2026-03-16  1 tag
refactor   extract-auth-middleware-2026-03-01     2026-03-01  0 tags
note       meeting-api-versioning-2026-03-20      2026-03-20  1 tag
```

## Exemples

### Parcourir par type

```bash
# Toutes les décisions — idéal avant les revues d'architecture
lore list --type decision

# Tous les bugfixes — utile pour les post-mortems
lore list --type bugfix

# Toutes les notes — trouver ce résumé de réunion
lore list --type note
```

### Scripting

```bash
# Compter le total de documents
lore list --quiet | wc -l

# Extraire juste les noms de fichiers
lore list --quiet | cut -f2

# Trouver les documents de mars
lore list --quiet | grep "2026-03"

# Boucler sur les résultats
lore list --quiet | while IFS=$'\t' read -r type slug date tags; do
  echo "Traitement : $slug"
done
```

### Combiné avec d'autres commandes

```bash
# List → Choisir → Lire (workflow en deux étapes)
lore list --type decision
# → Voir : database-selection-2026-02-10
lore show "database"
# → Document complet affiché
```

## Différence avec `lore show`

| | `lore list` | `lore show` |
|---|---|---|
| **But** | Voir TOUS les documents d'un coup | CHERCHER des documents spécifiques |
| **Sortie** | Tableau avec métadonnées (type, date, tags) | Contenu complet du document |
| **Entrée** | Pas d'arguments nécessaires | Nécessite un mot-clé |
| **Analogie** | Table des matières | Lire un chapitre |

## Tips & Tricks

- **Avant une code review :** `lore list --type decision` montre tous les choix architecturaux — contexte parfait pour les reviewers.
- **Avant une release :** `lore list` montre tout. Combinez avec `lore release` pour les notes.
- **Comptage rapide :** `lore list --quiet | wc -l` — combien de documents ?
- **Corpus vide ?** `lore list` affiche un message utile avec des suggestions.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Succès (même si vide — affiche un message utile) |
| `1` | Erreur (`.lore/` non trouvé) |

## Voir aussi

- [lore show](show.md) — Chercher et lire un document spécifique
- [lore status](status.md) — Dashboard santé avec statistiques
- [lore release](release.md) — Générer des notes de version depuis le corpus
