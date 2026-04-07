# FAQ

## Général

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

### Pourquoi un dialog s'affiche au lieu des questions interactives ?

Git redirige stdin vers `/dev/null` pour les hooks. Le hook de Lore reconnecte stdin depuis le terminal pour poser les questions de façon interactive.

**Fix :** Réinstallez le hook :

```bash
lore hook uninstall
lore hook install
```

Vérifiez que le hook contient la redirection stdin :

=== "macOS / Linux"

    ```bash
    grep "dev/tty" .git/hooks/post-commit
    # Devrait afficher : exec lore _hook-post-commit < /dev/tty
    ```

    Le hook utilise `< /dev/tty` pour reconnecter stdin depuis le terminal. Dans les environnements où `/dev/tty` n'est pas disponible (CI, Docker, pipes), les commits sont différés silencieusement.

=== "Windows (Git Bash)"

    ```bash
    grep "dev/tty" .git/hooks/post-commit
    # Même mécanisme — Git Bash (MSYS2) fournit /dev/tty
    ```

    Windows utilise Git Bash pour les hooks, qui fournit `/dev/tty` comme les systèmes Unix. Si vous utilisez PowerShell ou CMD directement, les commits seront différés vers pending.

> **Note :** Dans les environnements où le terminal n'est pas disponible (CI, Docker, pipes), les commits sont toujours différés vers pending — c'est voulu. Voir [Détection Contextuelle](guides/contextual-detection.md) pour les détails.

### Comment Lore gère les notifications sur les différentes plateformes ?

Quand un commit est différé (non-TTY), Lore peut envoyer une notification selon `notification.mode` dans `.lorerc` :

| Plateforme | Mode `dialog` | Mode `notify` |
|------------|--------------|---------------|
| **macOS** | Dialog AppleScript (`osascript`) | `terminal-notifier` (si installé) ou `display notification` |
| **Linux** | `zenity`, `kdialog` ou `yad` (le premier disponible) | `notify-send` |
| **Windows** | PowerShell `System.Windows.Forms` balloon | Notification PowerShell balloon |

Le stockage des clés API varie aussi par plateforme :

| Plateforme | Backend keychain |
|------------|-----------------|
| **macOS** | Trousseau système (`security` CLI) |
| **Linux** | `secret-tool` (GNOME Keyring / KWallet) |
| **Windows** | Windows Credential Manager (fallback sur config) |

### Puis-je documenter d'anciens commits rétroactivement ?

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

### Que fait Angela Polish ?

Envoie votre document au fournisseur IA pour réécriture. Affiche un diff interactif — acceptez ou rejetez chaque changement individuellement. Un seul appel API par document.

### J'ai un abonnement Claude.ai mais pas de crédits API. Puis-je utiliser Angela ?

`angela draft` fonctionne **100% hors ligne** — pas besoin d'API. Pour les fonctions polish/review, deux options gratuites :

**Option 1 : Ollama (local, gratuit)**

```bash
brew install ollama
ollama pull llama3.2
```

```yaml
# .lorerc
ai:
  provider: "ollama"
  model: "llama3.2"
```

**Option 2 : Polish manuel via le chat Claude.ai**

1. Lancez `lore angela draft <fichier>` pour les suggestions structurelles
2. Copiez le contenu de votre document dans Claude.ai avec ce prompt :

```
Améliore ce document de décision technique. Garde le format Markdown
avec les sections : What, Why, Alternatives, Impact. Sois concis et technique :

[coller le contenu du document]
```

3. Collez la version améliorée dans votre fichier

> **Note :** Claude.ai (abonnement chat) et l'API Anthropic sont deux produits séparés avec une facturation séparée. L'API nécessite des crédits achetés sur [console.anthropic.com](https://console.anthropic.com).

## Données et vie privée

### Où sont stockées mes données ?

Tout est dans `.lore/` dans votre dépôt. Rien n'est envoyé nulle part sauf si vous utilisez explicitement Angela Polish avec un fournisseur IA.

### Puis-je supprimer toutes les données Lore ?

`rm -rf .lore/` supprime tout. Votre historique Git et votre code ne sont pas touchés.

### Quelle licence pour Lore ?

AGPL-3.0. Une licence commerciale est disponible pour usage propriétaire.

## Utilisateur avancé

### Comment ajuster le Decision Engine ?

Éditez `.lorerc` :

```yaml
decision:
  threshold_full: 50      # Plus bas = plus de questions complètes (défaut : 60)
  always_ask: [feat, breaking, security]
  always_skip: [docs, style, ci, build, chore]
```

Lancez `lore decision --explain HEAD` pour voir le scoring de n'importe quel commit.

### Puis-je utiliser Lore dans un monorepo ?

Oui. `lore init` à la racine du repo. Les documents capturent le chemin complet des fichiers modifiés. Utilisez `lore show --type decision` + recherche par mot-clé pour trouver les décisions par service.

### Puis-je utiliser un modèle IA custom avec Ollama ?

```yaml
# .lorerc
ai:
  provider: "ollama"
  model: "llama3"          # N'importe quel modèle disponible dans votre instance Ollama
  endpoint: "http://localhost:11434"
```

Pas de clé API nécessaire. Le modèle tourne entièrement sur votre machine.

### Comment écrire un guide de style custom pour Angela ?

Ajoutez une section `style_guide` dans `.lorerc` :

```yaml
angela:
  style_guide:
    tone: "technique mais accessible"
    max_length: 500
    required_sections: ["Why", "Alternatives Considered"]
    avoid: ["voix passive", "jargon sans explication"]
```

Angela Draft et Polish vérifieront ces règles.

### Puis-je exporter mon corpus ?

Votre corpus EST l'export — ce sont des fichiers Markdown dans `.lore/docs/`. Copiez-les n'importe où. Ils sont autonomes avec du front matter YAML. Pas de format propriétaire, pas de lock-in.

### Comment migrer depuis les ADRs vers Lore ?

Vous ne migrez pas — ils sont complémentaires. Gardez vos ADRs pour les grandes décisions architecturales. Utilisez Lore pour le "pourquoi" quotidien derrière chaque commit.

### Puis-je utiliser Lore en CI/CD ?

```bash
# Échouer le build si des docs sont en attente
[ $(lore pending --quiet | wc -l) -eq 0 ] || exit 1

# Échouer le build si le corpus est en mauvaise santé
[ $(lore doctor --quiet) -eq 0 ] || exit 1

# Générer le badge de couverture
lore status --badge >> $GITHUB_STEP_SUMMARY
```

### Comment gérer les conflits de merge dans `.lore/docs/` ?

Rare — chaque commit crée un nom de fichier unique. Si ça arrive, résolvez comme tout conflit Markdown. Puis `lore doctor --fix` pour reconstruire l'index.

### Quel est l'impact sur les performances ?

Négligeable. Le Decision Engine score en ~0.4ms. Le hook entier (y compris le rendu des questions) ajoute < 100ms à un commit quand il est auto-skip. Quand vous répondez aux questions, le temps est votre vitesse de frappe.

### Comment désactiver Lore temporairement ?

```bash
# Ignorer un commit
git commit -m "chore: deps [doc-skip]"

# Désactiver le hook entièrement
lore hook uninstall

# Réactiver plus tard
lore hook install
```

---

**Question non listée ?** [Posez-la sur GitHub Discussions Q&A](https://github.com/greycoderk/lore/discussions/categories/q-a)
