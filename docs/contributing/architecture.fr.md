# Architecture (pour contributeurs)

Vue simplifiée du code Lore. Pour les directives de contribution, voir `CONTRIBUTING.md` à la racine du projet.

## Structure du projet

```
cmd/           Commandes Cobra — un fichier par commande CLI
internal/
  domain/      Interfaces et types partagés (aucune dep interne)
  config/      Cascade de configuration (.lorerc → .lorerc.local → env)
  git/         Adaptateur Git — hooks, log, diff, info commit
  storage/     Stockage documents, front matter, index, doctor
  workflow/    Flux réactif (hook) et proactif (lore new)
  generator/   Pipeline de génération de documents
  angela/      Logique documentation assistée par IA
  ai/          Implementations fournisseurs IA (Anthropic, OpenAI, Ollama)
  i18n/        Catalogues de messages bilingues (EN/FR)
  ui/          Interface terminal — couleurs, progression, listes
  engagement/  Messages milestones, prompt star
  fileutil/    Opérations fichiers atomiques
  notify/      Notification IDE (détection non-TTY)
  status/      Collecteur de santé du dépôt
  template/    Moteur de templates Go
.lore/
  docs/        Corpus de documentation (Markdown + YAML front matter)
  pending/     Commits interrompus/différés
  store.db     Index LKS (SQLite — reconstructible depuis les docs)
```

## Flux de données

```
commit → hook post-commit → workflow/reactive.go
  → questions (ui/) → moteur templates → generator/
  → écriture atomique dans .lore/docs/ → mise à jour index (storage/)
```

## Patterns clés

- **Markdown = source de vérité** — index, cache, LKS sont tous reconstructibles
- **Écritures atomiques** — `.tmp` + `os.Rename()` évite la corruption
- **IOStreams** — `stderr` pour les humains, `stdout` pour les machines
- **Zéro réseau implicite** — l'IA est opt-in, tout fonctionne hors ligne

## Comment contribuer

1. Fork depuis `main`
2. Écrire des tests (`go test ./...`)
3. Exécuter `go vet ./...`
4. Ouvrir une PR
