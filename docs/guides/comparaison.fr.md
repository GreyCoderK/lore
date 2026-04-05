# Comment Lore se compare

## Le paysage

Il existe de nombreuses façons de documenter les décisions logicielles. La plupart partagent un défaut fondamental : elles demandent aux développeurs d'arrêter ce qu'ils font et d'écrire la documentation *séparément* du code. C'est comme demander à un chirurgien de rédiger son compte-rendu opératoire le lendemain de mémoire.

Lore prend une approche différente : **capturer au moment de la décision**, pas après.

## Comparaison rapide

| | **Lore** | Swimm | Confluence | GitBook | ADRs | Rien |
|---|---|---|---|---|---|---|
| **Quand** | Au commit | Après coup | Après coup | Après coup | Quand on y pense | Jamais |
| **Où** | Local (`.lore/`) | SaaS | SaaS | SaaS | Local (Markdown) | — |
| **Friction** | 90 secondes | 30 minutes | 30 minutes | 15 minutes | 15 minutes | 0 |
| **IA** | Angela (opt-in) | Générique | IA générique | IA générique | Aucune | — |
| **Lock-in** | Markdown | Propriétaire | Propriétaire | Mixte | Markdown | — |
| **Hors ligne** | Oui (tout) | Non | Non | Non | Oui | — |
| **Automatisé** | Hook post-commit | Manuel | Manuel | Manuel | Manuel | — |
| **Bilingue** | EN/FR intégré | EN uniquement | Multi-langue | Multi-langue | Manuel | — |
| **Prix** | Gratuit (AGPL) | $28/siège | $5,75/user | $8/user | Gratuit | Gratuit |

## Pourquoi le moment du commit compte

La connaissance a une demi-vie. Au moment du commit, le développeur sait exactement pourquoi il a fait ce choix. Une heure plus tard, les détails commencent à s'estomper. Une semaine plus tard, c'est flou. Six mois plus tard — disparu.

```
Moment du commit  ████████████████████ 100% contexte
1 heure après     ████████████████░░░░  80% contexte
1 jour après      ████████████░░░░░░░░  60% contexte
1 semaine après   ████████░░░░░░░░░░░░  40% contexte
1 mois après      ████░░░░░░░░░░░░░░░░  20% contexte
6 mois après      ░░░░░░░░░░░░░░░░░░░░   0% contexte
```

Lore capture au sommet. Tout le reste capture sur la pente descendante.

## Comparaisons détaillées

### Lore vs Swimm

**Swimm** est une plateforme SaaS de documentation qui vit aux côtés de votre code. Elle est bien conçue et a de bonnes intégrations IDE.

| Aspect | Lore | Swimm |
|--------|------|-------|
| **Moment de capture** | Automatique au commit | Manuel, quand on y pense |
| **Localisation données** | Votre repo (`.lore/docs/`) | Serveurs Swimm |
| **Hors ligne** | Pleinement fonctionnel | Nécessite internet |
| **IA** | Angela : zéro-API + IA optionnelle | Assistant IA générique |
| **Prix** | Gratuit pour toujours | $28/siège/mois |
| **Risque vendeur** | Aucun (fichiers Markdown) | L'entreprise peut pivoter, augmenter ses prix, ou fermer |

**Quand Swimm est meilleur :** Grandes équipes avec édition collaborative et widgets IDE. **Quand Lore est meilleur :** Développeurs qui veulent zéro friction, local-first, capture au commit sans abonnement.

### Lore vs Confluence

**Confluence** est le wiki d'Atlassian. C'est le choix entreprise par défaut.

Le problème fondamental : personne ne met à jour Confluence. Les pages pourrissent. La page "Architecture d'authentification" a été écrite il y a 18 mois par quelqu'un qui est parti. Elle décrit un système qui n'existe plus. Tout le monde sait qu'elle est fausse, mais personne n'a le temps de la corriger.

Lore n'a pas ce problème car les documents sont créés **au moment du changement**. Ils ne peuvent pas pourrir silencieusement — `lore angela review` détecte les contradictions, et `lore doctor` signale le contenu obsolète.

### Lore vs ADRs

Les **Architecture Decision Records** (ADRs) sont des fichiers Markdown qui documentent les grandes décisions architecturales. Ils sont excellents.

Lore n'est **pas** un remplacement des ADRs. Ils sont complémentaires :

| | ADRs | Lore |
|---|---|---|
| **Portée** | Grandes décisions rares | Décisions quotidiennes au commit |
| **Fréquence** | Une fois par trimestre | Chaque commit |
| **Déclencheur** | Manuel ("quelqu'un devrait écrire un ADR") | Automatique (hook post-commit) |
| **Exemple** | "On a choisi PostgreSQL plutôt que MongoDB" | "Pourquoi on a ajouté cet index à la table users" |

La meilleure configuration : ADRs pour la vision globale, Lore pour les détails quotidiens. Avec le temps, les documents Lore alimentent naturellement les discussions ADR.

### Lore vs Conventional Commits

Les **Conventional Commits** (`feat:`, `fix:`, `docs:`) standardisent le **quoi**. Lore capture le **pourquoi**. Ils fonctionnent magnifiquement ensemble :

```bash
git commit -m "feat(auth): add JWT middleware"
# Conventional Commit vous dit : c'est une feature, dans le scope auth
# Lore demande : POURQUOI JWT ? POURQUOI maintenant ? Quelles alternatives ?
```

Lore pré-remplit même le champ "Quoi" depuis votre message de commit. Si vous utilisez les Conventional Commits, le Decision Engine de Lore reconnaît le préfixe de type et ajuste le scoring en conséquence.

### Lore vs Ne rien faire

La plupart des équipes ne font rien. Ça marche — jusqu'à ce que ça ne marche plus.

Le coût du contexte perdu est réel mais invisible :

- **Refactors aveugles** — Supprimer du code qui était là pour une raison que personne ne se rappelle
- **Erreurs répétées** — Prendre la même décision (et faire la même erreur) qui avait déjà été prise puis annulée
- **Friction d'onboarding** — Les nouveaux passent des semaines à demander "pourquoi c'est comme ça ?"
- **Retards de review** — Les PRs stagnent car les reviewers ne comprennent pas le raisonnement

Le pari de Lore : **90 secondes par commit, ça vaut le coup.** Sur un an, c'est ~6 heures de documentation pour ~1500 commits. Le retour : une base de connaissances cherchable qui économise des centaines d'heures de "pourquoi on a fait ça ?"

## Voir aussi

- [Philosophie](philosophy.md) — Les principes derrière Lore
- [Quickstart](../getting-started/quickstart.md) — Essayez en 5 minutes
- [FAQ](../faq.md) — Questions fréquentes
