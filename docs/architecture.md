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
    ├── application/
    │   ├── user/
    │   │   ├── commands/
    │   │   └── queries/
    │   └── task/
    │       ├── commands/
    │       └── queries/
    ├── domain/
    │   ├── user/
    │   │   ├── entity.go
    │   │   ├── errors.go
    │   │   ├── repository.go
    │   │   └── service.go       (cible, pas encore créé)
    │   └── task/
    │       ├── entity.go
    │       ├── errors.go
    │       ├── repository.go
    │       └── service.go       (cible, pas encore créé)
    ├── controllers/
    │   └── http/
    ├── infrastructure/
    │   ├── persistence/postgres/
    │   │   └── models.go
    │   ├── database/             (connexion, migrations SQL embarquées : ADR 0007)
    │   ├── http/    (cible, pas encore créé)
    │   ├── config/  (cible, pas encore créé)
    │   └── logger/
    └── shared/
        ├── errors/              (cible, pas encore créé)
        ├── pagination/          (cible, pas encore créé)
        └── response/            (cible, pas encore créé)
```

Aucun fichier `service.go` n'existe aujourd'hui dans `domain/user` ni
`domain/task` : la logique métier des entités suffit pour l'instant (voir
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

### Controllers

Les controllers HTTP décodent les requêtes, construisent une command ou une
query, appellent l'application et construisent la réponse HTTP. Ils ne doivent
pas accéder directement à GORM. Les détails techniques sont journalisés côté
serveur; les réponses 500 n'exposent qu'un message générique.

### Infrastructure

L'infrastructure implémente les ports du domaine et contient aujourd'hui
PostgreSQL, GORM et le logger. Elle n'utilise pas Echo et ne contient pas le
router : Echo est utilisé directement par les controllers
(`internal/controllers/http`) et par `internal/app`, où le router est
assemblé (`app.NewRouter`). Un package `infrastructure/http/` qui engloberait Echo et
le routage est une cible (voir Structure cible), pas l'état actuel. Il
n'existe pas non plus de package de configuration dédié : les variables
d'environnement (`DATABASE_URL`, `PORT`, `ENV`, `CORS_ALLOWED_ORIGINS`) sont lues directement dans
`main.go` et `internal/infrastructure/logger` ; un package `config/` est la
cible.

### Serveur HTTP et middlewares

- `app.Serve` : `http.Server` avec `ReadHeaderTimeout` 5 s, `ReadTimeout` 15 s,
  `WriteTimeout` 30 s, `IdleTimeout` 60 s ; SIGINT/SIGTERM déclenchent un arrêt
  gracieux (10 s pour les requêtes en cours), puis fermeture du pool SQL.
- Pool SQL : 25 connexions ouvertes au plus, 5 inactives, durée de vie 30 min.
- Ordre des middlewares : `RequestID` (X-Request-Id repris s'il correspond à
  `[A-Za-z0-9._-]{1,64}`, sinon UUID ; logger de requête porté par le contexte),
  `AccessLog` (méthode, chemin sans query, route, statut, durée ; jamais de
  corps), `Recover`, CORS (`CORS_ALLOWED_ORIGINS`, `*` par défaut), `BodyLimit`
  (1 MiB exacts, avec ou sans Content-Length : `http.MaxBytesReader` ; le
  `BodyLimit` d'Echo n'est pas utilisé car il laisse passer un JSON complet qui
  dépasse de quelques octets sans Content-Length).
- `http.NewErrorHandler` : les erreurs d'Echo (404 `ROUTE_NOT_FOUND`, 405
  `METHOD_NOT_ALLOWED`, 413 `PAYLOAD_TOO_LARGE`, panique 500) sortent dans
  l'enveloppe d'erreur avec leur `error_code`.

### Shared

`shared` est réservé aux composants réellement transversaux et stables, par
exemple pagination, erreurs communes ou réponses partagées. Il ne doit pas
devenir un dossier de helpers génériques.

## Direction des dépendances

```text
controllers ──> application ──> domain
infrastructure ────────────────> domain
controllers ────────────────────> infrastructure/logger
main ──> app ──> controllers, application, infrastructure
```

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

`GET /health` (`HealthController.HealthCheck`) est un controller sans command
ni query associée : il ne passe pas par la couche application.

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
