# Architecture (pour contributeurs)

Vue simplifiee du code Lore. Pour les directives de contribution, voir `CONTRIBUTING.md` a la racine du projet.

## Structure du projet

```
cmd/           Commandes Cobra — un fichier par commande CLI
internal/
  domain/      Interfaces et types partages (aucune dep interne)
  config/      Cascade de configuration (.lorerc → .lorerc.local → env)
  git/         Adaptateur Git — hooks, log, diff, info commit
  storage/     Stockage documents, front matter, index, doctor
  workflow/    Flux reactif (hook) et proactif (lore new)
  generator/   Pipeline de generation de documents
  angela/      Logique documentation assistee par IA
  ai/          Implementations fournisseurs IA (Anthropic, OpenAI, Ollama)
  i18n/        Catalogues de messages bilingues (EN/FR)
  ui/          Interface terminal — couleurs, progression, listes
  engagement/  Messages milestones, prompt star
  fileutil/    Operations fichiers atomiques
  notify/      Notification IDE (detection non-TTY)
  status/      Collecteur de sante du dépôt
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
  → écriture atomique dans .lore/docs/ → mise a jour index (storage/)
```

## Patterns cles

- **Markdown = source de verite** — index, cache, LKS sont tous reconstructibles
- **Ecritures atomiques** — `.tmp` + `os.Rename()` evite la corruption
- **IOStreams** — `stderr` pour les humains, `stdout` pour les machines
- **Zero réseau implicite** — l'IA est opt-in, tout fonctionne hors ligne

## Comment contribuer

1. Fork depuis `main`
2. Ecrire des tests (`go test ./...`)
3. Executer `go vet ./...`
4. Ouvrir une PR
