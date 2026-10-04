# Dépannage

## `connection refused` au démarrage

La base n'est pas joignable à l'adresse de `DATABASE_URL`. Vérifier qu'elle
tourne (`docker ps`) et que l'hôte et le port correspondent. Le serveur échoue
au démarrage plutôt que d'attendre la base.

## Le serveur refuse `DATABASE_URL`

`DATABASE_URL` doit être une URL `postgres://` ou `postgresql://` ; le format
`clé=valeur` est refusé. En local, ajouter `?sslmode=disable` si la base n'a pas
de TLS.

## Migration en échec (base « dirty »)

Un échec laisse la base marquée « dirty » et golang-migrate refuse d'avancer.
Le message d'erreur donne la version à passer à
`make migrate-force version=N` (N = dernière version entièrement appliquée),
voir l'[ADR 0007](./decisions/0007-versioned-migrations.md).

## Mise à jour depuis la 1.0.0 : « database has 1.0.0 tables »

La 1.0.0 stockait les données dans les tables `users` et `tasks` ; la 1.1.0
lit `user_models` et `task_models`. Plutôt que de démarrer sur des tables vides
et de masquer les données, le serveur (et `Migrate`) refuse de démarrer, code de
sortie 1, tant que ces tables existent dans le schéma courant, même si une 1.1.0
a déjà tourné une fois sur la base :

```text
failed to initialize database: migrate database: database has 1.0.0 tables (users, tasks): ...
```

Rien n'est copié ni supprimé automatiquement. Procédure, serveur arrêté et base
sauvegardée (`pg_dump`), dans `psql` ou tout client SQL. Les ids sont conservés.

1. Renommer les anciennes tables : le refus ne les voit plus.

```sql
-- step rename
BEGIN;
ALTER TABLE users RENAME TO users_v1;
ALTER TABLE tasks RENAME TO tasks_v1;
COMMIT;
```

2. Créer les tables 1.1.0 sans servir de trafic, depuis un checkout du dépôt
   au tag voulu, avec Go : `make migrate-up` (`DATABASE_URL` en `postgres://`).
   `make migrate-up` lance le CLI golang-migrate, qui ne fait pas le contrôle de
   refus : lancé avant l'étape 1, il crée des tables 1.1.0 vides ; le serveur
   refuse quand même de démarrer, et la procédure reste valable, car l'étape 3
   vérifie que les tables cibles sont vides.
   Démarrer le serveur ici est déconseillé : une écriture arrivée avant la copie
   entrerait en conflit avec les ids copiés.

3. Contrôler. La requête renvoie une ligne ; la copie n'est sûre que si les trois
   nombres valent 0 (tables cibles vides, emails égaux une fois les espaces retirés
   et passés en minuscules, tâches dont l'utilisateur n'existe pas). Sinon corriger
   `users_v1` et `tasks_v1`, ou fusionner à la main si la 1.1.0 a déjà reçu des données.

```sql
-- step check
SELECT
  (SELECT count(*) FROM user_models) + (SELECT count(*) FROM task_models) AS target_rows,
  (SELECT count(*) FROM (
     SELECT 1 FROM users_v1 WHERE email IS NOT NULL
     GROUP BY lower(btrim(email, E' \t\n\r\f\v')) HAVING count(*) > 1) AS d) AS duplicate_email_groups,
  (SELECT count(*) FROM tasks_v1 t
     WHERE NOT EXISTS (SELECT 1 FROM users_v1 u WHERE u.id = t.user_id)) AS orphan_tasks;
```

4. Copier, puis recaler les séquences sur le plus grand id copié.

```sql
-- step copy
BEGIN;
INSERT INTO user_models (id, name, email)
  SELECT id, name, lower(btrim(email, E' \t\n\r\f\v')) FROM users_v1;
INSERT INTO task_models (id, title, description, user_id, completed)
  SELECT id, title, description, user_id, completed FROM tasks_v1;
SELECT setval(pg_get_serial_sequence('user_models', 'id'),
              COALESCE((SELECT max(id) FROM user_models), 1),
              (SELECT max(id) FROM user_models) IS NOT NULL);
SELECT setval(pg_get_serial_sequence('task_models', 'id'),
              COALESCE((SELECT max(id) FROM task_models), 1),
              (SELECT max(id) FROM task_models) IS NOT NULL);
COMMIT;
```

5. Démarrer la 1.1.0 et vérifier les données. Les tables `users_v1` et `tasks_v1`
   ne gênent plus ; une fois vérifié, les supprimer (facultatif, irréversible sans sauvegarde).

```sql
-- step drop
DROP TABLE tasks_v1;
DROP TABLE users_v1;
```

Le SQL ci-dessus est exécuté tel quel par
`tests/integration/persistence/legacy_schema_test.go` : ne le modifier qu'avec ce test.

## Les tests d'intégration ou e2e ne démarrent pas

Ils lancent PostgreSQL avec Testcontainers : le démon Docker (ou Podman) doit
répondre.

```bash
docker ps
podman machine start   # avec Podman sous macOS
```

Avec Podman, Testcontainers ne trouve pas toujours le socket : fournir
`DOCKER_HOST` avec le socket de la machine Podman et
`TESTCONTAINERS_RYUK_DISABLED=true`. Sans fournisseur de conteneurs, les tests
PostgreSQL sont ignorés en local (signalé comme ignoré, jamais comme réussi) ;
en CI (`CI=true`), ils échouent.

## Un test dépasse le délai

Augmenter le délai de `go test`, par exemple
`go test -timeout 120s ./tests/...`.

## Base PostgreSQL locale sans Docker

```bash
createdb inbackend
export DATABASE_URL="postgres://localhost:5432/inbackend?sslmode=disable"
```
