# Changelog

Format basé sur [Keep a Changelog 1.1.0](https://keepachangelog.com/fr/1.1.0/).

## [Unreleased]

### Added

- Spécification OpenAPI 3.1 du contrat HTTP (`docs/specs/api/openapi.yaml`).
- Suite de tests unitaires, d'intégration (PostgreSQL via Testcontainers) et
  end-to-end (SQLite).

### Changed

- Backend restructuré en couches domain/application/infrastructure/controllers
  (Clean Architecture / DDD), CQRS applicatif pour les utilisateurs et les
  tâches.
- Logging migré vers `slog` (délégué à `zap`), réponses HTTP standardisées.
- Documentation d'architecture et ADR 0006 alignées sur l'état réel du code
  (structure des packages, comportement de la CI, hooks Git, durée mesurée du
  pipeline).

### Fixed

- `Q-01` (PR [inbackend#3](https://github.com/fdanielsin/inbackend/pull/3)) :
  `PUT /api/v1/tasks/{id}` applique correctement les mises à jour partielles —
  les valeurs `false`/`""` explicites sont persistées, un `title` vide est
  rejeté (400), une tâche disparue entre-temps renvoie 404 au lieu d'un succès
  silencieux.
- Jobs CI d'intégration et E2E ciblent désormais leurs packages réels.

### Removed

- `Q-10` : code mort retiré (`internal/infrastructure/database/testdb.go`,
  `GetZap`/`ZapLog`, `zapLevelForSlog`, méthode `Delete` inutilisée sur
  `user.Repository`).
- Configuration obsolète : `.env.example` n'expose plus que les variables
  réellement lues ; dépendance implicite à `golangci-lint` retirée de
  `make lint`.

## [1.0.0] - 2026-09-17

### Added

- Endpoint de statistiques agrégées.
