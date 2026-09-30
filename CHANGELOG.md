# Changelog

Format basé sur [Keep a Changelog 1.1.0](https://keepachangelog.com/fr/1.1.0/).

## [Unreleased]

### Added

- Spécification OpenAPI 3.1 du contrat HTTP (`docs/specs/api/openapi.yaml`).
- Tests de régression pour `Q-01` (unitaires, e2e, intégration PostgreSQL).
- Contrôles bloquants en CI : golangci-lint (`make lint`), Redocly sur le contrat
  OpenAPI (`make lint-api`) et govulncheck (`make vuln`).

### Changed

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
