# Documentation du backend

Cette documentation est la source de référence pour le backend Go.

## Documents principaux

- [Architecture](./architecture.md)
- [État de reprise pour les agents](./handoff.md)
- [Ajouter un endpoint](./adding-an-endpoint.md)
- [Déploiement](./deployment.md)
- [Dépannage](./troubleshooting.md)
- [Roadmap](./roadmap.md)
- [Décisions](./decisions/)
  - [ADR 0001 — Architecture](./decisions/0001-backend-architecture.md)
  - [ADR 0002 — Cleanup et gouvernance](./decisions/0002-code-cleanup.md)
  - [ADR 0004 — Stratégie de tests](./decisions/0004-test-strategy.md)
  - [ADR 0005 — Injection de dépendances](./decisions/0005-dependency-injection.md)
  - [ADR 0006 — Workflow et CI](./decisions/0006-git-workflow-and-cicd.md)
- [Spécifications](./specs/)
  - [API OpenAPI 3.1](./specs/api/openapi.yaml)

Les fichiers de CI et les commandes du projet restent à leur emplacement
réel. Ne pas considérer un lien comme valide sans vérifier que la cible existe.

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
- Les statuts du handoff et des ADR doivent être vérifiés dans le code, les
  tests et Git avant d'être considérés comme acquis.
