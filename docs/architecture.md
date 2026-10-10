# Architecture du backend

## Statut documentaire

Cette page décrit l'architecture retenue et la structure attendue. La
conformité effective doit être vérifiée dans le code, les tests et les
commandes de validation ; elle n'est pas déduite de cette page.

## Objectif

Réorganiser le backend Go selon une combinaison pragmatique de DDD, Clean
Architecture et architecture hexagonale, sans introduire de complexité
distribuée inutile.

Le domaine métier doit rester indépendant de GORM, PostgreSQL, Echo, HTTP et
JSON.

## Structure cible

```text
inbackend/
├── main.go
└── internal/
    ├── app/               (racine de composition : NewRouter)
    ├── application/       (user_response.go : UserResponse, partagé par les cas d'utilisation)
    │   ├── user/
    │   │   ├── commands/
    │   │   └── queries/
    │   └── tasks/
    │       ├── commands/
    │       └── queries/
    ├── domain/
    │   ├── users/
    │   │   ├── entity.go
    │   │   ├── errors.go
    │   │   ├── repository.go
    │   │   └── service.go       (cible, pas encore créé)
    │   └── tasks/
    │       ├── entity.go
    │       ├── errors.go
    │       ├── repository.go
    │       └── service.go       (cible, pas encore créé)
    ├── controllers/
    │   └── httpapi/
    ├── infrastructure/
    │   ├── persistence/pgstore/
    │   │   └── models.go
    │   ├── pgsetup/              (connexion, migrations SQL embarquées : ADR 0007)
    │   ├── http/    (cible, pas encore créé)
    │   ├── config/  (cible, pas encore créé)
    │   └── applog/
    └── shared/
        ├── errors/              (cible, pas encore créé)
        ├── pagination/          (cible, pas encore créé)
        └── response/            (cible, pas encore créé)
```

Aucun fichier `service.go` n'existe aujourd'hui dans `domain/users` ni
`domain/tasks` : la logique métier des entités suffit pour l'instant (voir
`entity.go`). Un `service.go` n'est ajouté que si une règle métier réelle ne
trouve sa place ni dans l'entité ni dans un cas d'usage applicatif — jamais
pour remplir une structure théorique.

`shared` est aujourd'hui composé de deux fichiers plats,
`internal/shared/errors.go` et `internal/shared/response.go` ; il n'y a pas
encore de `pagination`. Le découpage en sous-dossiers ci-dessus est la cible
si `shared` grossit au point de le justifier, pas l'état actuel.

## Responsabilités

### Domain

Le domaine contient les entités, les invariants, les erreurs métier et les
interfaces de repository. Il ne dépend d'aucune technologie externe.

### Application

La couche application contient les cas d'utilisation. Elle sépare :

- les **commands**, qui modifient l'état ;
- les **queries**, qui lisent l'état.

Il s'agit d'un CQRS applicatif avec une base PostgreSQL commune, pas d'un CQRS
distribué.

Les cas d'utilisation qui renvoient un utilisateur partagent le type
`application.UserResponse`.

### Controllers

Les controllers HTTP décodent les requêtes, construisent une command ou une
query, appellent l'application et construisent la réponse HTTP. Ils ne doivent
pas accéder directement à GORM. Les détails techniques sont journalisés côté
serveur; les réponses 500 n'exposent qu'un message générique.

### Infrastructure

L'infrastructure implémente les ports du domaine : persistance GORM sur
PostgreSQL (`persistence/pgstore`), migrations golang-migrate (`pgsetup`) et
logger. Elle n'utilise pas Echo et ne contient pas le router : Echo est utilisé
par les controllers (`internal/controllers/httpapi`), et `internal/app` assemble
le router (`app.NewRouter`). Un package `infrastructure/http/` qui engloberait
Echo et le routage est une cible (voir Structure cible), pas l'état actuel. Il
n'existe pas non plus de package de configuration dédié : `main.go` lit
`DATABASE_URL`, `PORT`, `ENV`, `CORS_ALLOWED_ORIGINS` et `API_DOCS_ENABLED`, et
`internal/infrastructure/applog` lit `ENV` ; un package `config/` est la cible.

