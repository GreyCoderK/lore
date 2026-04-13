# lore status

Tableau de bord de santé documentaire de votre dépôt.

## Synopsis

```
lore status [flags]
```

## Qu'est-ce que ça fait ?

`lore status` donne un aperçu de la santé documentaire de votre projet : combien de commits sont documentés, ce qui est en attente et ce qui nécessite une attention.

> **Analogie :** `git status` montre la santé de votre *code*. `lore status` montre la santé de votre *savoir*.

## Scénario concret

> Lundi matin. Première chose : vérifier la santé de la documentation.
>
> ```bash
> lore status
> ```
>
> 12 documentés, 2 en attente, santé bonne. Vous savez où vous en êtes.

![lore status](../assets/vhs/status-badge.gif)
<!-- Generate: vhs assets/vhs/status-badge.tape -->

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--badge` | bool | `false` | Générer un badge shields.io avec le % de couverture |
| `--quiet` | bool | `false` | Sortie machine : paires `clé=valeur` |

## Sortie Dashboard

```
Project     mon-projet
Hook        installed
Docs        12 documented, 2 pending
Express     25% express (3 docs), 75% complete
Angela      draft [anthropic], 2 docs need review
Review      3 findings, 2 days ago
Health      ✓ all good
```

| Ligne | Signification |
|-------|---------------|
| **Project** | Nom du projet (depuis `.lorerc` ou le nom du dossier) |
| **Hook** | Hook post-commit installé ? Si non, les commits ne déclenchent pas lore |
| **Docs** | Commits documentés vs en attente dans la queue |
| **Express** | Part des docs créés en mode express (rapide) vs mode complet (5 questions) |
| **Angela** | Mode IA et fournisseur. "Need review" = docs pas encore analysés par Angela |
| **Review** | Résultats de `lore angela review`. "No issues" = corpus propre |
| **Health** | État global. `✓` = bon. `✗` = lancez `lore doctor` |

## Mode Badge

```bash
lore status --badge
# → [![lore](https://img.shields.io/badge/lore-documented%2085%25-d4a)](...)
```

| Couverture | Couleur |
|------------|---------|
| < 50% | Gris |
| 50–79% | Vert |
| 80%+ | Or |
| 100% | Or + étoile |

> **Calcul de la couverture :** Commits documentés ÷ total des commits. Les merges, rebases et docs de démo sont exclus. `[doc-skip]` compte comme couvert (ignore intentionnel).

> **Avertissement :** Si plus de 70% de vos commits "documentés" sont en réalité des `[doc-skip]`, le badge affiche un avertissement. Ignorer est acceptable pour les commits triviaux, mais en excès cela signifie que vous évitez la documentation.

### Localisation du badge

| Langue | Texte du badge |
|--------|----------------|
| EN (`language: "en"`) | `lore \| documented 85%` |
| FR (`language: "fr"`) | `lore \| documenté 85%` |

## Mode Quiet (`--quiet`)

Sortie machine pour CI/CD :

```bash
lore status --quiet
# hook=installed
# docs=12
# pending=2
# health=ok
# angela=draft
# review_findings=3
```

## Flux

```mermaid
graph LR
    A[lore status] --> B{Mode ?}
    B -->|Défaut| C[Collecter toutes les métriques]
    C --> D[Afficher dashboard]
    B -->|--badge| E[Calculer couverture %]
    E --> F[Générer URL shields.io]
    F --> G[Sortie : badge Markdown]
    B -->|--quiet| H[Sortie : paires clé=valeur]
```

### Exemple CI

```bash
# Échouer le build si des docs sont en attente
pending=$(lore status --quiet | grep "pending=" | cut -d= -f2)
if [ "$pending" -gt 0 ]; then
  echo "⚠ $pending commits ont besoin de documentation"
  exit 1
fi
```

## Exemples

```bash
# Vérification quotidienne
lore status

# Badge pour le README
lore status --badge

# Gate CI
health=$(lore status --quiet | grep "health=" | cut -d= -f2)
[ "$health" = "ok" ] || exit 1
```

## Questions fréquentes

### "Que signifie 'Express 25%' ?"

25% des docs créés en mode express (rapide), 75% en mode complet. Ni l'un ni l'autre n'est meilleur.

### "Le badge est gris ?"

Couverture = documentés / total. Pour améliorer : `lore pending` et `lore new --commit`. Vert à 50%, or à 80%.

### "Health montre ✗"

Lancez `lore doctor` puis `lore doctor --fix`.

## Tips & Tricks

- **Ajoutez le badge au README :** `lore status --badge >> README.md` (puis déplacez-le dans la section badges).
- **Vérification quotidienne :** Lancez `lore status` en début de journée pour voir si quelque chose nécessite une attention.
- **Gate CI :** `--quiet` pour parser les valeurs et échouer le build si la documentation est en retard.
- **Après `lore doctor --fix` :** Lancez `lore status` pour confirmer que la santé est `✓ all good`.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Succès |
| `1` | Erreur (`.lore/` non trouvé) |

## Voir aussi

- [lore doctor](doctor.md) — Corriger les problèmes
- [lore list](list.md) — Voir tous les documents
- [lore pending](pending.md) — Résoudre les commits en attente
