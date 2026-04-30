---
type: reference
date: "2026-04-17"
status: published
related:
    - angela-draft.fr.md
    - angela-polish.fr.md
    - angela-review.fr.md
    - angela-consult.fr.md
angela_mode: reference
---
# Personas Angela

Angela évalue votre documentation à travers **7 lentilles d'expertise distinctes** — chacune avec ses priorités, ses angles morts, et ses questions signature. Ensemble, elles couvrent les axes sur lesquels la doc technique échoue : prose imprécise, manque de mental model utilisateur, contrats incomplets, hypothèses non vérifiées, drift narratif, désalignement business, et structure faible.

Les personas sont disponibles en trois modes d'activation : `angela consult` pour un deep-dive sur une seule persona, `angela polish --persona` pour orienter une réécriture, et `angela review --persona` pour injecter des lentilles dans l'analyse de cohérence du corpus.

## Les sept

| Icône | ID | Nom | Focus |
|---|---|---|---|
| ✏️ | `tech-writer` | Salou | Précision rédactionnelle et clarté technique |
| 🎨 | `ux-designer` | Gougou | Empathie utilisateur, mental models, accessibilité |
| 🔌 | `api-designer` | Ouattara | Contrats API, docs synthesizer-ready, sémantique HTTP |
| 🔍 | `qa-reviewer` | Kouame | Assurance qualité et critères de validation |
| 🏗️ | `architect` | Doumbia | Design système, trade-offs, scalabilité |
| 📊 | `business-analyst` | Beda | Traçabilité des exigences, valeur business |
| 📖 | `storyteller` | Affoue | Clarté narrative et authenticité |

Les prénoms affichés (Salou, Gougou, Ouattara, Kouame, Doumbia, Beda, Affoue) sont des prénoms ivoiriens courants. Lore est construit en Côte d'Ivoire et le projet assume ses racines culturelles plutôt que de retomber sur des placeholders génériques de l'industrie tech. L'émoji garde l'identité persona scannable dans les sorties terminal où les noms peuvent être tronqués.

## Quand utiliser laquelle

- **Vous écrivez une doc de feature ?** Commencez par `tech-writer` (Salou) pour la qualité rédactionnelle, puis `ux-designer` (Gougou) pour le mental model du lecteur.
- **Vous documentez un endpoint API ?** `api-designer` (Ouattara) attrape les méthodes manquantes, les naming incohérents, et les trous headers/body qui cassent les imports Postman.
- **Vous shippez une décision ?** Lancez `architect` (Doumbia) pour la clarté des trade-offs, et `qa-reviewer` (Kouame) pour forcer la colonne « qu'est-ce qui peut mal se passer ».
- **Un guide long ou une pièce d'onboarding ?** `storyteller` (Affoue) vérifie que la narration ne dérive pas en milieu de paragraphe.
- **Un spec produit ou feature ?** `business-analyst` (Beda) vérifie que les exigences renvoient à un objectif business nommé.

Pour le travail sur un seul document, prenez 1 persona. Pour une review corpus, 3–4 personas complémentaires donnent un signal cross-lens (quand deux personas flaguent indépendamment le même problème, cette convergence est un marqueur haute confiance — Angela la surface via l'attribution `Flaguée par :`).

## Modes d'activation

### `angela consult <persona> <fichier>` — check offline mono-lentille

```bash
lore angela consult api-designer docs/features/login.md
lore angela consult tech-writer docs/guides/quickstart.md
```

Pas d'appel IA, pas d'écriture. Lance la draft-check lens de la persona et affiche les suggestions. Utile après un polish ou une édition manuelle quand vous voulez l'avis d'un expert sans relancer tout le pipeline de draft.

Lancez `lore angela consult` (sans argument) pour lister toutes les personas.

### `angela polish --persona <id>` — orienter la réécriture IA

```bash
lore angela polish docs/features/login.md --persona ux-designer
```

Biaise le polish IA vers les priorités de la persona — par ex. `ux-designer` met l'accent sur les flows utilisateur et la clarté des chemins d'erreur ; `api-designer` met l'accent sur les shapes requête/réponse. Voir [angela-polish.fr.md](angela-polish.fr.md) pour le flow complet de polish.

### `angela review --persona <id>` — cohérence corpus multi-persona

```bash
lore angela review --persona tech-writer --persona ux-designer --persona api-designer --persona qa-reviewer
```

Chaque lens persona est injectée dans le prompt de review. L'IA est instruite d'attribuer chaque finding à la/les persona(s) dont l'expertise l'a flagué. Quand plusieurs personas concordent, elles sont listées ensemble sous `Flaguée par :` — signal fort que le problème compte à travers les lentilles. Voir [angela-review.fr.md](angela-review.fr.md).

Alternativement, configurez les personas dans `.lorerc` et activez-les via `--use-configured-personas` (saute la confirmation interactive).

## Session d'exemple

```bash
# 1. Voir les personas disponibles
$ lore angela consult

Available personas:
  🔌 api-designer         Ouattara
                          API contracts, synthesizer-ready docs, HTTP semantics
  # ... (7 personas)

# 2. Spot-check sur une doc API
$ lore angela consult api-designer docs/features/invoices.md
  warning  persona  [🔌 Ouattara] Endpoints listés sans exemple de requête HTTP — ajoute un bloc ```http avec méthode, URL, headers et body

# 3. Review de tout le corpus à travers plusieurs lentilles
$ lore angela review --persona tech-writer --persona qa-reviewer

  + gap   Documentation personas Angela manquante
          [abc123] commands/angela-consult.md vs commands/angela-consult.fr.md
          Flaguée par : Kouame, Gougou
```

## Config sélection de personas

Vous pouvez fixer un set par défaut de personas dans `.lorerc` :

```yaml
angela:
  review:
    personas:
      selection: "manual"
      manual_list:
        - tech-writer
        - api-designer
        - qa-reviewer
```

Puis `lore angela review --use-configured-personas` saute la confirmation interactive. Voir [angela-review.fr.md](angela-review.fr.md) et [config.fr.md](config.fr.md) pour la cascade complète.

## I4 — Zéro hallucination

L'injection de persona ne relâche **pas** la règle d'evidence. Chaque finding persona-attribué doit porter une citation vérifiable du corpus. Les findings sans evidence sont rejetés par le validateur post-processing, peu importe quelle persona les a flagués.

## Voir aussi

- [angela consult](angela-consult.fr.md) — check persona unique à la demande
- [angela polish](angela-polish.fr.md) — réécriture assistée IA avec support `--persona`
- [angela review](angela-review.fr.md) — cohérence corpus avec lentilles multi-persona
- [config](config.fr.md) — comment configurer les personas par défaut dans `.lorerc`
