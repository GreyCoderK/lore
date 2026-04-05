# lore check-update

Vérifier si une nouvelle version de lore est disponible.

## Synopsis

```
lore check-update
```

## Qu'est-ce que ça fait ?

C'est le pendant lecture seule de `lore upgrade`. Vérifie GitHub pour les nouvelles releases et montre ce qui est disponible — sans rien télécharger ni installer. Comme vérifier le menu avant de commander.

**En termes simples :** "Suis-je en retard ? Quelles versions sont sorties ?"

> Lore ne vérifie jamais les mises à jour automatiquement. Cette commande est le seul moyen — 100% opt-in, zéro appel réseau implicite.

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

1. Lit votre version courante (`lore --version`)
2. Récupère les releases récentes depuis GitHub
3. Compare les numéros de version
4. Liste toutes les versions plus récentes

**Aucune donnée envoyée.** Lore ne lit que les informations publiques de release depuis l'API GitHub.

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
# → À jour (v1.3.0).
```

### Build de développement

```bash
lore check-update
# → Mise à jour disponible : dev → v1.2.0
```

## Questions fréquentes

### "Ça fait des appels réseau ?"

Oui — une requête GET vers l'API GitHub Releases. Uniquement quand **vous** lancez la commande. Jamais en arrière-plan.

### "Comment installer une version depuis la liste ?"

```bash
lore check-update                      # Voir ce qui est disponible
lore upgrade --version v1.2.0          # Installer une version spécifique
```

### "Pourquoi `lore status` ne montre pas ça ?"

Par design. `lore status` fonctionne hors ligne — il ne fait jamais d'appels réseau. Les vérifications de mise à jour sont explicitement opt-in via cette commande.

## Tips & Tricks

- **Avant les gros chantiers :** Vérifiez que vous êtes à jour pour avoir les dernières features et corrections.
- **Pair avec upgrade :** `check-update` pour voir, `upgrade` quand vous êtes prêt.
- **Pas de vérification automatique :** Lore ne phone jamais home. C'est le seul moyen de savoir.
- **Dev builds :** Si vous avez compilé depuis les sources sans tags de version, `check-update` marche quand même.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Succès (à jour ou mises à jour disponibles) |
| `1` | Erreur (réseau, aucune release trouvée) |

## Voir aussi

- [lore upgrade](upgrade.md) — Télécharger et installer la nouvelle version
- [Installation](../getting-started/installation.md) — Méthodes d'installation initiale
