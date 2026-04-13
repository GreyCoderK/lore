# lore show

Rechercher et afficher des documents de votre corpus.

## Synopsis

```
lore show [mot-clé] [flags]
```

## Qu'est-ce que ça fait ?

`lore show` recherche votre corpus par mot-clé et affiche les documents correspondants. Donnez un terme, et il trouve chaque document dont le titre ou le contenu correspond.

> **Analogie :** Si `.lore/docs/` est votre corpus de projet, `lore show` est la recherche qui remonte l'entrée exacte où vous avez documenté "authentication".

## Scénario concret

> Code review. Le reviewer demande : "Pourquoi JWT au lieu de sessions ?" Au lieu de fouiller Slack :
>
> ```bash
> lore show "JWT"
> ```
>
> 3 secondes plus tard, le raisonnement complet est à l'écran — écrit le jour de la décision.

![lore show](../assets/vhs/show-search.gif)
<!-- Generate: vhs assets/vhs/show-search.tape -->

## Arguments

| Argument | Requis | Description |
|----------|--------|-------------|
| `mot-clé` | Oui (sauf `--all`) | Terme de recherche — correspond aux titres et contenus |

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--type <type>` | string | — | Afficher uniquement ce type |
| `--after <date>` | string | — | Documents après cette date (`YYYY-MM` ou `YYYY-MM-DD`) |
| `--all` | bool | `false` | Tous les documents (préférez `lore list`) |
| `--quiet` | bool | `false` | Sortie machine (tab-séparée) |
| `--feature` | bool | — | Raccourci pour `--type feature` |
| `--decision` | bool | — | Raccourci pour `--type decision` |
| `--bugfix` | bool | — | Raccourci pour `--type bugfix` |
| `--refactor` | bool | — | Raccourci pour `--type refactor` |
| `--note` | bool | — | Raccourci pour `--type note` |

> Les raccourcis de type sont mutuellement exclusifs.

## Comment la recherche fonctionne

Lore recherche le **titre** et le **contenu** de chaque document dans `.lore/docs/`. La recherche est exacte, pas fuzzy — un mot-clé doit apparaître tel quel pour correspondre.

### Un résultat → Affiché directement

```bash
lore show "JWT auth"
```

```markdown
---
type: decision
date: 2026-02-15
commit: b2c3d4e
branch: feature/jwt-auth
scope: auth
---
# Add JWT auth middleware

## Why
L'authentification stateless passe mieux à l'échelle que les sessions...
```

> **Branche et scope** sont capturés automatiquement au moment du commit (voir [Branch Awareness](../guides/configuration.fr.md#branch-awareness-branche-git)). Ils sont omis du front matter lorsqu'ils ne sont pas disponibles (HEAD détachée, absence de scope conventionnel).

### Plusieurs résultats → Sélection interactive (TTY)

```
  1  decision  Add JWT auth middleware        2026-02-15
  2  refactor  Extract auth middleware        2026-03-01
  3  feature   Add OAuth2 provider           2026-03-10

Select [1-3]:
```

### Plusieurs résultats → Liste (Non-TTY / Quiet)

```bash
lore show "auth" --quiet
# decision	Add JWT auth middleware	2026-02-15
# refactor	Extract auth middleware	2026-03-01
```

## Exemples

### Recherche basique

```bash
# Trouver les documents sur "database"
lore show "database"

# Toutes les décisions sur l'API
lore show "api" --decision

# Documents récents
lore show "rate" --after 2026-03
```

### Combiné avec d'autres commandes

```bash
# Pipe vers less
lore show "auth" --quiet | less

# Compter les décisions sur un sujet
lore show "auth" --decision --quiet | wc -l

# Exporter un document
lore show "JWT auth" > auth-decision.md
```

## Questions fréquentes

### "Pas de résultats ?"

- Vérifiez l'orthographe — la recherche est exacte, pas fuzzy
- Essayez des termes plus larges : "auth" au lieu de "authentication middleware"
- Vérifiez que des documents existent : `lore list`

### "Différence avec `lore list` ?"

| Commande | But |
|----------|-----|
| `lore list` | Afficher TOUS les documents avec métadonnées (type, date, tags) |
| `lore show` | **Rechercher** des documents spécifiques par mot-clé et afficher le contenu |

`lore list` = table des matières. `lore show` = lire un chapitre spécifique.

## Tips & Tricks

- **Pipe-friendly :** `lore show "auth" --quiet | less` pour paginer.
- **Export :** `lore show "JWT auth" > auth-decision.md` sauvegarde un document.
- **Combiner avec grep :** `lore show "api" --quiet | grep decision` — filtrer.
- **Pas de résultats ?** Termes plus larges — la recherche est exacte, pas fuzzy.
- **Raccourcis type :** `--decision` est plus rapide que `--type decision`.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Correspondance trouvée |
| `2` | Aucune correspondance (pas une erreur) |
| `3` | Pas de mot-clé fourni |

## Voir aussi

- [lore list](list.md) — Parcourir tous les documents
- [lore status](status.md) — Statistiques et santé du corpus
