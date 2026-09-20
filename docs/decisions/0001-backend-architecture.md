# ADR 0001 : architecture du backend Go

- **Statut** : accepté
- **Périmètre** : `inbackend`
- **Date** : 2026-09-19

## Contexte

Le backend actuel mélange dans les handlers HTTP la désérialisation JSON, la
logique applicative, l'accès GORM et la construction des réponses. Les modèles
servent simultanément de modèles métier, de modèles de persistance et de DTO
HTTP.

Cette organisation rend les tests et l'évolution du domaine plus difficiles.

## Décision

Organiser le backend autour de quatre zones :

1. `domain` pour le métier ;
2. `application` pour les cas d'utilisation et le CQRS ;
3. `controllers` pour les entrées HTTP ;
4. `infrastructure` pour les détails techniques.

Les repositories seront définis par des interfaces côté domaine et implémentés
par PostgreSQL/GORM côté infrastructure.

Les modèles métier, les modèles GORM et les DTO HTTP seront séparés lorsque
leurs responsabilités divergent.

## CQRS

Le projet utilisera un CQRS applicatif pragmatique :

- une command modifie les données ;
- une query lit les données ;
- une seule base PostgreSQL est conservée ;
- aucune réplication ou projection distribuée n'est introduite.

## Conséquences positives

- domaine indépendant des frameworks ;
- use cases testables sans PostgreSQL ;
- accès à la base isolé ;
- controllers plus simples ;
- évolution progressive sans casser les routes existantes.

## Conséquences acceptées

- davantage de fichiers et de types ;
- mappings explicites entre couches ;
- nécessité de maintenir des interfaces et des tests ciblés.

## Alternatives rejetées

- conserver toute la logique dans `handlers.go` ;
- introduire Event Sourcing ;
- introduire Saga ;
- séparer immédiatement les bases de lecture et d'écriture ;
- créer des abstractions spéculatives.