### Serveur HTTP et middlewares

- `app.Serve` : `http.Server` avec `ReadHeaderTimeout` 5 s, `ReadTimeout` 15 s,
  `WriteTimeout` 30 s, `IdleTimeout` 60 s ; SIGINT/SIGTERM déclenchent un arrêt
  gracieux (10 s pour les requêtes en cours), puis fermeture du pool SQL.
- Pool SQL : 25 connexions ouvertes au plus, 5 inactives, durée de vie 30 min.
- Ordre des middlewares : `SecurityHeaders` (`X-Content-Type-Options: nosniff` sur toute réponse, y compris préflight CORS, 404, 413 et panique : il passe avant tous les autres pour que rien ne le court-circuite), `Language` (négocie la langue, voir ci-dessous), `RequestID` (X-Request-Id repris s'il correspond à
  `[A-Za-z0-9._-]{1,64}`, sinon UUID ; logger de requête porté par le contexte),
  `AccessLog` (méthode, chemin sans query, route, statut, durée ; jamais de
  corps ni de requête brute, prouvé par `tests/e2e/query_not_logged_test.go`), `Recover`, CORS (`CORS_ALLOWED_ORIGINS`, `*` par défaut sauf si `ENV=production`, où elle est
  obligatoire et sans `*` ; chaque origine est validée au démarrage par
  `app.ValidateAllowedOrigins`, avant `DATABASE_URL`), `BodyLimit`
  (1 MiB exacts, avec ou sans Content-Length : `http.MaxBytesReader` ; le
  `BodyLimit` d'Echo n'est pas utilisé car il laisse passer un JSON complet qui
  dépasse de quelques octets sans Content-Length).
- `httpapi.NewErrorHandler` : les erreurs d'Echo (404 `ROUTE_NOT_FOUND`, 405
  `METHOD_NOT_ALLOWED`, 413 `PAYLOAD_TOO_LARGE`, panique 500) sortent dans
  l'enveloppe d'erreur avec leur `error_code`.

### Contrat et Swagger UI

Le contrat `docs/specs/api/openapi.yaml` est embarqué dans `main.go`
(`go:embed`) et passé au routeur par `app.WithAPIDocs`, qui enregistre
`GET /openapi.yaml` et `GET /docs` (`httpapi.DocsController`). `main.go` décide
avec `API_DOCS_ENABLED` : actif par défaut hors `ENV=production`, valeur
invalide refusée au démarrage. Sans l'option, les routes n'existent pas.
Leur `Content-Language` est la langue réelle du contenu, sans tenir compte
d'`Accept-Language` : `fr` pour le contrat, `en` pour la page. `/docs` charge
Swagger UI depuis un CDN à version et empreintes épinglées ; la
CSP autorise le script d'initialisation par son hash, calculé au démarrage. Le
`.dockerignore` laisse passer `openapi.yaml` pour que l'image se construise.
Le premier serveur du contrat est `/` pour que « Try it out » appelle l'origine
de la page et reste sous `connect-src 'self'`.

### Validation des corps de requête

`httpapi.RequestValidator`, branché par `app.WithRequestValidator` après
`BodyLimit` (le 413 passe avant), lit dans le contrat embarqué le schéma du corps
de `POST /api/v1/users`, `POST /api/v1/tasks` et `PUT /api/v1/tasks/{id}`
(`app.ValidatedOperations`). La validation ne dépend pas de `API_DOCS_ENABLED`.

Le validateur est partiel volontairement. Il applique : propriété inconnue,
casse comprise (400 `INVALID_BODY`), `null` sur une propriété facultative (400
`INVALID_BODY`), `maxLength` en caractères Unicode de la valeur reçue avant le
retrait des espaces, et `maximum` d'un nombre (400 `VALIDATION_FAILED`). Le reste
garde ses codes et textes, rendus par `Bind` et le domaine : type, corps absent
ou `null`, champ obligatoire, vide, `minLength: 1`, `minimum: 1` de `user_id`,
format d'email (partie locale ≤ 64, labels LDH ≤ 63, TLD de 2 à 24 lettres ou
`xn--` ; règle exacte dans le contrat). Le domaine retire aussi les espaces de
début et de fin du nom, de l'email, du titre et de la description.

