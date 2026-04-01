# FAQ

## General

### Qu'est-ce que Lore ?

Un outil CLI qui capture le *pourquoi* derriere vos changements de code au moment du commit. Trois questions, quatre-vingt-dix secondes, un document Markdown pour toujours.

### Lore necessite-t-il une connexion internet ?

Non. Tout fonctionne hors ligne par defaut. Les fonctions IA (Angela) sont optionnelles et necessitent une cle API.

### Quelles langues Lore supporte-t-il ?

L'interface CLI est bilingue : anglais et francais. Configurez `language: "fr"` dans `.lorerc` pour le francais.

### Lore fonctionne-t-il avec n'importe quel hebergeur Git ?

Oui. Lore fonctionne localement via des hooks Git. Compatible GitHub, GitLab, Bitbucket, ou tout hebergeur Git.

## Utilisation

### Puis-je passer la documentation pour un commit ?

Oui. Appuyez sur Ctrl+C pendant les questions — les reponses partielles sont sauvees dans pending. Ou ajoutez `[doc-skip]` a votre message de commit.

### Que se passe-t-il lors des commits de merge ?

Lore les ignore automatiquement — pas de documentation necessaire.

### Que se passe-t-il en CI ou dans un environnement non-TTY ?

Les commits sont differes silencieusement dans pending. Dans les terminaux VS Code, Lore envoie une notification. Utilisez `lore pending resolve` plus tard.

### Puis-je documenter d'anciens commits retroactivement ?

Oui : `lore new --commit abc1234`

### Comment annuler un commit documente ?

`lore delete <fichier>` avec confirmation.

## IA (Angela)

### L'IA est-elle obligatoire ?

Non. Lore fonctionne entierement sans IA. Angela est optionnelle.

### Quels fournisseurs IA sont supportes ?

Anthropic (Claude), OpenAI (GPT), et Ollama (modeles locaux).

### Que fait Angela Draft sans API ?

Analyse structurelle locale : sections manquantes, conformite au guide de style, documents lies, verifications de coherence. Zero appel reseau.

## Donnees et vie privee

### Ou sont stockees mes donnees ?

Tout est dans `.lore/` dans votre depot. Rien n'est envoye nulle part sauf si vous utilisez explicitement Angela Polish avec un fournisseur IA.

### Puis-je supprimer toutes les donnees Lore ?

`rm -rf .lore/` supprime tout. Votre historique Git et votre code ne sont pas touches.

### Quelle licence pour Lore ?

AGPL-3.0. Une licence commerciale est disponible pour usage proprietaire.
