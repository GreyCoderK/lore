---
type: reference
date: 2026-04-12
status: published
related:
  - init.md
  - new.md
  - show.md
  - list.md
  - status.md
  - delete.md
  - pending.md
  - doctor.md
  - hook.md
  - config.md
  - release.md
  - demo.md
  - decision.md
  - angela-draft.md
  - angela-polish.md
  - angela-review.md
  - check-update.md
  - upgrade.md
  - completion.md
  - ../guides/document-types.md
  - ../guides/contextual-detection.md
  - ../guides/configuration.md
angela_mode: polish
---
# Vue d'ensemble des commandes

Toutes les commandes lore CLI en un coup d'œil.

## Commandes principales

| Commande | Description |
|----------|-------------|
| [`lore init`](init.md) | Initialiser lore dans le dépôt courant |
| [`lore new`](new.md) | Créer une documentation à la demande |
| [`lore show`](show.md) | Rechercher et afficher des documents |
| [`lore list`](list.md) | Lister tous les documents du corpus |
| [`lore status`](status.md) | Tableau de bord de santé |
| [`lore delete`](delete.md) | Supprimer un document |
| [`lore pending`](pending.md) | Gérer les commits non documentés |

## Maintenance

| Commande | Description |
|----------|-------------|
| [`lore doctor`](doctor.md) | Diagnostiquer et réparer le corpus |
| [`lore hook`](hook.md) | Gérer le hook Git post-commit |
| [`lore config`](config.md) | Voir et gérer la configuration |
| [`lore release`](release.md) | Générer des notes de version |
| [`lore demo`](demo.md) | Démonstration interactive |
| [`lore decision`](decision.md) | Statut du moteur de décision |

## Angela (IA + Hors-ligne)

| Commande | Description |
|----------|-------------|
| [`lore angela draft`](angela-draft.md) | Analyse structurelle zéro-API (hors ligne) |
| [`lore angela polish`](angela-polish.md) | Réécriture assistée par IA + enrichissement synthesizer hors ligne |
| [`lore angela review`](angela-review.md) | Analyse de cohérence du corpus (IA) |
| [`lore angela consult`](angela-consult.md) | Consultation ponctuelle d'un seul persona (hors ligne) |

## Mises à jour

| Commande | Description |
|----------|-------------|
| [`lore check-update`](check-update.md) | Vérifier si une mise à jour est disponible |
| [`lore upgrade`](upgrade.md) | Mettre lore à jour vers la dernière version |

## Utilitaires

| Commande | Description |
|----------|-------------|
| [`lore completion`](completion.md) | Générer des scripts de complétion shell |

## Guides connexes

- [Types de Documents & Métadonnées](../guides/document-types.md) — Référence des types, statuts et front matter
- [Détection Contextuelle](../guides/contextual-detection.md) — Comment le hook décide quoi faire
- [Configuration](../guides/configuration.md) — Référence complète de la configuration
