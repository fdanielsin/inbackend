# Spécification des données — persistance

## Statut

`active` — schéma défini par les migrations SQL (`internal/infrastructure/pgsetup/migrations/`, ADR 0007) ; les modèles GORM doivent rester cohérents avec lui.

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
| `user_id` | `uint` | non nul à la création ; clé étrangère `fk_task_models_user` vers `user_models.id`, `ON DELETE CASCADE` (index `idx_task_models_user_id`) |
| `completed` | `bool` | valeur par défaut Go `false` |

## Persistance

- PostgreSQL est la base d'exécution.
- Les migrations versionnées (golang-migrate) sont appliquées au démarrage.
- Les tests de repository utilisent PostgreSQL via Testcontainers lorsqu'un
  provider Docker/Podman est disponible.
- Les modèles de persistance sont isolés dans
  `internal/infrastructure/persistence/pgstore/models.go`.

## Limites connues

- La clé étrangère `fk_task_models_user` est `ON DELETE CASCADE` (migration 000004) : supprimer un utilisateur supprime ses tâches dans la même instruction, y compris par un `DELETE` manuel en SQL, sans avertissement. Le nom de la contrainte est lu par le code (`pgstore/constraints.go`) : ne pas le changer. Les identifiants (`bigserial`) ne sont jamais réattribués par l'application ; seul un `TRUNCATE ... RESTART IDENTITY` ou une recréation du schéma (outillage de test) les réinitialise.
- Pas de pagination des listes.
- Les noms de tables `user_models` et `task_models` viennent de GORM ; un
  renommage serait une migration séparée.
