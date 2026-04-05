# lore upgrade

Met lore à jour vers la dernière version — ou une version spécifique.

## Synopsis

```
lore upgrade [flags]
```

## Qu'est-ce que ça fait ?

Pensez à `lore upgrade` comme mettre à jour une app sur votre téléphone, mais pour votre CLI. Au lieu d'aller sur un site web, télécharger un nouveau binaire et remplacer l'ancien manuellement, cette commande gère tout : vérifier ce qui est nouveau, télécharger, vérifier l'intégrité, et remplacer le binaire.

**En termes simples :** Vous lancez `lore upgrade`, et 5 secondes plus tard vous êtes sur la dernière version.

## Scénario concret

> La v1.2.0 vient de sortir avec des améliorations Angela. Vous voulez mettre à jour sans quitter votre terminal :
>
> ```bash
> lore upgrade
> # v1.0.0 → v1.2.0 — téléchargé, vérifié, installé. Terminé.
> ```

## Comment ça marche

1. **Détecte comment lore a été installé** — Homebrew ? `go install` ? Binaire direct ?
2. **Vérifie GitHub Releases** pour la version la plus récente
3. **Télécharge** l'archive correcte pour votre OS et architecture
4. **Vérifie le checksum SHA256** pour l'intégrité
5. **Remplace** le binaire actuel par le nouveau

> Si lore détecte une installation via **Homebrew** ou **go install**, il ne fait pas de self-update — il vous dit la bonne commande (`brew upgrade lore` ou `go install ...`).

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--version` | string | *(dernière)* | Cibler une version spécifique (ex : `v1.2.0` ou `v1.3.0-beta.1`) |

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

### Downgrade

```bash
lore upgrade --version v1.0.0
# → Oui, ça marche dans les deux sens
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

## Questions fréquentes

### "Puis-je downgrader ?"

Oui. `lore upgrade --version v1.0.0` cible n'importe quelle version publiée.

### "C'est sûr ? Si le téléchargement échoue ?"

L'upgrade est atomique : l'ancien binaire n'est remplacé qu'après que le nouveau est entièrement téléchargé et son checksum vérifié. En cas de problème, l'ancien reste en place.

### "Ça appelle automatiquement ?"

Non. `lore upgrade` ne s'exécute que quand **vous** l'appelez explicitement. Zéro appel réseau implicite. Utilisez `lore check-update` pour vérifier sans installer.

### "Et les permissions ?"

Si lore est installé dans un dossier système (ex : `/usr/local/bin`), vous pourriez avoir besoin de `sudo lore upgrade`.

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
