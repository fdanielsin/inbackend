# État de reprise du backend

## Mission

Réorganiser `inbackend` selon l'architecture documentée, sans modifier
`infrontend` et sans casser les routes ou réponses HTTP existantes.

## État vérifié

Les packages suivants existent dans le code :

```text
internal/domain/
internal/application/
internal/controllers/
internal/infrastructure/
```

Le CQRS applicatif est présent avec des `commands` et des `queries`. Les
repositories PostgreSQL/GORM sont assemblés dans `main.go`.

Les validations exécutées depuis `inbackend` pour cette remise en conformité
sont :

```text
go test -race ./tests/units/... ./tests/e2e/... ./internal/... → réussi
go vet ./...                                                   → réussi
go build ./...                                                 → réussi
make lint                                                      → réussi
go test ./...                                                   → réussi (6 tests PostgreSQL ignorés sans provider)
go test -count=1 -v ./tests/integration/persistence/...         → réussi (7 tests PostgreSQL ignorés sans provider)
go test -count=1 -v ./tests/e2e/...                             → réussi (8 tests SQLite)
go test -count=1 -v ./tests/e2e/... -run 'E2E'                  → aucun test exécuté (filtre CI corrigé)
DOCKER_HOST=... TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 -v ./tests/integration/persistence/... → 7 tests PostgreSQL réussis avec Podman
```

Les tests d'intégration PostgreSQL sont explicitement ignorés lorsque
Testcontainers ne trouve aucun provider Docker/Podman local. Avec un provider
disponible, ils restent bloquants en cas d'échec. Ces résultats ne constituent
pas une preuve de préparation production tant que PostgreSQL n'a pas été testé.

## Résultats de la remise en conformité

- Une branche dédiée `chore/backend-compliance-audit` est utilisée.
- Les modèles GORM sont maintenant dans
  `internal/infrastructure/persistence/postgres/models.go`; `APIResponse`
  reste dans `internal/shared/response.go`.
- Les comparaisons d'erreurs utilisent `errors.Is`.
- Les contrôleurs ne renvoient plus les messages d'erreur techniques bruts
  pour les erreurs 500; ils journalisent le détail et renvoient un message
  générique.
- La spécification API OpenAPI 3.1 est disponible dans
  `docs/specs/api/openapi.yaml` et couvre les neuf routes enregistrées.
- Les spécifications fonctionnelles, data, non fonctionnelles et de test sont
  indexées dans `docs/specs/`.
- Le logging `slog` délégué à `zap` est conservé comme choix explicite de la
  phase 6.

## Corrections confirmées

- Les jobs CI d'intégration et E2E ciblent désormais leurs packages respectifs.
- Le filtre `-run 'E2E'`, qui ne correspondait à aucun nom de test, a été
  supprimé afin d'éviter une réussite sans exécution.
- `make test` couvre maintenant `./...`, y compris les packages internes.
- Les références README à l'ancien chemin `internal/db` ont été corrigées.

## Contourné temporairement

- Ryuk doit être désactivé avec le socket Podman macOS utilisé localement, car
  Testcontainers tente sinon de monter le socket comme un volume. La commande
  de validation PostgreSQL réussie utilise
  `TESTCONTAINERS_RYUK_DISABLED=true`.

## Points non résolus

- Les anciennes et nouvelles responsabilités doivent être vérifiées pour
  confirmer qu'il ne reste aucun doublon inutile.
- La CI distante reste à vérifier sur la branche publiée; aucun push n'a été
  effectué depuis cet environnement.

Ces points sont des constats d'audit, pas des corrections déjà réalisées.

## Prochaine mission

Réaliser une remise en conformité avant d'ajouter une nouvelle fonctionnalité :

1. publier la branche selon l'autorisation du propriétaire ;
2. vérifier le résultat CI des jobs integration et E2E corrigés ;
3. ne commencer une nouvelle fonctionnalité qu'après cette validation.

## Documentation à lire

Un agent doit commencer par :

1. `../docs/README.md`
2. `../docs/context.md`
3. `../docs/workflow.md`
4. `../docs/documentation-guide.md`
5. `docs/README.md`
6. `docs/architecture.md`
7. `docs/roadmap.md`
8. `docs/decisions/`

## Règles de reprise

- Ne pas considérer un ancien handoff comme une preuve.
- Vérifier les affirmations dans le code, les tests et Git.
- Ne pas commencer une nouvelle fonctionnalité si une remise en conformité
  bloquante reste ouverte, sauf décision explicitement confirmée.
- Documenter les erreurs corrigées, contournées ou non résolues.
- Ne pas créer de nouveaux documents si un document canonique existe déjà.
- Ne pas créer de commit ou pousser sans autorisation explicite.

## Transmission

À la fin d'une tâche, indiquer :

```text
Branche :
Fichiers modifiés :
Corrigé :
Contourné temporairement :
Non résolu :
Tests exécutés :
Build :
Commit :
Push :
CI :
Décisions en attente :
Prochaine étape :
```

## Dernière mise à jour

2026-09-20 — audit de conformité, correction des commandes CI et validation
locale sans provider PostgreSQL.
