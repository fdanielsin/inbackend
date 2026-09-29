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
    │   ├── database/
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
(`internal/controllers/http`) et par `main.go`, où le router est assemblé
(`main.go:56-75`). Un package `infrastructure/http/` qui engloberait Echo et
le routage est une cible (voir Structure cible), pas l'état actuel. Il
n'existe pas non plus de package de configuration dédié : les variables
d'environnement (`DATABASE_URL`, `PORT`, `ENV`) sont lues directement dans
`main.go` et `internal/infrastructure/logger` ; un package `config/` est la
cible.

### Shared

`shared` est réservé aux composants réellement transversaux et stables, par
exemple pagination, erreurs communes ou réponses partagées. Il ne doit pas
devenir un dossier de helpers génériques.

## Direction des dépendances

```text
controllers ──> application ──> domain
infrastructure ────────────────> domain
controllers ────────────────────> infrastructure/logger
```

Le domaine définit les contrats. L'infrastructure fournit les implémentations
(persistance, logger) et ne dépend que du domaine. `main.go` assemble les
dépendances.

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
