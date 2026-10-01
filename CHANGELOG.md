# Changelog

Format basé sur [Keep a Changelog 1.1.0](https://keepachangelog.com/fr/1.1.0/).

## [Unreleased]

### Security

- Go 1.26.8 : corrige les vulnérabilités atteignables de la stdlib 1.25.0 (`Q-03`).
- `golang.org/x/crypto` v0.56.0 et `moby/go-archive` v0.3.0 : corrigent GO-2026-6355, GO-2026-6354 et GO-2026-6253, atteignables depuis les tests.

### Added

- `GET /openapi.yaml` (contrat embarqué) et `GET /docs` (Swagger UI, version épinglée avec empreintes d'intégrité) ; `API_DOCS_ENABLED` (actif par défaut, inactif en production, valeur invalide refusée).
- Messages `error` et `message` en français ou en anglais selon `Accept-Language` (français pour `fr` et ses variantes, anglais sinon) ; `Content-Language` et `Vary: Accept-Language` sur toutes les réponses.
- `GET /ready` : disponibilité (base joignable sous 2 s, sinon 503 `DATABASE_UNAVAILABLE`) ; `/health` reste une sonde de vivacité sans base ; réponses des deux sondes non mises en cache.
- Gabarit « ajouter un endpoint » (`docs/adding-an-endpoint.md`).
- Identifiant de requête `X-Request-Id` (repris du client s'il est valide), présent dans les logs ; erreurs d'Echo (route inconnue, méthode non supportée, corps trop gros) dans l'enveloppe avec `ROUTE_NOT_FOUND`, `METHOD_NOT_ALLOWED`, `PAYLOAD_TOO_LARGE` (413 au-delà de 1 MiB).
- Délais du serveur HTTP, arrêt gracieux sur SIGINT/SIGTERM et pool SQL borné.
- `CORS_ALLOWED_ORIGINS` (`*` par défaut).
- `make cover` : couverture de `internal/` avec minimum bloquant de 85 %, aussi en CI (`Q-04`).
- Migrations versionnées golang-migrate (`make migrate-up/down/force/new`, ADR 0007) ; `AutoMigrate` retiré (`B-05`).
- `B-02` : clé étrangère `task_models.user_id` -> `user_models.id` ; créer une tâche pour un utilisateur inexistant répond 404 `USER_NOT_FOUND`.
- Champ `error_code` (code stable, enum dans le contrat) dans toute réponse d'erreur ; une seule table erreur → statut/code/message (lot 4).
- `make mutate` : tests de mutation (gremlins) sur le diff, hors CI.
- Spécification OpenAPI 3.1 du contrat HTTP (`docs/specs/api/openapi.yaml`).
- Tests de régression pour `Q-01` (unitaires, e2e, intégration PostgreSQL).
- Contrôles bloquants en CI : golangci-lint (`make lint`), Redocly sur le contrat
  OpenAPI (`make lint-api`), govulncheck (`make vuln`) et actionlint sur les
  workflows (`make lint-ci`).

### Changed

- Contrat : `message` et `version` des réponses de santé ne sont plus des constantes ; `Stats` sans `format: int64` ; la réponse de `DELETE /tasks/{id}` déclare `message`.
- Healthcheck de Docker Compose et job `docker` de la CI sur `/ready` ; lignes d'accès des sondes réussies en Debug.
- Logs par `slog` seul (zap retiré) : JSON au niveau Info en production, texte au niveau Debug en développement, position du code source sur chaque ligne.
- Logs JSON de production : `ts` devient `time` (RFC 3339), `level` en majuscules, `caller` devient l'objet `source` ; la durée `latency` est en nanosecondes ; plus de pile d'appels sur les erreurs.
- Échec au démarrage journalisé en ERROR (« Server failed »), code de sortie 1 inchangé.
- `Q-11` : le hook pre-push (lint, build, tests unitaires, sans Docker) bloque réellement en cas d'échec.
- Logs de requête via `RequestLoggerWithConfig` (méthode, chemin sans query, route, statut, durée, request id) à la place de `middleware.Logger` déprécié ; format de ligne différent.
- `DATABASE_URL` doit être une URL `postgres://` (le format `clé=valeur` est refusé au démarrage avec un message clair) ; `.env.example` mis à jour.
- Emails existants normalisés en minuscules et unicité insensible à la casse en base ; le démarrage échoue avec un message clair s'il existe des doublons par casse ou des tâches orphelines (rien n'est supprimé).
- `B-03` : créer un utilisateur avec un email déjà pris répond 409 `EMAIL_TAKEN` au lieu de 500 (comparaison sur l'email normalisé).
- Emails normalisés à la création (espaces retirés, minuscules) et format minimal validé (un `@`, domaine à deux labels non vides, ni espace ni caractère de contrôle) (400 `VALIDATION_FAILED`).
- `B-04` : supprimer une tâche absente répond 404 `TASK_NOT_FOUND` au lieu de 200.
- `GET /api/v1/stats` compte les utilisateurs sans les charger.
- `B-08` : l'email et le nom des utilisateurs ne sont plus journalisés (l'identifiant seulement), ni le texte brut des erreurs PostgreSQL ; le logger GORM est silencieux.
- CI : `codecov-action` v3 remplacé par v5 (runtime signalé trop ancien par actionlint) ; actionlint ajouté.
- `Q-03` : CI, une seule version de Go (celle de `go.mod`).
- `Q-03` : CI, nouveau job `docker` bloquant (image, pile compose, `/health`).
- Backend restructuré en couches domain/application/infrastructure/controllers
  (DDD/Clean Architecture), CQRS applicatif introduit.
- Logging migré vers `slog` (délégué à `zap`), réponses HTTP standardisées.
- Tests réorganisés en unit/integration/e2e (ADR 0004).
- Documentation d'architecture et ADR 0006 alignées sur l'état réel du code.
- `make lint` et `make fmt` utilisent golangci-lint, gofumpt et goimports.
- Échec au démarrage : le log fatal devient « Server failed » avec un champ
  `error` ; le code de sortie reste 1.
- Contrat OpenAPI : descriptions de tags ajoutées, schéma inutilisé retiré.

### Fixed

- `Q-03` : l'image Docker se construit et `docker compose up` démarre PostgreSQL et le backend.
- `Q-05` : le scan de vulnérabilités de la CI bloque (govulncheck remplace `nancy`).
- `Q-12` : `GET /api/v1/users/{id}/tasks` retourne les tâches triées par
  `id` croissant.
- `Q-01` : `PUT /api/v1/tasks/{id}` persiste les mises à jour partielles
  ([#3](https://github.com/fdanielsin/inbackend/pull/3)).
- Jobs CI d'intégration et E2E ciblent désormais leurs packages réels.
- `B-06` : `make lint` exécute réellement golangci-lint, comme l'annonce `make help`.

### Removed

- `Q-10` : code mort retiré.
- Configuration obsolète : `.env.example` nettoyé (variables réellement lues
  uniquement).

## [1.0.0] - 2026-09-17

### Added

- Endpoint de statistiques agrégées.

[Unreleased]: https://github.com/fdanielsin/inbackend/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/fdanielsin/inbackend/releases/tag/v1.0.0
