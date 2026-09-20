# Spécification des données — persistance

## Statut

`active` — modèle correspondant aux modèles GORM actuellement migrés.

## Utilisateur

| Champ | Type Go | Contraintes |
| --- | --- | --- |
| `id` | `uint` | clé primaire générée par GORM |
| `name` | `string` | non vide au niveau domaine |
| `email` | `string` | index unique GORM, non vide au niveau domaine |

## Tâche

| Champ | Type Go | Contraintes |
| --- | --- | --- |
| `id` | `uint` | clé primaire générée par GORM |
| `title` | `string` | non vide à la création au niveau domaine |
| `description` | `string` | facultatif |
| `user_id` | `uint` | identifiant utilisateur non nul à la création |
| `completed` | `bool` | valeur par défaut Go `false` |

## Persistance

- PostgreSQL est la base d'exécution.
- Les migrations sont exécutées au démarrage via `AutoMigrate`.
- Les tests de repository utilisent PostgreSQL via Testcontainers lorsqu'un
  provider Docker/Podman est disponible.
- Les modèles de persistance sont isolés dans
  `internal/infrastructure/persistence/postgres/models.go`.

## Limites connues

- La contrainte de relation entre `tasks.user_id` et `users.id` n'est pas
  déclarée explicitement dans les modèles GORM.
- Les migrations versionnées et les suppressions destructives ne sont pas
  définies.
