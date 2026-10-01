# Spécification des données — persistance

## Statut

`active` — schéma défini par les migrations SQL (`internal/infrastructure/database/migrations/`, ADR 0007) ; les modèles GORM doivent rester cohérents avec lui.

## Utilisateur (table `user_models`)

| Champ | Type Go | Contraintes |
| --- | --- | --- |
| `id` | `uint` | clé primaire (`bigserial`) |
| `name` | `string` | non vide au niveau domaine |
| `email` | `string` | normalisé (espaces retirés, minuscules) et format minimal validé au niveau domaine ; index unique `idx_user_models_email_lower` sur `lower(email)` |

## Tâche (table `task_models`)

| Champ | Type Go | Contraintes |
| --- | --- | --- |
| `id` | `uint` | clé primaire (`bigserial`) |
| `title` | `string` | non vide à la création au niveau domaine |
| `description` | `string` | facultatif |
| `user_id` | `uint` | non nul à la création ; clé étrangère `fk_task_models_user` vers `user_models.id` (index `idx_task_models_user_id`) |
| `completed` | `bool` | valeur par défaut Go `false` |

## Persistance

- PostgreSQL est la base d'exécution.
- Les migrations versionnées (golang-migrate) sont appliquées au démarrage.
- Les tests de repository utilisent PostgreSQL via Testcontainers lorsqu'un
  provider Docker/Podman est disponible.
- Les modèles de persistance sont isolés dans
  `internal/infrastructure/persistence/pgstore/models.go`.

## Limites connues

- Pas de suppression d'utilisateur dans l'API : la clé étrangère est `RESTRICT`.
- Pas de pagination des listes.
- Les noms de tables `user_models` et `task_models` viennent de GORM ; un
  renommage serait une migration séparée.
