# lore new

Créer une entrée de documentation à la demande.

## Synopsis

```
lore new [type] ["what"] ["why"] [flags]
```

## Qu'est-ce que ça fait ?

`lore new` vous permet d'écrire une entrée de documentation **manuellement**, sans attendre un commit. C'est comme ouvrir votre journal de projet et écrire une nouvelle page quand vous voulez.

**Trois façons de l'utiliser :**

| Mode | Commande | Quand l'utiliser |
|------|----------|------------------|
| **Interactif** | `lore new` | Le plus courant — Lore pose les questions |
| **One-liner** | `lore new feature "add auth" "stateless scales"` | Capture rapide quand vous savez quoi écrire |
| **Rétroactif** | `lore new --commit abc1234` | Documenter un commit passé que vous avez raté |

> **Analogie :** Si le hook post-commit est comme un journaliste qui vous suit en temps réel, `lore new` c'est comme s'asseoir avec ce journaliste pour une interview dédiée sur quelque chose que vous avez fait plus tôt.

## Scénario concret

> Réunion du matin. L'équipe a décidé de migrer de MongoDB vers PostgreSQL. Pas de code encore — juste une décision. Vous voulez la capturer avant que les détails s'estompent :
>
> ```bash
> lore new decision "switch to PostgreSQL" "intégrité relationnelle pour les transactions ACID"
> ```
>
> Ou plus tard, vous réalisez que 3 commits de la semaine dernière n'ont jamais été documentés :
>
> ```bash
> git log --oneline -5
> lore new --commit abc1234
> ```

![lore new --commit](../assets/vhs/new-retroactive.gif)
<!-- Generate: vhs assets/vhs/new-retroactive.tape -->

## Arguments

| Argument | Requis | Description | Exemple |
|----------|--------|-------------|---------|
| `type` | Non | Type de document | `decision`, `feature`, `bugfix`, `refactor`, `note` |
| `what` | Non | Résumé en une ligne (entre guillemets) | `"add JWT auth middleware"` |
| `why` | Non | Raison (entre guillemets) | `"stateless auth scales better"` |

Sans arguments, Lore pose les questions interactivement.

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--commit <hash>` | string | — | Documenter un commit passé spécifique |
| `--type <type>` | string | — | Pré-définir le type de document |

## Types de Documents

| Type | Icône | Quand l'utiliser | Exemple |
|------|-------|------------------|---------|
| `decision` | 🏗️ | Vous avez choisi entre des options | "Pourquoi PostgreSQL plutôt que MongoDB" |
| `feature` | ✨ | Vous avez construit quelque chose de nouveau | "Ajouter le middleware rate limiting" |
| `bugfix` | 🐛 | Vous avez corrigé un bug | "Corriger la race condition du refresh token" |
| `refactor` | ♻️ | Vous avez restructuré du code | "Extraire l'auth dans un package dédié" |
| `note` | 📝 | Connaissance générale | "Notes de réunion : stratégie de versioning API" |

> **Astuce :** Pas sûr du type ? Demandez-vous : "Est-ce que je choisis entre des options ?" → `decision`. "Est-ce que je construis ?" → `feature`. "Est-ce que je répare ?" → `bugfix`. Toujours pas sûr ? → `note`.

## Le flux de questions

En mode interactif :

![lore new interactif](../assets/vhs/interactive.gif)
<!-- Generate: vhs assets/vhs/interactive.tape -->

```
$ lore new

? Type : [Utilisez les flèches]
  > feature
    bugfix
    decision
    refactor
    note
    release
    summary

? What changed : Add JWT auth middleware
  (Pré-rempli depuis le contexte. Éditez ou appuyez sur Entrée.)

? Why was this done : L'authentification stateless scale mieux que
  les sessions côté serveur. On ne veut pas gérer Redis pour l'état de session.

✓ Capturé : decision-add-jwt-auth-middleware-2026-03-16.md
```

Pour le type **decision**, 2 questions bonus :

```
? Alternatives considered : Auth par sessions avec Redis ; OAuth2 seul
? Impact : Tous les endpoints API nécessitent maintenant un Bearer token
```

### Mode Express

Si vous répondez aux 3 questions en moins de 3 secondes, Lore entre en **mode express** et saute les questions bonus.

## Ce qui est créé

Un fichier Markdown dans `.lore/docs/` :

```markdown
---
type: decision
date: 2026-03-16
status: draft
commit: abc1234567890abcdef
generated_by: manual
---
# Switch to PostgreSQL

## Why
Intégrité relationnelle pour les comptes utilisateurs. Besoin de transactions
ACID pour le flux de paiement, et le driver pgx de PostgreSQL est excellent.

## Alternatives Considered
- MongoDB : Schéma flexible mais on réimplémenterait les clés étrangères
- SQLite : Excellent pour l'embarqué, pas pour une API multi-utilisateur

## Impact
Toute la persistance passe par PostgreSQL. Migrations gérées avec golang-migrate.
```

## Exemples

### Interactif (le plus courant)

```bash
lore new
# → Pose Type, What, Why interactivement
# → Crée .lore/docs/feature-add-auth-2026-03-16.md
```

### One-liner (quand vous savez quoi écrire)

```bash
lore new decision "switch to PostgreSQL" "intégrité relationnelle pour les comptes"
# → Crée le document immédiatement, pas de prompts
```

### Rétroactif (documenter un commit passé)

```bash
git log --oneline -5
# abc1234 feat: add rate limiting
# def5678 fix: token refresh bug

lore new --commit abc1234
# → Pré-remplit "What" depuis le message de commit
# → Vous ajoutez juste "Why"
```

## Questions fréquentes

### "Quelle différence entre `lore new` et le hook automatique ?"

Le **hook** se déclenche automatiquement après chaque commit. `lore new` est pour documenter **délibérément** : un commit passé, une décision prise en réunion, ou une note sans commit associé.

### "Puis-je modifier un document après l'avoir créé ?"

Oui ! Les documents sont de simples fichiers Markdown dans `.lore/docs/`. Ouvrez-les dans n'importe quel éditeur. Ou utilisez `lore angela polish` pour une édition assistée par IA.

### "Que se passe-t-il si je fournis un mauvais hash de commit ?"

```bash
lore new --commit inexistant
# → Erreur : commit non trouvé
# Lore valide le hash avant de continuer.
```

## Tips & Tricks

- **One-liners pour scripts :** `lore new feature "add auth" "stateless scales"` — pas de prompts.
- **Après les réunions :** `lore new decision` pour capturer les décisions tant que le contexte est frais.
- **Batch rétroactif :** `git log --oneline -10` puis `lore new --commit <hash>` pour chacun.
- **Pré-définir le type :** `--type refactor` saute le sélecteur de type.
- **Mode express :** Répondez vite (< 3 secondes) et Lore saute les questions bonus.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Document créé avec succès |
| `1` | Erreur (commit non trouvé, pas dans un repo git) |
| `3` | Arguments invalides |

## Voir aussi

- [lore pending](pending.md) — Documents différés (Ctrl+C, non-TTY)
- [lore show](show.md) — Consulter les documents créés
- [Types de documents](../guides/document-types.md) — Référence complète des types
- [Quickstart](../getting-started/quickstart.md) — Guide pas à pas