Un validateur générique (`openapi3filter` de kin-openapi) n'a pas été retenu :
pour un contrat OpenAPI 3.1 il ne rend que du texte anglais, sans mot-clé ni
chemin exploitables, qu'on ne pourrait relier à nos codes et messages qu'en
l'analysant, ce qui est fragile. D'où le risque d'un mot-clé du contrat ignoré en
silence, que `NewRequestValidator` écarte au démarrage : il refuse (le serveur ne
démarre pas) une opération absente, sans corps JSON, ou sans
`additionalProperties: false`, et un schéma qui utilise un mot-clé non appliqué.
Mots-clés acceptés, corps : `type`, `properties`, `additionalProperties`,
`required` (seulement `name`, `email`, `title`, `user_id`) ; propriété : `type`,
`maxLength`, `maximum`, `format` (`email` ou `uint32`), `minLength: 1` (seulement
`name`, `email`, `title`), `minimum: 1` (seulement `user_id`) ; annotations, sans
effet : `description`, `title`, `default`, `example`, `examples`, `deprecated`,
`x-*`. Tout autre mot-clé (`pattern`, `enum`, `oneOf`...) impose d'abord
d'étendre le validateur. Les colonnes `name`, `email`, `title` et `description`
sont des `text` non bornés : les limites ne vivent que dans le contrat.

### Langue des messages

