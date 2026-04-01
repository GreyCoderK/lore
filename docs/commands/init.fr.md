# lore init

Initialise un dépôt de documentation Lore dans le projet Git courant.

## Synopsis

```
lore init [flags]
```

## Description

Crée la structure du répertoire `.lore/`, génère les fichiers de configuration, installe le hook Git post-commit et produit un `.lore/README.md` comme pont de découverte pour les collaborateurs. Le répertoire courant doit se trouver dans un dépôt Git.

Si `.lore/` existe déjà, la commande se termine silencieusement (idempotent).

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--no-demo` | bool | `false` | Passer l'invitation à la démo après l'initialisation |

## Global Flags

| Flag | Type | Description |
|------|------|-------------|
| `--language` | string | Forcer la langue d'affichage (`en`, `fr`) |
| `--quiet` | bool | Supprimer les messages non essentiels |
| `--verbose` | bool | Afficher les détails de l'exécution |
| `--no-color` | bool | Désactiver la couleur dans la sortie |

## Ce qui est créé

```
.lore/
├── docs/             # Corpus de documentation (vide)
├── pending/          # Commits différés (vide)
├── templates/        # Modèles personnalisés (optionnel)
├── store.db          # Index LKS (généré automatiquement au premier usage)
└── README.md         # Pont de découverte — présente Lore aux collaborateurs
```

Également :
- `.lorerc` — Modèle de configuration partagé (commité dans Git)
- `.lorerc.local` — Surcharges personnelles avec espace réservé pour la clé API (chmod 600, gitignored)
- `.gitignore` — Mis à jour pour inclure `.lorerc.local` s'il n'y figure pas déjà
- `.git/hooks/post-commit` — Hook installé (ou à l'emplacement de `core.hooksPath`)

## Details de comportement

1. **Pas un dépôt Git** → Erreur, code de sortie 1. Lore nécessite Git.
2. **Déjà initialisé** → Sortie silencieuse, code 0 (peut être relancé sans risque).
3. **`core.hooksPath` configuré** → Avertissement : impossible d'installer le hook automatiquement. Suggère une installation manuelle avec les marqueurs `# LORE-START` / `# LORE-END`.
4. **`lore` absent du PATH** → Avertissement avec instructions pour l'ajouter.
5. **Après l'initialisation** → Propose une démo interactive (sauf si `--no-demo`).

## Exemples

```bash
# Initialisation standard
cd my-project
lore init
# → Crée .lore/, installe le hook, propose la démo

# Initialisation silencieuse (CI/scripts)
lore init --no-demo --quiet

# Fonctionne aussi depuis un sous-répertoire
cd my-project/src
lore init
# → Trouve .git/ dans le parent, initialise à cet endroit
```

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Succès (ou déjà initialisé) |
| `1` | Pas un dépôt Git |

## Voir aussi

- [lore hook](hook.fr.md) — Gérer le hook séparément
- [lore demo](demo.fr.md) — Essayer Lore sans initialiser
- [lore doctor](doctor.fr.md) — Diagnostiquer les problèmes après l'initialisation
