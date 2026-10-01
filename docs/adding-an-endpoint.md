# Ajouter un endpoint

Gabarit à suivre dans l'ordre. L'architecture (couches, dépendances, CQRS,
serveur et middlewares) est décrite dans [`architecture.md`](./architecture.md),
les migrations dans l'[ADR 0007](./decisions/0007-versioned-migrations.md) : ce
document ne les répète pas.

## Ordre

1. **Contrat** : `specs/api/openapi.yaml` d'abord (chemin, `operationId`,
   schémas, tous les statuts possibles, codes d'erreur dans l'enum `ErrorCode`).
   `make lint-api` doit passer.
2. **Domaine** : entité et erreurs métier (`internal/domain/<contexte>/`).
3. **Application** : une command ou une query (`internal/application/...`).
4. **Persistance** : méthode du repository ; si le schéma change, une
   migration (`make migrate-new name=...`, jamais de modification d'une
   migration appliquée). Les violations de contrainte sont détectées par code
   SQLSTATE et nom de contrainte (`persistence/pgstore/constraints.go`).
5. **Erreurs** : chaque erreur domaine est ajoutée à l'unique table de
   `internal/shared/errors.go` (statut, `error_code`, clé de message). Tout texte
   de réponse (erreur ou succès) est une clé de `internal/shared/messages.go`,
   avec son texte anglais et français.
6. **Controller** (`internal/controllers/httpapi/`) : `requestLogger(c, ...)`
   pour les logs, `respondWithError` pour toute erreur, jamais le texte d'une
   erreur dans une réponse ni un log (il peut contenir des données).
7. **Route** : une ligne dans `internal/app/router.go`.
8. **Tests** : les 5 catégories (nominal, erreur, limites, valeurs frontières,
   exceptions techniques) ; la réponse de chaque couple opération x statut
   déclaré dans le contrat est validée par le helper de contrat de
   `tests/testsupport`, et un statut déclaré sans test fait échouer la suite.
9. **Documentation** : `CHANGELOG.md` (une ligne), matrice de tests du workspace
   (`docs/specs/test-matrix.md` à la racine d'indrive, hors de ce dépôt),
   `handoff.md` si l'état du lot change.

## Exemple minimal : `GET /ready`

Sonde de disponibilité, sans command ni query (comme `/health`).

- Contrat : `/ready` avec 200 `ReadyResponse` et 503 `DATABASE_UNAVAILABLE`.
- Code d'erreur : `CodeDatabaseUnavailable` dans `shared/errors.go`.
- Controller : `HealthController.Ready` (`health_controller.go`), vérification
  injectée (`ReadinessCheck`) pour pouvoir la remplacer dans les tests.
- Route : `e.GET("/ready", healthCtrl.Ready)` dans `router.go`.
- Tests : base jointe (200), base indisponible (503), contrôle qui dépasse le
  délai de 2 s, récupération, absence de détail technique dans la réponse et
  les logs.
