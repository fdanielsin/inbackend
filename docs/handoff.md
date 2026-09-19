# État de reprise pour les agents

## Mission

Réorganiser uniquement le backend Go dans `inbackend` selon l'architecture
documentée, sans casser l'API existante et sans modifier `infrontend`.

## Décisions déjà prises

- Architecture en `domain`, `application`, `controllers`, `infrastructure`.
- CQRS applicatif avec `commands` et `queries`.
- Une seule base PostgreSQL.
- Repositories définis côté domaine et implémentés côté infrastructure.
- `shared` autorisé uniquement pour les préoccupations réellement transversales.
- `service.go` ajouté uniquement lorsqu'une vraie règle métier le justifie.
- Pas d'Event Sourcing.
- Pas de Saga.
- Pas d'Event Store.
- Pas de transactions distribuées.
- Pas de Domain Events sans consommateur concret.
- Logging slog avec délégation vers zap.

## État du code — après Phase 6

✅ **Phase 1 complétée** : Fondations du domaine User et Task

✅ **Phase 2 complétée** : Application layer avec CQRS

✅ **Phase 3 complétée** : Infrastructure PostgreSQL/GORM

✅ **Phase 4 complétée** : Controllers HTTP et Dependency Injection

✅ **Phase 5 complétée** : Validation et Tests End-to-End

✅ **Phase 6 complétée** : Restructuration Architecture + Logging slog

### Phase 6 : Changements majeurs

1. **Restructuration `internal/db` → `internal/infrastructure/database/`**
   - ✅ Aligne avec Clean Architecture hexagonale
   - ✅ InitDB() et testdb.go déplacés dans l'infrastructure
   - ✅ main.go importé depuis `internal/infrastructure/database`
   - ✅ Ancien `internal/db/` supprimé

2. **Centralisation des tests**
   - ✅ Tests unitaires restent dans `internal/` (convention Go)
   - ✅ Tests d'intégration/E2E centralisés dans `tests/`
   - ✅ Ancien `e2e_test.go` supprimé (remplacé par controller_integration_test.go)
   - ✅ 38 tests totaux passants

3. **Intégration slog (stdlib Go 1.21+)**
   - ✅ `internal/logger/slog_handler.go` — Handler slog qui délègue vers zap
   - ✅ `internal/logger/logger.go` — Initialisation dual slog/zap
   - ✅ Logs structurés ajoutés aux 4 controllers HTTP
   - ✅ Logs structurés ajoutés au UserRepository
   - ✅ Aucune dépendance externe supplémentaire
   - ✅ slog utilise zap en arrière-plan pour la performance

### Fichiers créés/modifiés en Phase 6

**Créés:**
- `internal/logger/slog_handler.go` — Handler slog custom (74 lignes)
- `internal/infrastructure/database/init.go` (ex `internal/db/db.go`)
- `internal/infrastructure/database/testdb.go` (ex `internal/db/testdb.go`)

**Modifiés:**
- `internal/logger/logger.go` — InitLogger() retourne zap + initialise slog
- `main.go` — Import depuis `internal/infrastructure/database`
- `internal/controllers/http/user_controller.go` — Logs slog ajoutés
- `internal/controllers/http/task_controller.go` — Logs slog ajoutés
- `internal/controllers/http/stats_controller.go` — Logs slog ajoutés
- `internal/controllers/http/health_controller.go` — Logs slog ajoutés
- `internal/infrastructure/persistence/postgres/user_repository.go` — Logs slog ajoutés

**Supprimés:**
- `internal/db/` (déplacé vers infrastructure/database)
- `tests/e2e_test.go` (obsolète)

### Arborescence architecture

```
internal/
├── domain/
│   ├── user/ (entity, errors, repository interface)
│   └── task/ (entity, errors, repository interface)
├── application/
│   ├── user/ (commands, queries)
│   └── task/ (commands, queries)
├── controllers/
│   └── http/ (4 controllers + logging slog)
├── infrastructure/
│   ├── database/ (InitDB, testdb)
│   ├── persistence/
│   │   └── postgres/ (repositories GORM + logging slog)
├── logger/ (zap + slog handler)
├── shared/ (errors, responses)
├── models/ (GORM entities)
├── handlers/ (legacy, unused)
└── ...

tests/
├── controller_integration_test.go (9+ E2E tests)
└── *_test.go (consolidated integration tests)
```

### Status Phase 6
- ✅ Architecture directory complète et cohérente
- ✅ `internal/db` restructuré dans `infrastructure/database`
- ✅ Tests organisés: unitaires dans `internal/`, intégration dans `tests/`
- ✅ slog intégré avec delegation vers zap
- ✅ Logs structurés à travers controllers et repositories
- ✅ Build sans erreur
- ✅ Code formaté (gofmt)
- ✅ 38+ tests passants (domain 9 + app 13 + infra 6 + integration 9 + E2E 1)
- ✅ Routes API inchangées (9 routes)
- ✅ Formats de réponse identiques (APIResponse)
- ✅ Codes HTTP préservés
- ✅ Architecture entièrement validée

## État du code au démarrage initial

Le backend actuel contenait notamment :

- `README.md` et `CHANGELOG.md` à la racine du projet ;
- `internal/models/models.go` ;
- `internal/handlers/handlers.go` ;
- `internal/db/db.go` (❌ SUPPRIMÉ, migré vers infrastructure/database)
- `internal/logger/logger.go` (✅ AMÉLIORÉ avec slog)
- `main.go`.

Les handlers utilisaient encore directement GORM. La migration est complète.

La documentation détaillée est dans `docs/`. Les fichiers opérationnels
présents à la racine restent dans leur rôle et ne sont pas recopiés dans ce
dossier.

## Prochaines étapes optionnelles

- Ajouter des logs slog au TaskRepository et autres repositories
- Intégration avec Docker Compose et Testcontainers pour les tests E2E réels
- Performance profiling et optimisation (GetStats utilise COUNT au lieu de fetch all)
- Caching et pagination avancée
- Audit logging et tracing distribué
- GraphQL API en parallèle
- Rate limiting et authentification JWT

## Contraintes de travail

- Ne pas modifier des changements non liés.
- Préserver les routes et les réponses existantes autant que possible.
- Utiliser les tests existants comme filet de sécurité.
- Documenter toute nouvelle décision importante dans `docs/decisions/`.
- Ne pas ajouter de technologie ou de dépendance sans besoin démontré.
- Respecter le [guide de gouvernance documentaire](../../docs/documentation-guide.md).
- Ne jamais créer de fichiers `v1`, `v2`, `final` ou de copies concurrentes.
- Pour les problèmes découverts, suivre la classification du
  [workflow global](../../docs/workflow.md) et ne pas élargir silencieusement le
  périmètre.
- Travailler sur une branche dédiée et suivre la procédure Git et CI du
  [workflow global](../../docs/workflow.md).
- Ne déclarer un push ou une CI réussie qu'après vérification effective.

## Contrôle avant transmission

Avant de terminer une tâche, l'agent doit relire son travail et déclarer dans
son rapport :

- ce qui a été corrigé ;
- ce qui a été contourné temporairement ;
- ce qui reste non résolu ;
- ce qui est hors périmètre ;
- les commandes de validation réussies ou échouées ;
- la prochaine étape unique pour l'agent suivant.

Un contournement doit rester visible et ne doit pas être présenté comme une
correction définitive. Le handoff doit refléter l'état réel du code, même si
cela signifie reconnaître une erreur ou une validation impossible.
