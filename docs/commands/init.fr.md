# lore init

Initialiser un dépôt de documentation Lore dans votre projet.

## Synopsis

```
lore init [flags]
```

## Qu'est-ce que ça fait ?

Pensez à `lore init` comme configurer un journal pour votre projet. De même qu'un journal a besoin d'un cahier avant de pouvoir écrire dedans, Lore a besoin d'un dossier `.lore/` avant de pouvoir capturer vos décisions.

**En termes simples :** Cette commande prépare votre projet à documenter le "pourquoi" derrière vos changements de code.

## Prérequis

- Vous devez être dans un **dépôt Git** (un dossier où vous avez lancé `git init`)
- C'est tout ! Pas de compte, pas de clé API, pas de connexion internet

## Ce qui se passe quand vous lancez

```bash
cd mon-projet
lore init
```

Lore fait 5 choses :

1. **Crée le dossier `.lore/`** — C'est là que vit toute votre documentation

```
.lore/
├── docs/          # Vos fichiers de documentation
├── pending/       # Commits en attente de documentation
├── templates/     # Templates personnalisés (optionnel, avancé)
├── store.db       # Index intelligent (auto-géré, ignorez-le)
└── README.md      # Explique Lore à quiconque clone votre repo
```

2. **Crée `.lorerc`** — Fichier de config partagé (committé dans git, visible par l'équipe)
3. **Crée `.lorerc.local`** — Fichier de config personnel (gitignore, pour les clés API)
4. **Installe le hook Git** — Un petit script qui déclenche Lore après chaque commit
5. **Propose une démo** — Vous montre comment Lore fonctionne en ~45 secondes

> **Analogie :** Pensez au hook Git comme un post-it sur votre écran qui dit "Pourquoi tu viens de faire ça ?" après chaque commit. C'est automatique — pas besoin de se rappeler de documenter.

![lore init](../assets/vhs/init.gif)
<!-- Generate: vhs assets/vhs/init.tape -->

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--no-demo` | bool | `false` | Sauter le prompt de démo après l'initialisation |
| `--language` | string | `en` | Définir la langue de l'interface (`en` ou `fr`) |
| `--quiet` | bool | `false` | Pas de sortie sauf erreurs |
| `--verbose` | bool | `false` | Afficher les détails |
| `--no-color` | bool | `false` | Désactiver la sortie colorée |

## Exemples

### Configuration de base (le plus courant)

```bash
cd mon-projet
lore init
# → ✓ Dossier .lore/ créé
# → ✓ Hook post-commit installé
# → ✓ .lore/README.md généré
# → Voulez-vous voir une démo ? [O/n]
```

### Configuration silencieuse (CI/Scripts)

```bash
lore init --no-demo --quiet
# → Pas de sortie, tout est configuré
```

### Déjà initialisé ?

```bash
lore init
# → Déjà initialisé (ne fait rien, pas d'erreur)
# Sûr de relancer plusieurs fois !
```

## Questions fréquentes

### "C'est quoi un hook Git ?"

Un hook Git est un script qui s'exécute automatiquement à certains moments de votre workflow Git. Lore utilise un **hook post-commit** — il s'exécute juste après `git commit`. Pas besoin de comprendre les hooks pour utiliser Lore ; il gère tout pour vous.

### "Ça va casser mes hooks existants ?"

Non. Lore utilise des marqueurs spéciaux (`# LORE-START` / `# LORE-END`) dans le fichier hook. Si vous avez déjà des hooks (comme Husky ou pre-commit), Lore ajoute sa section sans toucher les vôtres.

### "Et si je ne suis pas dans un dépôt Git ?"

```bash
lore init
# → Erreur : pas un dépôt Git
# Correction : lancez "git init" d'abord, puis "lore init"
```

### "Puis-je annuler ?"

Oui. Supprimez le dossier `.lore/` : `rm -rf .lore` — votre code et historique Git sont complètement intacts.

## Que se passe-t-il ensuite ?

Après `lore init`, la prochaine fois que vous lancez `git commit`, Lore posera automatiquement 3 questions :

1. **Type** — Quel genre de changement ? (feature, bugfix, decision, refactor, note)
2. **Quoi** — Pré-rempli depuis votre message de commit. Appuyez sur Entrée.
3. **Pourquoi** — La question importante ! Pourquoi ce choix ?

C'est tout — 90 secondes, et le "pourquoi" est capturé pour toujours.

## Tips & Tricks

- **Sûr de relancer :** `lore init` est idempotent — le lancer deux fois ne fait rien.
- **Après un clone :** Les membres de l'équipe devraient lancer `lore init` après clonage pour installer leur hook local.
- **Setup CI :** `lore init --no-demo --quiet` dans les pipelines.
- **Monorepo :** Lancez à la racine du repo. Les documents capturent les chemins complets.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Succès (ou déjà initialisé) |
| `1` | Pas un dépôt Git |

## Voir aussi

- [lore demo](demo.md) — Essayer Lore sans initialiser (aperçu sûr)
- [lore hook](hook.md) — Gérer le hook séparément
- [Quickstart](../getting-started/quickstart.md) — Guide complet 5 minutes
- [Configuration](../guides/configuration.md) — Personnaliser les paramètres
