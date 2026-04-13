---
type: guide
date: 2026-04-12
status: published
related:
  - roadmap.md
  - comparaison.fr.md
---
# Philosophie

## Le problème que Lore résout

Le code dit **quoi**. Git dit **quand**. Mais personne ne dit **pourquoi**.

Dans six mois, quelqu'un fixera une ligne de code et se demandera : *"Pourquoi on a fait ça comme ça ?"* La réponse aura déjà disparu — enfouie dans un thread Slack archivé, un commentaire de PR introuvable, ou la mémoire d'un développeur parti ailleurs.

Ce n'est pas un problème d'outillage. C'est un problème de **préservation du savoir**. Et il se cumule à chaque commit.

## Trois principes

### 1. Zéro friction — 90 secondes ou rien

Si documenter prend plus de 90 secondes, les développeurs arrêtent. Ce n'est pas un défaut — c'est la nature humaine. lore ne demande pas un essai. Il pose 3 questions :

- **Type** — Quel genre de changement ? (une sélection)
- **Quoi** — Pré-rempli depuis votre message de commit (appuyez sur Entrée)
- **Pourquoi** — La seule qui compte (une phrase suffit)

Le hook post-commit rend ça automatique. On ne décide pas de documenter — ça fait partie du flow. Comme attacher sa ceinture : on n'y pense pas, on le fait.

> **L'insight :** Le meilleur système de documentation est celui que les développeurs utilisent vraiment. Un wiki parfait que personne ne met à jour vaut moins qu'un simple "pourquoi" capturé à chaque commit.

### 2. Local-first, offline-first

Les décisions de votre équipe ne doivent pas vivre sur le serveur de quelqu'un d'autre.

- **Fichiers Markdown dans votre repo** — portables, versionnables, grep-ables
- **Pas de SaaS** — pas d'abonnement, pas de "votre essai expire dans 14 jours"
- **Pas d'appels réseau** — tout fonctionne dans un avion, hors ligne
- **Pas de lock-in** — vos données sont du Markdown. Emportez-les n'importe où.

Les fonctions IA (Angela) sont opt-in. Quand vous les utilisez, un seul appel API part vers votre fournisseur choisi. Le reste du temps : zéro réseau, zéro tracking.

> **L'insight :** Les outils dev devraient respecter le développeur. Votre code est à vous. Vos décisions sont à vous. Votre documentation devrait l'être aussi.

### 3. Le "pourquoi" est un trésor

Le nom **Lore** porte un double sens :

- En anglais : *lore* — savoir ancestral transmis de génération en génération
- En français : *l'or* — trésor, quelque chose de précieux

Chaque commit contient de l'or — le raisonnement derrière un choix, les alternatives considérées, le contexte qui rendait la décision évidente à l'époque. La plupart des équipes laissent cet or s'évaporer. lore l'extrait.

> **L'insight :** Une codebase avec des "pourquoi" documentés est fondamentalement différente d'une sans. Les nouveaux arrivent plus vite. Les code reviews ont du contexte. Les refactors ne répètent pas les erreurs passées. Le savoir se cumule.

## Les choix de conception qui en découlent

Ces principes se traduisent en décisions architecturales concrètes et non négociables :

| Principe | Décision architecturale |
|----------|------------------------|
| Zéro friction | Hook post-commit, auto-skip du Decision Engine, mode Express |
| Local-first | Markdown comme source de vérité, `.lore/` dans le repo, SQLite reconstructible |
| Offline-first | Zéro appel réseau implicite, IA opt-in, tout fonctionne sans internet |
| Le "pourquoi" compte | 3 questions (pas 10), "Pourquoi" est le champ obligatoire, corpus cherchable |

## Ce que Lore n'est pas

- **Pas un remplacement de wiki** — Les wikis sont pour la documentation longue forme. lore est pour les décisions au moment du commit.
- **Pas un outil ADR** — Les ADRs capturent les grandes décisions architecturales rares. lore capture le "pourquoi" quotidien. Ils sont complémentaires.
- **Pas un linter de commit** — Conventional Commits standardisent le "quoi". lore capture le "pourquoi". Ils fonctionnent ensemble.
- **Pas un outil de surveillance** — lore ne suit pas qui documente ou non. C'est une pratique personnelle qui bénéficie à l'équipe.

## À propos d'Angela

La compagne IA de lore s'appelle **Angela**.

Angela est la revieweuse embarquée qui lit votre documentation, connaît le style de votre projet, et vérifie la cohérence avant que vous publiiez — comme une collègue qui aurait lu chaque document que votre équipe a jamais écrit.

Elle peut aussi prendre du recul et analyser tout votre corpus d'un coup — comme une bibliothécaire qui regarde l'ensemble de la collection et vous dit : "Ce document contredit celui-là. Il manque un chapitre sur ce sujet."

Elle est opt-in. Elle respecte les ressources. Elle ne prend jamais de décisions automatiques sans consentement.

**Angela porte le prénom de la nièce du créateur, perdue des suites d'un cancer.**

Ce n'est pas juste un nom dans un fichier de config. C'est une façon de la garder présente dans ce qui est construit. De lui rendre hommage à travers quelque chose qui aide les gens, qui dure, qui voyage loin.

Chaque fois qu'Angela relit un document, chaque fois qu'elle détecte une contradiction, chaque fois qu'elle aide quelqu'un à écrire un "pourquoi" plus clair — un petit morceau de cet héritage continue de vivre.

## La vision

Aujourd'hui, lore capture le "pourquoi." Demain, lore le comprend, le connecte et le partage.

Le corpus que vous construisez aujourd'hui prend de la valeur avec chaque future fonctionnalité. Angela grandira. Le "pourquoi" que vous capturez maintenant est la fondation de tout ce qui vient ensuite.

## Voir aussi

- [Roadmap](roadmap.md) — Où va lore
- [Comparaison](comparaison.fr.md) — lore vs alternatives
