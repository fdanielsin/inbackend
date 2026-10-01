# Roadmap d'implémentation

## Phase 0 — état initial

- Lire le code actuel et les tests.
- Recenser les routes et les contrats de réponse.
- Ne pas modifier le frontend.

## Phase 1 — fondations du domaine

- **Statut : conforme après remise en conformité**
- Créer les packages `domain/user` et `domain/task`.
- Définir les entités et les erreurs métier.
- Définir les interfaces de repository.
- Ajouter les règles métier réellement identifiées.

## Phase 2 — application et CQRS

- **Statut : conforme après remise en conformité**
- Créer les commands utilisateur et tâche.
- Créer les queries utilisateur et tâche.
- Déplacer l'orchestration hors des handlers HTTP.
- Tester les use cases avec des repositories de test.

## Phase 3 — infrastructure

- **Statut : conforme après remise en conformité**
- Créer les modèles PostgreSQL/GORM.
- Implémenter les repositories.
- Isoler la connexion et les migrations.
- Conserver les transactions PostgreSQL locales.

## Phase 4 — controllers et composition

- **Statut : conforme après remise en conformité**
- Remplacer les handlers monolithiques par des controllers HTTP.
- Conserver les routes existantes.
- Construire les dépendances dans `internal/app` (`app.NewRouter`, lot 3 ; `main.go` ne garde que la configuration).
- Centraliser la conversion des erreurs HTTP.

## Phase 5 — validation

- **Statut : CI corrigée ; intégration PostgreSQL exécutée avec succès via
  Docker/Podman pour le lot 1 (2026-09-29, voir Plan de lots) ; sans provider,
  elle reste ignorée explicitement, jamais réussie par défaut.**
- Adapter les tests unitaires.
- Conserver et corriger les tests d'intégration.
- Exécuter les tests end-to-end.
- Exécuter `go test ./...` depuis `inbackend`.
- Vérifier le formatage et le build.

## Plan de lots (backend)

Suite à l'audit croisé du 2026-09-29 (voir `docs/roadmap.md` à la racine pour
le détail des identifiants et le plan complet, y compris les lots frontend et
QA) :

| Lot | Objet | Constats backend | État |
| --- | --- | --- | --- |
| 0 | CI verte | — | Fait : `develop` à `66335a8`, run CI `36632944817` vert. |
| 1 | `PUT /api/v1/tasks/{id}` ignorait les valeurs falsy/vides | `Q-01` | **Corrigé** : PR [inbackend#3](https://github.com/fdanielsin/inbackend/pull/3) mergée dans `develop` (`b28ee65`), CI verte. |
| 2 | Erreurs lisibles | `B-03`, `B-04` (+ `F-01`, `F-02` côté frontend) | Backend accepté QA, en PR (`feat/lot4-errors-contrat`) ; frontend à faire. |
| 3 | Tests honnêtes | `Q-06`, `Q-07`, `Q-12`, `Q-09`, `Q-10` | **Fait** : `Q-12` (PR #6, `d289dc2`), `Q-10` (nettoyage), `Q-06`, `Q-07`, `Q-09` (lot 3, PR #9). |
| 4 | Schéma (migrations versionnées d'abord) | `B-05`, puis `B-02`, `B-07` (pagination : décision de contrat) | `B-05` et `B-02` codés sur `feat/lot5-migrations` (en attente de QA) ; `B-07` à décider ; `B-08` fait au lot 2 backend. |
| 5 | Portes CI | `Q-04`, `Q-05`, `Q-13`, `Q-11`, `B-06` | Partiel : `Q-05` et `B-06` faits (PR #7) ; `Q-04` (`make cover`, seuil 85 %) et `Q-11` (pre-push) codés sur `chore/lot6-outillage` ; `Q-13` à faire. |
| 6 | Docker | `Q-03` | Accepté QA (PR #8) ; merge bloqué par la CI (compte GitHub). |
| 7 | Documentation | `F-03` (frontend) | À faire |
| 8 | Fonctionnalités | `FP-03`, `FP-01`, `FP-02`, `BP-02` (tri), `FP-04` | À faire |
| — | Nettoyage backend (demande utilisateur, hors audit) | — | **Fait** : mergé dans `develop` (PR #5), sans changement de comportement ni de contrat. |
| — | Outillage lint (rattaché au lot 5) | `Q-05`, `B-06` | **Fait** (PR #7) : golangci-lint, govulncheck et Redocly bloquants en CI, sans changement de comportement. |

## Règle de progression

Chaque phase doit préserver le comportement existant. Toute modification
fonctionnelle ou décision nouvelle doit être ajoutée à `docs/decisions/`.
