# lore config

Gérer les identifiants API et consulter la configuration.

## Synopsis

```
lore config <set-key|delete-key|list-keys>
```

## Qu'est-ce que ça fait ?

`lore config` gère les clés API qui alimentent les fonctions IA d'Angela. Pensez-y comme un gestionnaire de mots de passe spécifiquement pour vos fournisseurs IA — il stocke les clés de façon sécurisée dans le trousseau de votre OS pour qu'elles ne finissent jamais dans un fichier qu'on pourrait committer par accident.

> **Analogie :** C'est comme la page de paramètres d'une app. On ne l'utilise pas tous les jours, mais quand on doit connecter un nouveau service, c'est là qu'on va.

## Scénario concret

> Vous venez de recevoir votre clé API Anthropic. Il est temps de débloquer Angela :
>
> ```bash
> lore config set-key anthropic
> # Enter API key: [masqué]
> # ✓ Clé stockée de façon sécurisée
> ```
>
> Maintenant `lore angela polish` fonctionne. Votre clé est dans le trousseau OS — jamais dans un fichier en clair.

![lore config](../assets/vhs/config.gif)
<!-- Generate: vhs assets/vhs/config.tape -->

## Sous-commandes

| Sous-commande | Description |
|---------------|-------------|
| `set-key <fournisseur>` | Stocker une clé API de façon sécurisée |
| `delete-key <fournisseur>` | Supprimer une clé stockée |
| `list-keys` | Afficher le statut de tous les fournisseurs |

**Fournisseurs connus :** `anthropic`, `openai`, `ollama`

## Flags

Cette commande n'a pas de flags. Le fournisseur est spécifié en argument.

## Comment le stockage des clés fonctionne

Lore essaie l'option la plus sécurisée d'abord, puis utilise un fallback :

```mermaid
graph TD
    A[lore config set-key] --> B{Trousseau OS disponible ?}
    B -->|Oui, macOS| C[Trousseau d'accès]
    B -->|Oui, Linux| D[secret-service / libsecret]
    B -->|Oui, Windows| E[Gestionnaire d'identifiants]
    B -->|Non| F[Fallback .lorerc.local]
    F --> G[Mode fichier 0600 — lecture propriétaire uniquement]
```

## Exemples

### Configurer Anthropic (Claude)

```bash
lore config set-key anthropic
# → Enter API key: [masqué — pas d'écho]
# → ✓ Clé stockée pour anthropic

# Vérifier
lore config list-keys
# anthropic     stored
# openai        not set
# ollama        stored
```

### Configurer Ollama (Local — Pas de clé nécessaire)

```yaml
# .lorerc (pas de clé API nécessaire !)
ai:
  provider: "ollama"
  model: "llama3"
  endpoint: "http://localhost:11434"
```

### Supprimer une clé

```bash
lore config delete-key anthropic
# → ✓ Clé supprimée pour anthropic
```

### CI/CD (Pas de trousseau)

En CI, utilisez les variables d'environnement :

```bash
export LORE_AI_API_KEY="sk-ant-..."
export LORE_AI_PROVIDER="anthropic"
```

## Questions fréquentes

### "Où exactement est stockée ma clé ?"

Lancez `lore config list-keys`. Si ça dit "stored", la clé est dans votre trousseau OS. Si vous utilisez le fallback, elle est dans `.lorerc.local` (gitignore et chmod 600).

Backend keychain par plateforme :

| Plateforme | Backend | Outil utilisé |
|------------|---------|---------------|
| **macOS** | Trousseau système | `security add-generic-password` / `find-generic-password` |
| **Linux** | GNOME Keyring / KWallet | `secret-tool store` / `secret-tool lookup` |
| **Windows** | Credential Manager | Fallback sur `.lorerc.local` (keychain natif prévu) |

### "J'ai mis la clé mais Angela dit 'pas de fournisseur configuré'"

Deux choses sont nécessaires :
1. La **clé** (via `lore config set-key`)
2. Le **nom du fournisseur** dans `.lorerc` :

```yaml
ai:
  provider: "anthropic"   # Dit à Angela QUEL fournisseur utiliser
  model: "claude-sonnet-4-20250514"
```

### "Puis-je avoir des clés différentes par projet ?"

Oui. `.lorerc.local` est par projet (il vit dans la racine de votre projet, pas globalement).

### "C'est sécurisé ?"

- Trousseau OS : même sécurité que vos mots de passe sauvegardés
- Fallback `.lorerc.local` : mode fichier `0600` (vous seul pouvez lire)
- `.lorerc.local` est dans `.gitignore` — jamais committé
- Les clés sont nettoyées des messages d'erreur

## Tips & Tricks

- **Toujours utiliser `lore config set-key`** plutôt que d'éditer `.lorerc.local` manuellement — le trousseau est plus sécurisé.
- **CI/CD :** Utilisez `LORE_AI_API_KEY` en variable d'env.
- **Ollama = gratuit :** Pas de clé API, pas de coût. Idéal pour expérimenter.
- **Rotation des clés :** `delete-key` puis `set-key` pour remplacer une clé expirée.
- **Valider après setup :** Lancez `lore angela draft` pour confirmer que le fournisseur fonctionne.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Succès |
| `1` | Erreur (fournisseur invalide, trousseau indisponible) |
| `3` | Arguments invalides (nom de fournisseur inconnu) |

## Voir aussi

- [Guide configuration](../guides/configuration.md) — Référence complète avec exemples `.lorerc`
- [lore angela draft](angela-draft.md) — Tester votre setup (zéro-API, pas de clé)
- [lore angela polish](angela-polish.md) — Utilise la clé configurée
- [lore doctor --config](doctor.md) — Valider votre configuration
