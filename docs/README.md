# Documentation du backend

Cette documentation est la source de référence pour le backend Go.

## Documents principaux

- [Architecture](./architecture.md) — Couches, patterns, diagrammes
- [État de reprise pour les agents](./handoff.md) — Phases 1-6, status actuel
- [Roadmap](./roadmap.md) — Phases complétées et futures
- [Décisions (ADR)](./decisions/) — Décisions architecturales
  - [ADR 0001: Clean Architecture Backend](./decisions/0001-backend-architecture.md)
  - [ADR 0002: Code Cleanup & Governance](./decisions/0002-code-cleanup.md)
  - [ADR 0004: Test Strategy (Unit/Integration/E2E)](./decisions/0004-test-strategy.md)
  - [ADR 0005: Dependency Injection Pattern](./decisions/0005-dependency-injection.md)
  - [ADR 0006: Git Workflow & CI/CD Pipeline](./decisions/0006-git-workflow-and-cicd.md)
- [Spécifications](./specs/) — Spécifications fonctionnelles

## Infrastructure & Développement

- **CI/CD Pipeline** : [`CI_CD_PIPELINE.md`](../CI_CD_PIPELINE.md) — 7-stages complet
- **Commandes locales** : Voir [`Makefile`](../Makefile)
  - `make test` : tous les tests
  - `make test-unit` : tests unitaires
  - `make docker-compose up` : PostgreSQL pour intégration
  - `make lint` / `make fmt` : qualité code

## Références globales

- [Glossaire](../../docs/glossary.md)
- [Guide documentaire](../../docs/documentation-guide.md)
- [Workflow](../../docs/workflow.md)

## Règles

- Le périmètre est limité à `inbackend`.
- Les routes et contrats HTTP existants doivent être préservés sauf décision
  documentée.
- Toute décision spécifique au backend va dans `decisions/`.
- Les notes temporaires ne doivent pas être ajoutées à ce dossier.

## Fichiers obsolètes

- `internal/handlers/handlers.go` — **LEGACY (non utilisé)**
  - Contenait handlers monolithiques avant Clean Architecture
  - Remplacé par controllers + application handlers (Phases 1-4)
  - À supprimer lors de nettoyage futur
