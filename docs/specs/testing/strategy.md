# Spécification de test — stratégie actuelle

## Statut

`active` — stratégie appliquée par l'arborescence et les commandes du projet.

## Catégories

- **Unit** : logique sans I/O dans `tests/units/domain/`,
  `tests/units/application/` (handlers, stubs) et `tests/units/shared/`.
- **Integration** : contrats des repositories PostgreSQL dans
  `tests/integration/persistence/`, avec Testcontainers.
- **E2E** : flux HTTP des controllers dans `tests/e2e/`, sur PostgreSQL
  (Testcontainers), via `app.NewRouter`.

## Commandes de validation

```text
make test-unit
make test-integration
make test-e2e
make test
make lint
make lint-api
make vuln
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
- Les tests PostgreSQL (intégration et E2E) exigent Docker ou Podman ;
  avec `CI=true` leur indisponibilité fait échouer les tests au lieu de les ignorer.
