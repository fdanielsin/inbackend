# Spécification de test — stratégie actuelle

## Statut

`active` — stratégie appliquée par l'arborescence et les commandes du projet.

## Catégories

- **Unit** : logique de domaine sans I/O dans `tests/units/domain/`.
- **Integration** : contrats des repositories PostgreSQL dans
  `tests/integration/persistence/`, avec Testcontainers.
- **E2E** : flux HTTP des controllers dans `tests/e2e/`, avec base SQLite
  en mémoire pour un feedback local rapide.

## Commandes de validation

```text
make test-unit
make test-integration
make test-e2e
make test
make lint
go vet ./...
go build ./...
```

## Règles

- Les tests doivent préserver le comportement HTTP existant.
- Les cas nominaux et les erreurs importantes doivent être couverts.
- Une absence de Docker/Podman doit apparaître comme un test PostgreSQL
  ignoré, avec sa cause.
- Une modification de contrat doit mettre à jour
  `docs/specs/api/openapi.yaml` et les tests concernés.

## Limites connues

- La couverture chiffrée par couche n'est pas encore définie.
- Les tests E2E locaux utilisent SQLite et ne remplacent pas la validation
  PostgreSQL.
