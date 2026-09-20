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
    │   │   └── service.go
    │   └── task/
    │       ├── entity.go
    │       ├── errors.go
    │       ├── repository.go
    │       └── service.go
    ├── controllers/
    │   └── http/
    ├── infrastructure/
    │   ├── persistence/postgres/
    │   │   └── models.go
    │   ├── http/
    │   ├── config/
    │   └── logger/
    └── shared/
        ├── errors/
        ├── pagination/
        └── response/
```

Les fichiers `service.go` et les packages de `shared` ne doivent contenir que
des responsabilités réelles. Ils ne sont pas créés pour remplir une structure
théorique.

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

L'infrastructure implémente les ports du domaine et contient PostgreSQL, GORM,
Echo, le router, la configuration et le logger.

### Shared

`shared` est réservé aux composants réellement transversaux et stables, par
exemple pagination, erreurs communes ou réponses partagées. Il ne doit pas
devenir un dossier de helpers génériques.

## Direction des dépendances

```text
controllers ────────┐
infrastructure ─────┼──> application ───> domain
                    │
```

Le domaine définit les contrats. L'infrastructure fournit les implémentations.
`main.go` assemble les dépendances.

## CQRS retenu

| Type | Exemples |
| --- | --- |
| Commands | `CreateUser`, `CreateTask`, `UpdateTask`, `DeleteTask` |
| Queries | `GetUser`, `ListUsers`, `ListUserTasks`, `GetTaskStats`, `HealthCheck` |

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
