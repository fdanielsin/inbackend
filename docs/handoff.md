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
repositories PostgreSQL/GORM sont assemblés dans `internal/app` (`app.NewRouter`).

Les validations locales exécutées depuis `inbackend` et vérifiables dans cet
état sont :

```text
make lint                                         → réussi (golangci-lint, gofumpt, goimports)
make test                                         → réussi ; 7 tests PostgreSQL ignorés sans provider
go test ./...                                     → réussi
go vet ./...                                      → réussi
go build ./...                                    → réussi
go test -count=1 -v ./tests/integration/persistence/... → réussi ; 7 tests ignorés
go test -count=1 -v ./tests/e2e/...               → réussi ; 8 tests exécutés
TESTCONTAINERS_RYUK_DISABLED=true DOCKER_HOST=... go test -count=1 -v ./tests/integration/persistence/... → réussi ; 7 tests PostgreSQL exécutés
TESTCONTAINERS_RYUK_DISABLED=true DOCKER_HOST=... go test -count=1 ./... → réussi avec PostgreSQL actif
```

Les tests d'intégration PostgreSQL sont explicitement ignorés lorsque
Testcontainers ne trouve aucun provider Docker/Podman local. Avec un provider
disponible, ils restent bloquants en cas d'échec. La validation avec Podman
ci-dessus constitue une preuve locale d'exécution PostgreSQL, sans constituer
à elle seule une preuve de préparation production.

Une première exécution sans socket explicite a produit
`rootless Docker not found, failed to create Docker provider`. La validation
réussie a ensuite utilisé le socket Podman macOS explicite et
`TESTCONTAINERS_RYUK_DISABLED=true`.

## Résultats de la remise en conformité

- Une branche dédiée `chore/backend-compliance-audit` est utilisée.
- Les modèles GORM sont maintenant dans
  `internal/infrastructure/persistence/pgstore/models.go`; `APIResponse`
  reste dans `internal/shared/response.go`.
- Les comparaisons d'erreurs utilisent `errors.Is`.
- Les contrôleurs ne renvoient plus les messages d'erreur techniques bruts
  pour les erreurs 500; ils journalisent le détail et renvoient un message
  générique.
- La spécification API OpenAPI 3.1 est disponible dans
  `docs/specs/api/openapi.yaml` et couvre les neuf routes enregistrées.
- Les spécifications fonctionnelles, data, non fonctionnelles et de test sont
  indexées dans `docs/specs/`.
- Le logging `slog` délégué à `zap` a été conservé en phase 6, puis remplacé
  par `slog` seul au lot E1b.

## Correctif Q-12 — Ordre des tâches non garanti

