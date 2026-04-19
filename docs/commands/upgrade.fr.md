# lore upgrade

Met Lore à jour vers la dernière version — ou une version spécifique.

## Synopsis

```
lore upgrade [flags]
```

## Qu'est-ce que ça fait ?

`lore upgrade` gère la mise à jour en une seule commande : vérifier une nouvelle version, télécharger le bon binaire pour votre OS et architecture, vérifier son checksum, et remplacer le binaire actuel. Aucun téléchargement manuel ni réinstallation.

## Scénario concret

> La v1.2.0 vient de sortir avec des améliorations Angela. Vous voulez mettre à jour sans quitter votre terminal :
>
> ```bash
> lore upgrade
> # v1.0.0 → v1.2.0 — téléchargé, vérifié, installé. Terminé.
> ```

## Comment ça marche

1. **Détecte comment Lore a été installé** — Homebrew ? `go install` ? Binaire direct ?
2. **Vérifie GitHub Releases** pour la version la plus récente
3. **Télécharge** l'archive correcte pour votre OS et architecture
4. **Vérifie le checksum SHA256** pour l'intégrité
5. **Remplace** le binaire actuel par le nouveau

> Si Lore détecte une installation via **Homebrew** ou **go install**, il ne fait pas de self-update — il vous dit la bonne commande (`brew upgrade lore` ou `go install ...`).

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--version` | string | *(dernière)* | Cibler une version spécifique (ex : `v1.2.0` ou `v1.3.0-beta.1`) |

## Flags globaux

| Flag | Type | Description |
|------|------|-------------|
| `--language` | string | Langue d'affichage (`en`, `fr`) |
| `--quiet` | bool | Supprimer les sorties non essentielles |
| `--no-color` | bool | Désactiver la sortie colorée |

## Exemples

### Mettre à jour vers la dernière (le plus courant)

```bash
lore upgrade
# → Recherche de mises à jour...
# → Nouvelle version : v1.0.0 → v1.2.0
# → Téléchargement...
# → Vérification checksum...
# → Installation...
# → ✓ Mis à jour vers v1.2.0.
```

### Version spécifique

```bash
lore upgrade --version v1.1.0
```

### Version spécifique (downgrade)

```bash
lore upgrade --version v1.0.0
```

### Déjà à jour

```bash
lore upgrade
# → Déjà à jour (v1.2.0).
```

### Homebrew détecté

```bash
lore upgrade
# → Installé via Homebrew. Lancez :
# →   brew upgrade lore
```

### Build de développement

```bash
# Si compilé depuis les sources sans tags de version :
lore upgrade
# → Build de développement — la mise à jour n'est disponible que pour les binaires de release.
```

## Questions fréquentes

### "Puis-je revenir à une version antérieure ?"

Oui. Utilisez `--version` pour cibler n'importe quelle release publiée :

```bash
lore upgrade --version v1.0.0
```

### "C'est sûr ? Si le téléchargement échoue ?"

L'upgrade est atomique : l'ancien binaire n'est remplacé qu'après que le nouveau est entièrement téléchargé et son checksum vérifié. En cas de problème, l'ancien reste en place.

### "Est-ce que ça vérifie automatiquement ?"

Non. `lore upgrade` ne s'exécute que quand vous l'appelez explicitement — zéro appel réseau implicite. Utilisez `lore check-update` pour vérifier une nouvelle version sans installer.

### "Et les permissions ?"

Si Lore est installé dans un dossier système (ex : `/usr/local/bin`), vous pourriez avoir besoin de `sudo lore upgrade`.

## Tips & Tricks

- **Vérifier d'abord :** `lore check-update` avant de mettre à jour pour voir ce qui a changé.
- **Rollback :** `lore upgrade --version v1.0.0` en cas de problème avec la nouvelle version.
- **Homebrew :** Lore détecte Homebrew et dit d'utiliser `brew upgrade lore`.
- **Permissions :** Si dans `/usr/local/bin`, utilisez `sudo lore upgrade`.
- **Régénérer les complétions :** Après upgrade, relancez `eval "$(lore completion zsh)"` pour les nouvelles commandes.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Succès (ou déjà à jour) |
| `1` | Erreur (réseau, checksum, permissions) |

## Voir aussi

- [lore check-update](check-update.md) — Vérifier sans installer
- [Installation](../getting-started/installation.md) — Première installation
