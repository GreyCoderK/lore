# lore show

Rechercher et afficher des documents de votre corpus.

## Synopsis

```
lore show [mot-clé] [flags]
```

## Qu'est-ce que ça fait ?

`lore show` est la façon de **lire** votre documentation. Donnez un mot-clé, et il trouve les documents correspondants. C'est un moteur de recherche pour l'historique des décisions de votre projet.

> **Analogie :** Si `.lore/docs/` est le journal de votre projet, `lore show` c'est feuilleter pour trouver la page où vous avez écrit à propos de "authentication".

## Scénario concret

> Code review. Le reviewer demande : "Pourquoi JWT au lieu de sessions ?" Au lieu de fouiller Slack :
>
> ```bash
> lore show "JWT"
> ```
>
> 3 secondes plus tard, le raisonnement complet est à l'écran — écrit le jour de la décision.

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

### Un résultat → Affiché directement

```bash
lore show "JWT auth"
# → Affiche le document complet
```

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

```bash
# Recherche basique
lore show "database"

# Filtrer par type
lore show "middleware" --decision

# Filtrer par date
lore show "api" --after 2026-03

# Pipe vers less
lore show "auth" --quiet | less

# Exporter un document
lore show "JWT auth" > auth-decision.md
```

## Questions fréquentes

### "Pas de résultats ?"

- Vérifiez l'orthographe — la recherche est exacte, pas fuzzy
- Essayez des termes plus larges : "auth" au lieu de "authentication middleware"
- Vérifiez que des documents existent : `lore list`

### "Différence avec `lore list` ?"

`lore list` = table des matières. `lore show` = lire un chapitre spécifique.

## Tips & Tricks

- **Pipe-friendly :** `lore show "auth" --quiet | less` pour paginer.
- **Export :** `lore show "JWT auth" > auth-decision.md` sauvegarde un document.
- **Combiner avec grep :** `lore show "api" --quiet | grep decision` — filtrer.
- **Pas de résultats ?** Termes plus larges. Fuzzy search arrive au Cercle 4.
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
