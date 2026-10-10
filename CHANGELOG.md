# Changelog

Format basé sur [Keep a Changelog 1.1.0](https://keepachangelog.com/fr/1.1.0/).

## [Unreleased]

### Added

- Toute réponse de l'API porte `X-Content-Type-Options: nosniff`, erreurs, préflight CORS et 413 compris (avant, seulement `/docs` et `/openapi.yaml`) (inbackend#42).
- `GET /api/v1/users/{id}/tasks?completed=true|false` ne garde que les tâches terminées ou non ; sans paramètre, toutes ; toute autre valeur donne 400 `VALIDATION_FAILED`.

### Changed

- Le contrat déclare que `PUT /api/v1/tasks/{id}` répond aussi 400 `INVALID_ID` (comportement inchangé), et les tests refusent désormais un code d'erreur que la description de la réponse ne nomme pas (inbackend#42).
- Création d'utilisateur : e-mail refusé (400 `VALIDATION_FAILED`) si la partie locale dépasse 64 caractères, commence ou finit par un point ou contient `..`, si un label dépasse 63, s'il contient `_` ou un tiret au bord, ou si le TLD n'a pas 2 à 24 lettres (ou la forme `xn--`) ; `a@b.c` n'est plus accepté. Les utilisateurs existants ne changent pas.
- La CI ne se lance plus au push des branches de travail mais sur leur pull request, ce qui évite les runs en double ([#39](https://github.com/fdanielsin/inbackend/pull/39)).

## [1.2.0] - 2026-10-06

> **À vérifier avant de mettre à jour (de 1.1.0 vers 1.2.0).**
>
> - `CORS_ALLOWED_ORIGINS` : une origine mal formée empêche le démarrage, avec le rang de l'entrée et la raison dans l'erreur (schéma autre que `http://` ou `https://`, identifiants, chemin, slash final, query, fragment, majuscules, port par défaut, motif `https://*.example.com`, hôte numérique qui n'est pas une IPv4 canonique, IPv6 mappée IPv4, `*` mêlé à d'autres origines) ; vérifier chaque entrée avant de déployer.

### Changed

- `GET /health` renvoie `version` 1.2.0 et le contrat déclare `info.version` 1.2.0.
- `CORS_ALLOWED_ORIGINS` est validée au démarrage, avant `DATABASE_URL` : chaque origine doit être `http(s)://hôte[:port]` en minuscules, le refus donne le rang de l'entrée et la raison sans recopier la valeur ([#35](https://github.com/fdanielsin/inbackend/pull/35)).

## [1.1.0] - 2026-10-04

> **À vérifier avant de mettre à jour (de 1.0.0 vers 1.1.0).** Changements qui
> peuvent casser un déploiement ou un client existant :
>
> - `ENV=production` : le serveur refuse de démarrer si `CORS_ALLOWED_ORIGINS` est absente, vide ou contient `*` ; lister les origines.
> - `ENV=production` : `/openapi.yaml` et `/docs` répondent 404 sauf `API_DOCS_ENABLED=true`.
> - Logs : `slog` seul ; en production, JSON avec `time`, `level` en majuscules, objet `source`, `latency` en nanosecondes ; adapter les parseurs et alertes de logs.
> - Base de données de la 1.0.0 : la 1.0.0 créait les tables `users` et `tasks`, la 1.1.0 lit `user_models` et `task_models`. Le serveur refuse de démarrer (code 1, « database has 1.0.0 tables ») tant que `users` ou `tasks` existe ; rien n'est copié ni supprimé automatiquement. Sauvegarder la base, puis suivre la procédure SQL de `docs/troubleshooting.md` (renommer les anciennes tables, `make migrate-up`, copier avec les ids, recaler les séquences).
> - `DATABASE_URL` : URL `postgres://` ou `postgresql://` obligatoire, le format `clé=valeur` est refusé.
> - Migrations : elles s'appliquent au démarrage et la migration 2 réécrit les emails en minuscules (espaces retirés). Une base 1.0.0 passe par la procédure de `docs/troubleshooting.md` : les migrations tournent sur des tables vides, et doublons d'emails ou tâches orphelines se traitent aux étapes « contrôler » et « copier ». Pour une base créée par une version de développement (`user_models` déjà remplie), une migration refusée (doublons une fois normalisés, tâches orphelines) laisse la base « dirty » : corriger les données, puis `make migrate-force version=N` (N donné par le message d'erreur), depuis un checkout du dépôt avec Go (l'image ne l'a pas). Sauvegarder la base avant.
> - Corps de requête : v1.0.0 ne validait rien. Seul `application/json` est accepté, une propriété inconnue ou `null` sur une propriété facultative est refusée (400 `INVALID_BODY`). Répondent 400 `VALIDATION_FAILED` : nom, email et titre absents ou faits d'espaces, format d'email invalide, `user_id` absent ou 0, valeur trop longue. Nom, titre et description sont rognés et l'email est mis en minuscules à la création : la valeur relue diffère de celle envoyée.
> - Codes de statut : 409 `EMAIL_TAKEN` (au lieu de 500) ; 404 `TASK_NOT_FOUND` sur `DELETE /tasks/{id}` (au lieu de 200) et sur `PUT /tasks/{id}` (v1.0.0 insérait une nouvelle tâche, réponse 200) ; 404 `USER_NOT_FOUND` à la création d'une tâche ; 400 `INVALID_ID` pour un identifiant de chemin non numérique ou au-delà de 4 294 967 295 (v1.0.0 : 404 sur `GET /users/{id}`, 500 ailleurs) ; 413 `PAYLOAD_TOO_LARGE` au-delà de 1 MiB.
> - Messages `error` et `message` en français ou en anglais selon `Accept-Language` ; ne pas comparer les textes, utiliser `error_code`.
> - Listes d'utilisateurs et de tâches triées par `id` croissant.
> - `/ready` (base joignable) s'ajoute à `/health` ; les healthchecks d'orchestration doivent viser `/ready`.
> - Nouvelle image Docker `ghcr.io/fdanielsin/inbackend` (privée), non utilisée tant qu'aucun déploiement ne la tire.

### Security

- Go 1.26.8 : corrige les vulnérabilités atteignables de la stdlib 1.25.0 (`Q-03`).
- `golang.org/x/crypto` v0.56.0 et `moby/go-archive` v0.3.0 : corrigent GO-2026-6355, GO-2026-6354 et GO-2026-6253, atteignables depuis les tests.

### Added

- Hook `pre-push` : refuse un push vers `main`, sauf avec `RELEASE_TAG` pointant sur le commit poussé (`scripts/check-push-main.sh`) ; activation locale par `git config core.hooksPath .githooks`.
- Image Docker du backend publiée sur `ghcr.io/fdanielsin/inbackend` (privée) à chaque push sur `develop` et sur les tags `v*` (tags sha complet, `develop`, nom du tag), avec `HEALTHCHECK`, labels OCI et rétention des versions (`develop`, `v*` et 10 versions sha gardés, suppression activée par la variable de dépôt `PRUNE_IMAGE_VERSIONS`) ; un push plus récent sur la même ref annule le run de CI en cours ([#31](https://github.com/fdanielsin/inbackend/pull/31)) ; la CI se lance aussi à la main (`workflow_dispatch`) et le job `docker` contrôle `/openapi.yaml`.
- Lint plus strict : `govet` shadow, `revive` import-shadowing, `importas` (alias `usercommands` et `userqueries` seulement), `forbidigo` (`os.Exit` hors `main.go`), `testifylint` ; contrôle des noms de paquets dans `make lint` (`make lint-packages`, exceptions justifiées dans `scripts/package-name-exceptions.txt`) ([#30](https://github.com/fdanielsin/inbackend/pull/30)).
- Corps de requête validés contre le contrat : propriété inconnue refusée (400 `INVALID_BODY`) ; nom (100), email (254), titre (200) et description (2000) limités en caractères, `user_id` limité à 4 294 967 295 (400 `VALIDATION_FAILED`) ([#23](https://github.com/fdanielsin/inbackend/pull/23)).
- `GET /openapi.yaml` (contrat embarqué) et `GET /docs` (Swagger UI, version épinglée avec empreintes d'intégrité) ; `API_DOCS_ENABLED` (actif par défaut, inactif en production, valeur invalide refusée).
- Messages `error` et `message` en français ou en anglais selon `Accept-Language` (français pour `fr` et ses variantes, anglais sinon) ; `Content-Language` et `Vary: Accept-Language` sur toutes les réponses.
- `GET /ready` : disponibilité (base joignable sous 2 s, sinon 503 `DATABASE_UNAVAILABLE`) ; `/health` reste une sonde de vivacité sans base ; réponses des deux sondes non mises en cache.
- Gabarit « ajouter un endpoint » (`docs/adding-an-endpoint.md`).
- Identifiant de requête `X-Request-Id` (repris du client s'il est valide), présent dans les logs ; erreurs d'Echo (route inconnue, méthode non supportée, corps trop gros) dans l'enveloppe avec `ROUTE_NOT_FOUND`, `METHOD_NOT_ALLOWED`, `PAYLOAD_TOO_LARGE` (413 au-delà de 1 MiB).
- Délais du serveur HTTP, arrêt gracieux sur SIGINT/SIGTERM et pool SQL borné.
- `CORS_ALLOWED_ORIGINS` (`*` par défaut).
- `make cover` : couverture de `internal/` avec minimum bloquant de 85 %, aussi en CI (`Q-04`).
- Migrations versionnées golang-migrate (`make migrate-up/down/force/new`, ADR 0007) ; `AutoMigrate` retiré (`B-05`). Le démarrage refuse une base qui contient encore les tables `users` et `tasks` de la 1.0.0, avec la procédure de copie dans `docs/troubleshooting.md`.
- `B-02` : clé étrangère `task_models.user_id` -> `user_models.id` ; créer une tâche pour un utilisateur inexistant répond 404 `USER_NOT_FOUND`.
- Champ `error_code` (code stable, enum dans le contrat) dans toute réponse d'erreur ; une seule table erreur → statut/code/message (lot 4).
- `make mutate` : tests de mutation (gremlins) sur le diff, hors CI.
- Spécification OpenAPI 3.1 du contrat HTTP (`docs/specs/api/openapi.yaml`).
- Tests de régression pour `Q-01` (unitaires, e2e, intégration PostgreSQL).
- Contrôles bloquants en CI : golangci-lint (`make lint`), Redocly sur le contrat
  OpenAPI (`make lint-api`), govulncheck (`make vuln`) et actionlint sur les
  workflows (`make lint-ci`).

### Changed

- `GET /health` renvoie `version` 1.1.0 et le contrat déclare `info.version` 1.1.0.
- Le serveur refuse de démarrer quand `ENV=production` et que `CORS_ALLOWED_ORIGINS` est absente, vide ou contient `*` : changement cassant pour un déploiement de production sans origines CORS ([#29](https://github.com/fdanielsin/inbackend/pull/29)).
- Paquets internes renommés pour ne plus entrer en collision avec la bibliothèque standard ou une dépendance : `logger` en `applog`, `database` en `pgsetup`, `domain/user` en `users`, `domain/task` en `tasks` (ADR 0008) ; aucun effet sur l'API ([#27](https://github.com/fdanielsin/inbackend/pull/27)).
- Seul `application/json` est accepté pour un corps de requête : XML et formulaire répondent 400 `INVALID_BODY` ([#23](https://github.com/fdanielsin/inbackend/pull/23)).
- `null` sur une propriété facultative (`description`, `completed`, `title` en `PUT`) répond 400 `INVALID_BODY` au lieu d'être ignoré ; `user_id` au-dessus de 4 294 967 295 répond 400 au lieu de 404 ([#23](https://github.com/fdanielsin/inbackend/pull/23)).
- Nom, titre et description sont débarrassés de leurs espaces de début et de fin ; un nom ou un titre fait d'espaces est refusé (400 `VALIDATION_FAILED`) ([#23](https://github.com/fdanielsin/inbackend/pull/23)).
- Textes français : « adresse e-mail » remplace « email » ; sondes en « Le service est opérationnel » et « Le service est prêt » ([#23](https://github.com/fdanielsin/inbackend/pull/23)).
- Contrat : `message` et `version` des réponses de santé ne sont plus des constantes ; `Stats` sans `format: int64` ; la réponse de `DELETE /tasks/{id}` déclare `message`.
- Healthcheck de Docker Compose et job `docker` de la CI sur `/ready` ; lignes d'accès des sondes réussies en Debug.
- Logs par `slog` seul (zap retiré) : JSON au niveau Info en production, texte au niveau Debug en développement, position du code source sur chaque ligne.
- Logs JSON de production : `ts` devient `time` (RFC 3339), `level` en majuscules, `caller` devient l'objet `source` ; la durée `latency` est en nanosecondes ; plus de pile d'appels sur les erreurs.
- `Q-11` : le hook pre-push (lint, build, tests unitaires, sans Docker) bloque réellement en cas d'échec.
- Logs de requête via `RequestLoggerWithConfig` (méthode, chemin sans query, route, statut, durée, request id) à la place de `middleware.Logger` déprécié ; format de ligne différent.
- `DATABASE_URL` doit être une URL `postgres://` (le format `clé=valeur` est refusé au démarrage avec un message clair) ; `.env.example` mis à jour.
- Emails existants normalisés en minuscules et unicité insensible à la casse en base ; le démarrage échoue avec un message clair s'il existe des doublons par casse ou des tâches orphelines (rien n'est supprimé).
- `B-03` : créer un utilisateur avec un email déjà pris répond 409 `EMAIL_TAKEN` au lieu de 500 (comparaison sur l'email normalisé).
- Emails normalisés à la création (espaces retirés, minuscules) et format minimal validé (un `@`, domaine à deux labels non vides, ni espace ni caractère de contrôle) (400 `VALIDATION_FAILED`).
- `B-04` : supprimer une tâche absente répond 404 `TASK_NOT_FOUND` au lieu de 200.
- `GET /api/v1/stats` compte les utilisateurs sans les charger.
- `B-08` : l'email et le nom des utilisateurs ne sont plus journalisés (l'identifiant seulement), ni le texte brut des erreurs PostgreSQL ; le logger GORM est silencieux.
- CI : `codecov-action` v3 remplacé par v5 (runtime signalé trop ancien par actionlint) ; actionlint ajouté.
- `Q-03` : CI, une seule version de Go (celle de `go.mod`).
- `Q-03` : CI, nouveau job `docker` bloquant (image, pile compose, `/health`).
- Backend restructuré en couches domain/application/infrastructure/controllers
  (DDD/Clean Architecture), CQRS applicatif introduit.
- Réponses HTTP standardisées.
- Tests réorganisés en unit/integration/e2e (ADR 0004).
- Documentation d'architecture et ADR 0006 alignées sur l'état réel du code.
- `make lint` et `make fmt` utilisent golangci-lint, gofumpt et goimports.
- Échec au démarrage journalisé en ERROR (« Server failed », champ `error`) ; le code de sortie reste 1.
- Contrat OpenAPI : descriptions de tags ajoutées, schéma inutilisé retiré.

### Fixed

- `GET /api/v1/users` retourne les utilisateurs triés par `id` croissant ([#24](https://github.com/fdanielsin/inbackend/pull/24)).
- `Q-03` : l'image Docker se construit et `docker compose up` démarre PostgreSQL et le backend.
- `Q-05` : le scan de vulnérabilités de la CI bloque (govulncheck remplace `nancy`).
- `Q-12` : `GET /api/v1/users/{id}/tasks` retourne les tâches triées par
  `id` croissant.
- `Q-01` : `PUT /api/v1/tasks/{id}` persiste les mises à jour partielles
  ([#3](https://github.com/fdanielsin/inbackend/pull/3)).
- Jobs CI d'intégration et E2E ciblent désormais leurs packages réels.
- `B-06` : `make lint` exécute réellement golangci-lint, comme l'annonce `make help`.
- `HEAD` répond comme `GET` (sans corps) sur chaque route de lecture, au lieu de 405 ; déclaré dans le contrat, autorisé par CORS ([#22](https://github.com/fdanielsin/inbackend/pull/22)).

### Removed

- `Q-10` : code mort retiré.
- Configuration obsolète : `.env.example` nettoyé (variables réellement lues
  uniquement).

## [1.0.0] - 2026-09-17

### Added

- Endpoint de statistiques agrégées.

[Unreleased]: https://github.com/fdanielsin/inbackend/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/fdanielsin/inbackend/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/fdanielsin/inbackend/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/fdanielsin/inbackend/releases/tag/v1.0.0
