---
type: reference
date: 2026-04-12
status: published
related:
  - upgrade.md
  - ../getting-started/installation.md
angela_mode: polish
---
# lore check-update

Vérifier si une nouvelle version de Lore est disponible.

## Synopsis

```
lore check-update
```

## Qu'est-ce que ça fait ?

Le pendant lecture seule de `lore upgrade`. Vérifie GitHub pour les nouvelles releases et montre ce qui est disponible — sans rien télécharger ni installer.

**En termes simples :** "Suis-je en retard ? Quelles versions sont sorties ?"

> Lore ne vérifie jamais les mises à jour automatiquement. Cette commande est le seul moyen de le savoir — 100% opt-in, zéro appel réseau implicite.

## Scénario concret

> Avant de commencer un gros refactor, vous voulez vérifier que vous êtes à jour :
>
> ```bash
> lore check-update
> # Current : v1.0.0 — Latest : v1.2.0
> # Lancez : lore upgrade
> ```

## Flags

Cette commande ne prend pas de flags ni d'arguments. Elle vérifie la dernière GitHub Release.

## Comment ça marche

1. Récupère les releases récentes depuis GitHub (y compris les pre-releases)
2. Compare votre version actuelle avec ce qui est disponible
3. Liste toutes les versions plus récentes, avec des labels `(pre-release)` si applicable

## Exemples

### Nouvelles versions disponibles

```bash
lore check-update
# → Recherche de mises à jour...
# → Mise à jour disponible : v1.0.0 → v1.3.0-beta.1
# →
# →   v1.3.0-beta.1        (pre-release)
# →   v1.2.1
# →   v1.2.0
# →   v1.1.0
# →
# → Lancez : lore upgrade
```

### Déjà à jour

```bash
lore check-update
# → Recherche de mises à jour...
# → À jour (v1.3.0).
```

### Build de développement

```bash
lore check-update
# → Recherche de mises à jour...
# → Mise à jour disponible : dev → v1.2.0
# →
# →   v1.2.0
# →   v1.1.0
# →   v1.0.0
# →
# → Lancez : lore upgrade
```

> Sur les builds de développement, `check-update` fonctionne quand même — `dev` est toujours traité comme plus ancien que toute release publiée, donc toutes les releases sont affichées.

## Questions fréquentes

### "Ça fait des appels réseau ?"

Oui — une requête GET vers l'API GitHub Releases. Uniquement quand **vous** lancez la commande. Lore ne vérifie jamais les mises à jour en arrière-plan ni lors d'autres commandes.

### "Comment installer une version spécifique depuis la liste ?"

Utilisez le flag `--version` sur `lore upgrade` :

```bash
lore check-update                    # Voir ce qui est disponible
lore upgrade --version v1.2.0        # Installer une version spécifique
```

### "Pourquoi `lore status` ne montre pas ça ?"

Par conception. `lore status` fonctionne entièrement hors ligne et ne fait jamais d'appels réseau. Les vérifications de mise à jour sont explicitement opt-in via cette commande.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Succès (à jour ou mises à jour disponibles) |
| `1` | Erreur (réseau, aucune release trouvée) |

## Tips & Tricks

- **Avant les gros chantiers :** Vérifiez que vous êtes à jour pour avoir les dernières features et corrections.
- **En pair avec upgrade :** `check-update` pour voir ce qui est disponible, `upgrade` quand vous êtes prêt.
- **Pas de vérification automatique :** Lore ne phone jamais home. C'est le seul moyen de savoir si vous êtes en retard — 100% opt-in.
- **Dev builds :** Si vous avez compilé depuis les sources sans tags de version, `check-update` marche quand même — il compare avec les releases publiées.

## Voir aussi

- [lore upgrade](upgrade.md) — Télécharger et installer la nouvelle version
- [Installation](../getting-started/installation.md) — Méthodes d'installation initiale