Deux langues, l'anglais par défaut et le français. `shared.NegotiateLanguage`
lit `Accept-Language` : le français si la langue prise en charge la plus
demandée (`q` le plus haut, RFC 9110) est une variante de `fr`, l'anglais sinon
et pour un en-tête absent ou invalide ; jamais de 400. Chaque élément doit
respecter la RFC 9110 (x/text tolère `q` supérieur à 1 et tirets bas, pas nous :
un élément invalide invalide tout l'en-tête), et un en-tête de plus de
1024 octets, qu'aucun client réel n'envoie, donne l'anglais sans être analysé. Le middleware `Language`
met la langue dans le contexte et pose `Content-Language` et
`Vary: Accept-Language` sur toute réponse, avant CORS pour couvrir le préflight.
Les textes `error` et `message` viennent du catalogue `shared/messages.go`
(clés `Msg*`, un texte par langue, test de complétude) ; les `error_code`, les
valeurs machine et les logs restent en anglais et ne dépendent pas de la langue.

### Shared

`shared` est réservé aux composants réellement transversaux et stables, par
exemple pagination, erreurs communes ou réponses partagées. Il ne doit pas
devenir un dossier de helpers génériques.

## Direction des dépendances

```text
controllers ──> application ──> domain
infrastructure ────────────────> domain
main ──> app ──> controllers, application, infrastructure
```

Les controllers reçoivent un `*slog.Logger` injecté par `app.NewRouter`.

Le domaine définit les contrats. L'infrastructure fournit les implémentations
(persistance, logger) et ne dépend que du domaine. `internal/app` assemble les
dépendances (repositories, handlers, controllers, middlewares, routes) ; `main.go`
lit la configuration, ouvre la base et appelle `app.NewRouter`. Rien ne dépend
de `app` sauf `main.go` et les tests E2E.

## CQRS retenu

| Type | Exemples |
| --- | --- |
| Commands | `CreateUser`, `CreateTask`, `UpdateTask`, `DeleteTask` |
| Queries | `GetUser`, `ListUsers`, `ListUserTasks`, `GetStats` |

`GET /health` (vivacité, sans base) et `GET /ready` (disponibilité : la base
répond sous 2 s, sinon 503 `DATABASE_UNAVAILABLE`) sont servis par
`HealthController` sans command ni query : ils ne passent pas par la couche
application. Leurs réponses portent `Cache-Control: no-store` et, réussies,
leurs lignes d'accès sont journalisées en Debug. Pour ajouter un endpoint :
[`adding-an-endpoint.md`](./adding-an-endpoint.md).

## Éléments explicitement exclus

- Event Sourcing ;
- Saga ;
- Event Store ;
- transactions distribuées ;
- deux bases séparées pour le CQRS ;
- microservices artificiels ;
- Domain Events sans consommateur concret.

Des Domain Events classiques pourront être ajoutés plus tard si une réaction
métier réelle, comme un audit ou une notification, le justifie. Ils ne serviront
pas à reconstruire l'état de la base.

## Références

Le code Go suit deux références, avec les écarts volontaires ci-dessous :

- [Effective Go](https://go.dev/doc/effective_go) ;
- [Google Go Style, Best practices](https://google.github.io/styleguide/go/best-practices.html)
  (et, quand elles y renvoient, les pages [Guide](https://google.github.io/styleguide/go/guide.html)
  et [Decisions](https://google.github.io/styleguide/go/decisions.html)).

Écarts volontaires retenus :

- les ports de persistance (`user.Repository`, `task.Repository`) sont définis
  côté domaine, pas côté consommateur : architecture hexagonale
  ([ADR 0001](./decisions/0001-backend-architecture.md)) ;
- les tests vivent dans `tests/` (unitaires, intégration, e2e), en paquets
  externes, et non à côté du code ([ADR 0004](./decisions/0004-test-strategy.md)) ;
- `testify` est utilisé pour les assertions (le guide Google préfère les
  comparaisons écrites dans le test) ;
- le logger est injecté (`Init` le retourne, `app.WithLogger` et les
  constructeurs de controllers le reçoivent), mais `Init` appelle aussi
  `slog.SetDefault` : le repli de `NewRouter` et les tests de capture de logs
  s'appuient sur `slog.Default` ;
- le paquet `shared` (enveloppe de réponse et table des erreurs) garde son nom,
  malgré la mise en garde du guide contre les paquets fourre-tout : sa
  responsabilité est décrite plus haut (« Shared ») ;
- `app.NewRouter` prend des options variadiques (`app.Option`) bien qu'il n'y en
  ait qu'une aujourd'hui (`WithAllowedOrigins`) : les appelants existants
  restent valides quand une option s'ajoute.

## Nommage des paquets

Un paquet ne reprend ni le nom d'un paquet de la bibliothèque standard ni celui
d'une dépendance importée : `httpapi` et non `http` (`net/http`), `pgstore` et
non `postgres` (driver GORM). Un alias d'import n'est admis que pour une vraie
collision entre deux imports (`usercommands` et `userqueries`, deux paquets
`commands`), jamais pour contourner le nom d'un de nos propres paquets.

`make lint` applique la règle (`scripts/check-package-names.sh`, aussi lancé seul
par `make lint-packages`) : il compare chaque paquet du module aux paquets de la
bibliothèque standard et des dépendances. Une exception se déclare dans
`scripts/package-name-exceptions.txt` (une ligne « chemin : justification », la
justification est obligatoire) ; elle est vide aujourd'hui, et une exception qui
ne correspond plus à une collision est refusée.

## Règles de lint

En plus des règles de base, `make lint` (donc la CI) active : `govet` avec
`shadow` (une variable ne masque pas une variable de la portée englobante),
`revive` avec `import-shadowing` (une variable ne porte pas le nom d'un paquet
importé), `importas` avec `no-extra-aliases` (seuls `usercommands` et
`userqueries` sont admis), `forbidigo` (`os.Exit` seulement dans `main.go` ; un
`TestMain` retourne, sauf celui de `tests/e2e/server_test.go`, qui sort
explicitement pour que la porte de couverture du contrat échoue la suite, avec
`//nolint:forbidigo` justifié) et `testifylint` (bon usage de `require` et
`assert`). Une règle ne se désactive que localement, avec sa justification.
