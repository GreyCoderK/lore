# FAQ

## General

### Qu'est-ce que Lore ?

Un outil CLI qui capture le *pourquoi* derrière vos changements de code au moment du commit. Trois questions, quatre-vingt-dix secondes, un document Markdown pour toujours.

### Lore nécessite-t-il une connexion internet ?

Non. Tout fonctionne hors ligne par défaut. Les fonctions IA (Angela) sont optionnelles et nécessitent une clé API.

### Quelles langues Lore supporte-t-il ?

L'interface CLI est bilingue : anglais et français. Configurez `language: "fr"` dans `.lorerc` pour le français.

### Lore fonctionne-t-il avec n'importe quel hébergeur Git ?

Oui. Lore fonctionne localement via des hooks Git. Compatible GitHub, GitLab, Bitbucket, ou tout hébergeur Git.

## Utilisation

### Puis-je passer la documentation pour un commit ?

Oui. Appuyez sur Ctrl+C pendant les questions — les réponses partielles sont sauvées dans pending. Ou ajoutez `[doc-skip]` à votre message de commit.

### Que se passe-t-il lors des commits de merge ?

Lore les ignore automatiquement — pas de documentation nécessaire.

### Que se passe-t-il en CI ou dans un environnement non-TTY ?

Les commits sont différés silencieusement dans pending. Dans les terminaux VS Code, Lore envoie une notification. Utilisez `lore pending resolve` plus tard.

### Puis-je documenter d'anciens commits retroactivement ?

Oui : `lore new --commit abc1234`

### Comment annuler un commit documenté ?

`lore delete <fichier>` avec confirmation.

## IA (Angela)

### L'IA est-elle obligatoire ?

Non. Lore fonctionne entièrement sans IA. Angela est optionnelle.

### Quels fournisseurs IA sont supportés ?

Anthropic (Claude), OpenAI (GPT), et Ollama (modèles locaux).

### Que fait Angela Draft sans API ?

Analyse structurelle locale : sections manquantes, conformité au guide de style, documents liés, vérifications de cohérence. Zéro appel réseau.

## Données et vie privée

### Où sont stockées mes données ?

Tout est dans `.lore/` dans votre dépôt. Rien n'est envoyé nulle part sauf si vous utilisez explicitement Angela Polish avec un fournisseur IA.

### Puis-je supprimer toutes les données Lore ?

`rm -rf .lore/` supprime tout. Votre historique Git et votre code ne sont pas touchés.

### Quelle licence pour Lore ?

AGPL-3.0. Une licence commerciale est disponible pour usage propriétaire.
