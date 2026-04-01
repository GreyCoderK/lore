# lore config

Gérer les identifiants API et consulter la configuration.

## Synopsis

```
lore config <set-key|delete-key|list-keys>
```

## Sous-commandes

| Sous-commande | Description |
|---------------|-------------|
| `set-key <provider>` | Stocker une clé API de manière sécurisée |
| `delete-key <provider>` | Supprimer une clé API enregistrée |
| `list-keys` | Afficher l'état de tous les fournisseurs |

**Fournisseurs reconnus :** `anthropic`, `openai`, `ollama`

## Description

Gère les identifiants API pour les fonctionnalités Angela (IA). Les clés sont stockées dans le gestionnaire de credentials du système (Trousseau macOS, secret-service Linux, Gestionnaire d'identifiants Windows) ou dans `.lorerc.local` en solution de repli.

## `lore config set-key`

```bash
lore config set-key anthropic
# → Enter API key: [hidden input]
# → ✓ Key stored for anthropic
```

Lit la clé depuis stdin sans écho (saisie sécurisée).

## `lore config delete-key`

```bash
lore config delete-key anthropic
# → ✓ Key removed for anthropic
```

## `lore config list-keys`

```bash
lore config list-keys
# anthropic     stored
# openai        not set
# ollama        stored
```

## Tips & Tricks

- Stockez vos clés via `lore config set-key` plutôt qu'en éditant `.lorerc.local` manuellement — il utilise le trousseau du système quand c'est possible.
- En CI, utilisez les variables d'environnement : `LORE_AI_API_KEY=sk-...` (pas besoin de trousseau).
- Ollama n'a pas besoin de clé API (modèles locaux), mais vous pouvez configurer l'endpoint dans `.lorerc`.

## Voir aussi

- [Guide de configuration](../guides/configuration.md) — Référence complète de la configuration
- [lore angela draft](angela-draft.fr.md) — Utilise le fournisseur configuré
- [lore doctor --config](doctor.fr.md) — Valider la configuration