`TaskRepository.ListByUserID` (`internal/infrastructure/persistence/pgstore/task_repository.go`)
ne triait pas explicitement ses résultats ; l'ordre retourné par
`GET /api/v1/users/{id}/tasks` n'était pas garanti par SQL. Ajout de
`.Order("id ASC")` ; contrat `openapi.yaml` mis à jour en premier pour
documenter le tri par `id` croissant. Décision utilisateur : ce lot ne
touche que les tâches. `UserRepository.List` (même absence d'`ORDER BY`)
n'est pas traité ici — c'est une question fonctionnelle distincte, non
tranchée, à traiter séparément si besoin.

## Correctif Q-01 — PUT /api/v1/tasks/{id} n'appliquait pas les valeurs falsy/vides

- **Bug** : `UpdateTaskRequest` utilisait des champs Go non-pointeurs ; une clé
  JSON absente était indiscernable d'une valeur explicite à zéro. Combiné à
  `TaskRepository.Update`, qui passait un `struct` à `Updates()` (GORM ignore
  silencieusement les champs à valeur zéro dans ce cas), un `PUT`
  `{"completed":false}` ou `{"description":""}` ne persistait jamais le
  changement, alors que la réponse HTTP (construite en mémoire) affichait la
  valeur demandée comme si elle avait été enregistrée.
- **Correction** : `UpdateTaskRequest` utilise désormais des pointeurs
  (`*string`/`*bool`) ; seul un champ non-nil est appliqué. `title` explicite
  vide renvoie `400` (`task.ErrTaskTitleEmpty`). `TaskRepository.Update`
  utilise `Select("title", "description", "completed").Updates(...)` (jamais
  `Save`, pour exclure tout risque d'upsert recréant une tâche supprimée
  entre-temps) et renvoie `task.ErrTaskNotFound` (404) si `RowsAffected == 0`.
- **Contrat** : `docs/specs/api/openapi.yaml` documente explicitement la
  sémantique de mise à jour partielle de `PUT /api/v1/tasks/{id}` (champ
  absent = non modifié) et ajoute `minLength: 1` sur `title`. La route reste
  `PUT` (option retenue plutôt qu'un nouvel endpoint `PATCH`).
- **Tests** : unitaires application dans
  `tests/units/application/update_task_test.go` (logique de fusion/validation,
  sans DB) ; régression e2e et intégration PostgreSQL ajoutées par `qa`.

## Nettoyage backend (demande utilisateur, hors plan d'audit)

Branche `chore/cleanup-backend` depuis `develop` (`d2fede3`). Objectif :
code écrit comme par un développeur senior (ADR 0001), sans changement de
comportement ni de contrat.

- **Supprimé** : `internal/infrastructure/database/testdb.go` (`Q-10`) — zéro
  appelant vérifié par grep croisé dans tout le dépôt (y compris les tests) ;
  liait `testcontainers-go` dans le binaire de production sans raison, car ce
  n'était pas un fichier `_test.go`. `go mod tidy` exécuté après coup : aucun
  changement de `go.mod`/`go.sum` (`testcontainers-go` reste requis par de
  vrais fichiers de test ailleurs).
- **Supprimé** : `GetZap()`/`ZapLog` (`internal/infrastructure/logger/logger.go`)
  et `zapLevelForSlog()` (`slog_handler.go`) — zéro appelant vérifié.
  `InitLogger()`, `GetSlog()`/`SlogLog` conservés alors (`GetSlog`/`SlogLog` supprimés en E1, voir « Lot E1 »).
- **Supprimé** : `Delete` de l'interface `user.Repository` et de son
  implémentation PostgreSQL — zéro appelant, aucune route `DELETE /users`,
  absent du tableau CQRS d'`architecture.md`.
- **Retiré** : une quinzaine de commentaires qui paraphrasaient la ligne
  suivante, dans `create_user.go`, `create_task.go`, `update_task.go`,
  `get_stats.go`, `user_repository.go`, `task_repository.go`. Conservés : tous
  les commentaires godoc sur symboles exportés, et les blocs expliquant une
  décision non évidente (dont ceux du lot Q-01).
- **Corrigé** : `.env.example` ne liste plus que les variables réellement
  lues par le code (`DATABASE_URL`, `PORT`, `ENV`, vérifié par grep sur
  `os.Getenv`) ; `DATABASE_URL` ajoutée (obligatoire, absente du template
  précédent) ; `DB_USER`/`DB_PASSWORD`/`DB_NAME`/`DB_HOST`/`DB_PORT`/
  `CORS_ORIGIN`/`LOG_LEVEL` retirées (jamais lues nulle part).
- **Corrigé** : `architecture.md`, `infrastructure/database/` ajouté à
  l'arborescence documentée (existe dans le code, n'était pas documenté).
- **Écart connu, conservé volontairement comme cible** (décision utilisateur ;
  règle ajoutée à AGENTS.md : une section « cible » se garde, annotée) :
  `architecture.md` documente aussi `domain/user/service.go`,
  `domain/task/service.go`, `infrastructure/http/`, `infrastructure/config/`
  et `shared/errors/`/`shared/pagination/`/`shared/response/` comme des
  sous-dossiers, alors qu'aucun n'existe dans le code (`shared` a deux
  fichiers plats `errors.go`/`response.go`, pas de `pagination`). Chacun
  annoté « (cible, pas encore créé) » dans l'arbre. La section
  « Infrastructure » a été réécrite en cohérence : Echo et le router ne sont
  pas dans `infrastructure/` (Echo est utilisé par les controllers et
  `main.go:56-75`) ; l'absence de package `config/` dédié est décrite comme un
  fait, avec `config/`/`http/` comme cible.
- **Corrigé** : ADR 0006 suite à la revue `qa` — le schéma de dépendances
  (`infrastructure ──> application` était faux, corrigé en
  `infrastructure ──> domain`), le tableau CQRS (`GetTaskStats` inexistant →
  `GetStats` ; `HealthCheck` est un controller, pas une query), la case
  « Pre-commit hook » (hooks `.githooks/` existants, activation locale non
  automatique), l'étape « Security » (`nancy sleuth` échoue en 401 sur OSS
  Index à chaque run, masqué par `\|\| true`, ne bloque jamais — vérifié sur
  le run `36637119785`), la description de l'intégration CI (démarre son
  propre conteneur Testcontainers, ignore `DATABASE_URL` et le service
  `postgres` du runner), la durée du pipeline (`~10-11 min` mesurées,
  remplace `~4-5 min`/`3-4 min`), et la case « Commit convention enforced »
  (aucun contrôle ne l'impose, 7 commits existants sur `origin/develop`
  contiennent `Co-authored-by`, réécriture d'historique prévue). Les règles
  « No merge commits » elles-mêmes ne sont pas modifiées (choix utilisateur de
  les suivre) ; aucune assertion factuelle « `develop` est linéaire » n'a été
  trouvée ailleurs dans l'ADR ou ce handoff nécessitant une note sur les
  merges `b28ee65`/`d2fede3`.
- **`Dockerfile` et `docker-compose.yml`** : traités par le lot Go 1.26 +
  Docker (`Q-03`), voir « Go 1.26 et Docker ».
- **`CHANGELOG.md`** réécrit au format Keep a Changelog 1.1.0 (`[Unreleased]`
  + `[1.0.0] - 2026-09-17`, rubriques Added/Changed/Fixed/Removed, une ligne
  par changement notable) suite à la nouvelle règle ajoutée à AGENTS.md.
- **Vérifications** : `make lint`, `go vet ./...`, `go build ./...` PASS ;
  `go test ./...` PASS (y compris `tests/integration/persistence` réel via
  Docker) avant et après toutes ces modifications ; aucun test existant
  modifié. Cette série de corrections est uniquement documentaire (aucun
  fichier Go touché), revérification `go test` non nécessaire.
- **Vérification `qa`** : verdict initial « bloqué, uniquement sur la doc »
  (code accepté : même comportement HTTP sur 25 requêtes, mêmes 41 tests avant
  et après, aucun appelant des symboles supprimés) ; corrections ci-dessus
  faites en réponse, `qa` prévenue pour re-vérification.

## Outillage lint (branche `chore/lint-tooling`)

- **Ajouté** : `.golangci.yml` (errcheck, govet, staticcheck, unused,
  ineffassign, errorlint, bodyclose, noctx, gosec, revive réduit, gocritic
  diagnostic ; formatters gofumpt et goimports), `docs/specs/api/redocly.yaml`,
  cibles `make lint-api` et `make vuln`. `make lint`, `make fmt` et la CI
  utilisent les mêmes commandes ; Redocly et govulncheck sont bloquants en CI
  (`Q-05` : govulncheck remplace `nancy`, qui ne vérifiait rien).
- **Pourquoi `go run ...@version` et pas `go get -tool`** : la dépendance
  d'outil faisait passer `go.mod` à `go 1.26.0` et montait des dépendances de
  production par résolution MVS (`x/crypto`, `x/net`, `protobuf`, `otel`) ;
  `go.sum` reste donc inchangé. Tous les jobs CI utilisent la version de
  `go.mod` (lot Go 1.26 + Docker).
- **Exclusions justifiées dans la config** : `misspell` non activé (faux
  positifs sur le français), `noctx` limité à `httptest.NewRequest` dans les
  `_test.go` (les ids uint sont journalisés avec `slog.Uint64`, sans exclusion
  gosec `G115`), staticcheck `SA1019` sur `middleware.Logger` d'Echo
  (exclusion retirée au lot 6 : `RequestLoggerWithConfig`). Redocly (`recommended-strict`, pour que
  `operation-4xx-response` soit une erreur bloquante) : `security-defined` (pas d'authentification, décision produit),
  `info-license` (dépôt privé), `no-server-example.com` (serveur localhost du
  contrat) ; `operation-4xx-response` reste active, avec 4 opérations ignorées
  dans `docs/specs/api/.redocly.lint-ignore.yaml` (`/health`, `/ready`, `GET /users`,
  `GET /stats` : aucune 4xx possible).
- **Code** : aucun changement de comportement. `main.go` : la logique passe
  dans `run` qui renvoie une erreur, `main` appelle `log.Fatal` (zap vidait ses
  buffers avant de quitter ; remplacé par `log.Error` et `os.Exit(1)` en E1b) ; le message de
  log d'échec au démarrage change de libellé, le code de sortie reste 1.
  Paramètres inutilisés renommés `_`, commentaires de package et godoc
  ajoutés. Les 8 corrections dans `tests/` ont été faites par `qa`.
- **Contrat** : descriptions de tags ajoutées, schéma inutilisé
  `MessageResponse` supprimé ; aucun code HTTP ajouté, ceux déjà documentés
  correspondent aux controllers.
- **Non couvert** : les hooks `.githooks/` restent sur `gofmt` et `go vet`
  (contrôle rapide, sous-ensemble de `make lint`).

## Go 1.26 et Docker (Q-03)

Branche `fix/Q03-go126-docker`.

- **Go** : `go 1.26.0` + `toolchain go1.26.8` dans `go.mod`, version unique pour
  tous les jobs CI (`go-version-file: go.mod`) et le Dockerfile.
  Motif sécurité : govulncheck sur le build en Go 1.25.0 signale 25
  vulnérabilités stdlib atteignables ; 0 avec 1.26.8 (`go 1.26.0` seul en
  laisserait 16, d'où la ligne `toolchain`). `go.sum` inchangé.
- **Image** : `golang:1.26.8-alpine` puis `alpine:3.24.2`, `CGO_ENABLED=0`,
  `-trimpath -ldflags="-s -w"`, utilisateur non root (uid 10001), ~36 MB.
- **Compose** : PostgreSQL 16 avec healthcheck, `DATABASE_URL` construite avec
  des identifiants de développement, backend derrière `service_healthy`, healthcheck
  HTTP sur `/health` (sur `/ready` depuis le lot 7). `.dockerignore` versionné (retiré du `.gitignore`).
- **CI** : `codecov-action` passé de v3 à v5 (actionlint signale le runtime de v3
  comme trop ancien). La panne de démarrage de la CI sur inbackend#8 n'est PAS due à
  codecov (réfuté : échec persistant en v5) ; diagnostic en cours côté
  compte/GitHub. `make lint-ci` (actionlint v1.7.7) dans le
  job Lint détecte ce type d'action.
- **CI** : job `docker` bloquant (build, `compose up -d --wait`, `/health`
  (puis `/ready` depuis le lot 7), logs si échec, `down -v`).
- **Non couvert** : Podman/`make docker-run` non testés ; le job CI `docker`
  n'a pas encore tourné sur GitHub (vérifié seulement en local avec Docker).

## Racine de composition `internal/app` (QP-01)

Branche `test/lot3-backend` (worktree `inbackend/.worktrees/lot3-backend`).

- Le câblage repositories, handlers, controllers, middlewares (Logger, Recover,
  CORS) et les 9 routes quitte `main.go` pour `app.NewRouter(db *gorm.DB) *echo.Echo`
  (`internal/app/router.go`). `main.go` garde logger, configuration, `InitDB`,
  `app.NewRouter(db)` et `Start`.
- Refactoring sans changement de comportement. Décision utilisateur du
  2026-09-30 : les E2E testent l'assemblage réel ; ADR 0005 amendé.

## SQLite retiré des tests (Q-06)

- Les E2E passent par `app.NewRouter` sur PostgreSQL (Testcontainers,
  `tests/testsupport/postgres.go`). `gorm.io/driver/sqlite` et `go-sqlite3`
  sont retirés de `go.mod` par `go mod tidy` ; les tests tournent avec
  `CGO_ENABLED=0`.
- `make test-e2e` exige Docker ou Podman. Sans provider, les tests PostgreSQL
  sont ignorés en local avec leur cause ; avec `CI=true` ils échouent (Q-09).
- Job CI : le service `postgres` et `DATABASE_URL`, inutilisés, sont retirés
  des jobs integration-tests, e2e-tests et coverage.
- Dépendances : `x/crypto` v0.56.0 et `moby/go-archive` v0.3.0 corrigent 3
  vulnérabilités que govulncheck atteint via `tests/testsupport` (`make vuln` :
  0). Montent aussi, par résolution : `x/net` v0.57.0, `x/text` v0.41.0,
  `klauspost/compress` v1.18.7, `moby/sys/user` v0.4.1. `x/crypto`, `x/net` et
  `x/text` sont aussi liés au binaire de production (transitif).
- Base du lot : `test/lot3-backend` est empilée sur `fix/Q03-go126-docker`
  (Go 1.26.8, requis par `x/crypto` v0.56.0).
- Docs alignées : ADR 0004, 0005, 0006, strategy.md, users-and-tasks.md,
  README, lignes `*.sqlite` du `.gitignore` retirées.

## Corrections confirmées

- Les jobs CI d'intégration et E2E ciblent désormais leurs packages respectifs.
- Le filtre `-run 'E2E'`, qui ne correspondait à aucun nom de test, a été
  supprimé afin d'éviter une réussite sans exécution.
- `make test` couvre maintenant `./...`, y compris les packages internes.
- Les références README à l'ancien chemin `internal/db` ont été corrigées.

## Contournement documenté

- Sans provider, les tests PostgreSQL sont ignorés explicitement.
- Dans l'environnement validé, le socket Podman macOS doit être fourni via
  `DOCKER_HOST` et `TESTCONTAINERS_RYUK_DISABLED=true` doit être activé.

## Validation CI distante

- L'API GitHub confirme que le run `35479583554` est `completed/success` pour
  `chore/backend-compliance-audit`, commit `5a76cb3`.
- Le run le plus récent observé est `35480322121`, également
  `completed/success`, pour le commit `a44832c` actuellement en tête de la
  branche.
- Ces résultats valident les commits distants concernés ; le diff documentaire
  local actuel reste non couvert par ces runs tant qu'il n'est pas committé.

## Audit du workspace et de Git

- Les seuls éléments à la racine sont `docs/`, `inbackend/` et `infrontend/`.
  `README.md`, `scripts/`, `docker-compose.yml` et `.github/` sont absents de
  la racine comme prévu.
- `inbackend` et `infrontend` sont deux dépôts Git indépendants avec leurs
  propres README et fichiers de projet. Aucun fichier frontend n'a été modifié.
- Le backend est sur `chore/backend-compliance-audit`; les modifications
  documentaires de cette reprise restent non committées.

## Classification de l'état

- **Corrigé et vérifié** : structure des couches, routes enregistrées,
  documentation indexée, formatage, vet, build, tests unitaires, E2E et
  PostgreSQL avec Podman, ainsi que les deux derniers runs CI distants observés.
- **Contournement documenté** : détection automatique du socket Podman ;
  la commande locale doit fournir `DOCKER_HOST` explicitement.
- **Non résolu** : les limites documentées de relation GORM, migrations
  versionnées, authentification et autorisation restent hors de cette mission.
- **Hors périmètre** : nouvelle fonctionnalité, modification du frontend,
  création de fichiers à la racine, commit ou push.

## Lot 4 : erreurs et contrat (branche `feat/lot4-errors-contrat`)

- Toute erreur porte un `error_code` stable (enum dans `openapi.yaml`) ; une
  seule table erreur -> statut/code/message dans `internal/shared/errors.go`.
- `B-03` : email en double -> 409 `EMAIL_TAKEN` (code PostgreSQL 23505). Email
  normalisé dans `NewUser` (trim, minuscules) et format minimal validé (400).
  Les doublons par casse déjà en base relèvent du lot des migrations.
- `B-04` : DELETE d'une tâche absente -> 404 `TASK_NOT_FOUND`.
- Stats : `COUNT` SQL (`user.Repository.Count`).
- `B-08` : aucun email, nom ni texte brut d'erreur dans les logs ; logger GORM
  `Silent` ; les repositories ne journalisent plus, les controllers journalisent
  `error_code` et l'id. Logger injecté dans les tests par `slog.SetDefault`.
- Tests de contrat : kin-openapi v0.149.0, helper `tests/testsupport/contract.go`.
- `make mutate` : mutation (gremlins) sur le diff, hors CI.

## Lot 5 : migrations (branche `feat/lot5-migrations`)

- golang-migrate (ADR 0007) : SQL embarqué dans
  `internal/infrastructure/database/migrations/`, appliqué au démarrage par
  `database.Migrate` ; `database.MigrateTo(dsn, version)` sert aux tests ;
  `make migrate-up/down/force/new`. `AutoMigrate` retiré.
- 000001 = schéma de l'ancien AutoMigrate (tables `user_models`/`task_models`,
  comparé par `pg_dump` : identique hors `schema_migrations`). 000002 = emails
  (refuse sur doublons par casse, normalise, index unique sur `lower(email)`).
  000003 = clé étrangère et index sur `task_models.user_id` (refuse sur
  orphelins). Les refus ne donnent que des nombres.
- `DATABASE_URL` : URL `postgres://` obligatoire (format `clé=valeur` refusé au
  démarrage : « DATABASE_URL must be a postgres:// URL », valeur jamais affichée).
- Création de tâche pour un utilisateur inexistant : 404 `USER_NOT_FOUND`
  (violation 23503 de `fk_task_models_user`, sans SELECT préalable).
- Un échec laisse la base « dirty » : `make migrate-force version=N` (N = dernière
  version entièrement appliquée), indiqué dans l'erreur.

## Lot 6 : robustesse et outillage (branche `chore/lot6-outillage`)

- `http.Server` avec délais, arrêt gracieux, pool SQL 25/5/30 min, request id
  maison, `AccessLog` (`RequestLoggerWithConfig`, plus d'exclusion SA1019),
  CORS par `CORS_ALLOWED_ORIGINS`, `BodyLimit` maison de 1 MiB exacts
  (`http.MaxBytesReader`, y compris en chunked), `NewErrorHandler` (erreurs d'Echo dans l'enveloppe).
  Détails dans `architecture.md`, « Serveur HTTP et middlewares ».
- `Q-04` : `make cover` (minimum `COVERAGE_MIN`, 85 %), étape bloquante du job
  coverage ; le package `logger` n'est couvert que par le binaire des tests
  e2e, non instrumenté, donc compté à 0 % (pas d'exclusion).
- `Q-11` : `.githooks/pre-push` (`make lint`, build, tests unitaires) ; activer
  avec `git config core.hooksPath .githooks`.

## Lot 7 : sondes et gabarit (branche `chore/lot7-health`)

- `GET /ready` (ping de la base, délai 2 s, 503 `DATABASE_UNAVAILABLE` sans
  détail) ; `/health` toujours sans base ; `Cache-Control: no-store` sur les
  deux ; sondes réussies journalisées en Debug (`probeRoutes`).
- Compose : healthcheck sur `/ready` (Docker ne redémarre pas un conteneur
  « unhealthy »). Job `docker` : `/health` puis `/ready`.
- Gabarit : `docs/adding-an-endpoint.md`.

## Lot E1 : conformité Go (branche `refactor/e1-go-style`)

Sans changement de comportement HTTP ni de contrat.

- `InitLogger` ; `SlogLog` et `GetSlog` supprimés (signature finale en E1b,
  ci-dessous).
- Logger injecté : `app.WithLogger` (nil garde `slog.Default()` lu à
  `NewRouter`) ; constructeurs de controllers `New…Controller(log, …)`.
- `InitDB` garde sa signature mais ne journalise plus ses échecs : erreurs
  enveloppées (`migrate database`, `open database`, `access SQL connection
  pool`), journalisées une fois par `main`.
- `interface{}` → `any`, shadow corrigé dans `UpdateTask`, commentaires
  paraphrasés ou périmés retirés.
- Suite prévue (un plan par lot) : E2 code mort, E3 renommages, E4 tests,
  règles de lint (dont `shadow`, qui signale aussi `tests/testsupport/contract.go:46`).

## Lot slog : slog seul (branche `chore/replace-zap-with-slog`)

Décision utilisateur : zap et `ZapHandler` supprimés, un seul logger `*slog.Logger`
injecté (`main`, `InitDB`, `app.WithLogger`, controllers).

- `logger.New(w io.Writer, env string)` : JSON au niveau Info si
  `env == "production"`, texte au niveau Debug sinon, `AddSource` dans les deux
  cas. `InitLogger() *slog.Logger` l'appelle sur stderr avec `ENV` et fait
  `slog.SetDefault` ; il ne retourne plus d'erreur.
- `main` : `log.Error("Server failed", ...)` puis `os.Exit(1)`.
- `InitDB(dsn, log *slog.Logger)`.
- Format visible : voir CHANGELOG (Changed). Pas de pile d'appels sur les
  erreurs ; durée `latency` en nanosecondes en JSON.

## Lot code mort (branche `chore/remove-dead-code`)

Retrait, sans appelant en production (mesuré avec `deadcode`) : `MapErrorToMessage`,
`IsNotFoundError`, `IsValidationError` (`shared`), `Task.CompleteTask` et
`Task.UncompleteTask`, ainsi que `internal/shared/errors_test.go` ; son seul
cas encore vivant (`Classify` sur une erreur combinée par `errors.Join`) est
porté dans `tests/units/shared/errors_test.go`. `database.MigrateTo` est gardée : seuls les
tests de migration l'appellent, son godoc le dit.

## Lot README (branche `chore/slim-readme-essentials`)

README réduit à l'essentiel (projet, prérequis, démarrage, variables, commandes,
liens). Déploiement et dépannage dans `docs/deployment.md` et
`docs/troubleshooting.md` ; la mention de licence MIT est retirée (aucun fichier
`LICENSE`, décision utilisateur).

## Lot Echo v5 (branche `chore/upgrade-echo-v5`)

Echo v4.15.4 → v5.4.0, sans changement visible : mêmes statuts, corps, en-têtes
et lignes de log d'accès (comparés sur 23 requêtes avant et après).
`echo.Context` devient `*echo.Context` ; le gestionnaire d'erreurs prend
`(c, err)` ; `Committed` se lit par `echo.UnwrapResponse`. Piège v5 : les
erreurs 404/405/413 d'Echo sont un type non exporté, reconnues par l'interface
`echo.HTTPStatusCoder` (sinon elles sortent en 500). `e.Logger` est notre logger.
`govulncheck` : 0 vulnérabilité atteignable ; `GO-2026-5932` (`x/crypto`, sans
correctif) est dans un module requis mais jamais appelé.

## Lot messages localisés (branche `feature/localized-api-messages`)

`Accept-Language` : français pour `fr` et ses variantes, anglais sinon, `q` le
plus haut l'emporte, jamais de 400 (`shared.NegotiateLanguage`, qui rejette en
plus ce que x/text tolère : `q` supérieur à 1, tirets bas). Catalogue
`shared/messages.go` (23 clés), `APIError` porte une clé, middleware
`httpapi.Language`. Contrat en deux commits : les en-têtes de réponse déclarés
sont validés par les tests de contrat, ils arrivent donc avec le code.

## Prochaine mission

La conformité locale est vérifiée pour les commandes exécutées ci-dessus,
y compris PostgreSQL avec Podman. Une nouvelle décision n'est nécessaire que
si l'architecture ou le workflow change.

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

2026-10-01 : lot 7 sondes (voir « Lot 7 »), lot 6 outillage (voir « Lot 6 »), lot 5 migrations (voir « Lot 5 »), après le lot 4 erreurs et contrat.
2026-09-30 — outillage lint sur `chore/lint-tooling` (golangci-lint, govulncheck,
Redocly), sans changement de comportement ; voir « Outillage lint » ci-dessus.
Le nettoyage backend précédent est détaillé dans « Nettoyage backend ».
