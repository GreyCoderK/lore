---
type: feature
date: "2026-04-15"
status: draft
---

# Invoice Management

## Module 1

### Endpoints

| Méthode | URL | Description |
|---------|-----|-------------|
| `POST` | `/api/invoices/search` | Recherche paginée des factures du mois |
| `POST` | `/api/invoices/export-excel` | Export XLSX async + envoi email |

### Filtres acceptés (12 champs)

`organizationId`, `branchCode`, `accountNumber`, `month` (yyyy-MM, **requis**), `referencePrefix`, `operationType`, `creditAmount` (+Min/Max), `debitAmount` (+Min/Max), `page`, `size`

## Sécurité

- Utilisateur authentifié obligatoire (Spring Security)
- `organizationId` et `branchCode` injectés par le controller (jamais depuis le client)
- Isolation des données par organisation via clauses `WHERE`
