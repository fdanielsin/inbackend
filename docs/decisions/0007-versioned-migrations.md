# ADR 0007 : Migrations versionnées avec golang-migrate

- **Statut** : accepted
- **Périmètre** : `inbackend`
- **Date** : 2026-10-01 (décision utilisateur)
- **Mise à jour** : le paquet `database` est devenu `pgsetup` (ADR 0008) ; les
  chemins et noms ci-dessous se lisent avec ce nom.

## Contexte

Le schéma était créé par `GORM AutoMigrate` au démarrage : aucune trace des
changements, aucun retour arrière, aucune contrainte ajoutée (pas de clé
étrangère, index unique de l'email sensible à la casse). Les nouvelles
fonctionnalités ont besoin de faire évoluer le schéma de façon sûre.

## Décision

- Migrations SQL numérotées, `up` et `down`, dans
  `internal/infrastructure/database/migrations/`, exécutées par
  [golang-migrate](https://github.com/golang-migrate/migrate) v4 (driver
  `pgx/v5`, source `iofs`).
- Elles sont embarquées dans le binaire (`go:embed`) et appliquées au démarrage
  par `database.InitDB` (`database.Migrate`). `AutoMigrate` est retiré ; les
  modèles GORM ne décrivent plus le schéma, ils doivent rester cohérents avec lui.
- Un verrou consultatif est pris par le driver : plusieurs instances peuvent
  démarrer en même temps.
- `DATABASE_URL` doit être une URL `postgres://` ou `postgresql://` (le format
  `clé=valeur` n'est pas supporté) ; sinon le démarrage échoue tout de suite avec
  « DATABASE_URL must be a postgres:// URL », sans afficher la valeur.
- Outils : `make migrate-up`, `migrate-down` (une migration), `migrate-force
  version=N`, `migrate-new name=...` (CLI épinglée via `go run`, `DATABASE_URL`
  requis).

## Règles

1. Toute migration a un `up` et un `down`.
2. Nommage `000NNN_sujet` (six chiffres, `make migrate-new`), un seul sujet.
3. Une migration appliquée n'est jamais modifiée : on en ajoute une nouvelle.
4. Non destructive : un `up` qui pourrait perdre ou déformer des données
   vérifie d'abord sa précondition et échoue avec un message clair, sans rien
   modifier. Ce message ne contient que des nombres, jamais de donnée
   utilisateur (les erreurs finissent dans les logs). Seul le `down` de la
   migration initiale supprime des tables.
5. Les contraintes sont nommées explicitement : le code détecte une violation
   par son code SQLSTATE et le nom de la contrainte (`constraints.go`).
6. Chaque migration est testée en montée, en descente puis en montée, sur base
   vide et, pour 000001, sur une base créée par GORM `AutoMigrate` pour les
   modèles `pgstore`.
7. Les tables de la 1.0.0 (`users`, `tasks`) ne sont pas migrées : `Migrate`
   refuse la base, avant toute écriture, tant qu'elles existent dans le schéma
   courant (`ErrLegacySchema`, code de sortie 1). Décision utilisateur du
   2026-10-03 : pas de renommage ni de reprise automatique, une copie SQL
   manuelle documentée dans `troubleshooting.md` et testée. Limite : le CLI
   `make migrate-up` ne passe pas par `Migrate` et ne fait pas ce contrôle ; il
   crée alors des tables vides, le serveur refuse quand même, et la procédure
   reste valable car elle vérifie que les tables cibles sont vides.

## Migrations actuelles

| Version | Effet |
| --- | --- |
| 000001 | Schéma initial (`user_models`, `task_models`, `idx_user_models_email`), identique à ce que `AutoMigrate` crée pour les modèles `pgstore`, idempotent. Pas de reprise des tables `users` et `tasks` de la 1.0.0 (règle 7). |
| 000002 | Emails : refuse s'il existe des doublons une fois les espaces retirés et passé en minuscules ; sinon normalise les emails existants et remplace l'index par l'index unique `idx_user_models_email_lower` sur `lower(email)`. La mise en minuscules n'est pas réversible. |
| 000003 | Clé étrangère `fk_task_models_user` (`task_models.user_id` -> `user_models.id`) et index `idx_task_models_user_id` ; refuse s'il existe des tâches orphelines. |
| 000004 | `fk_task_models_user` passe à `ON DELETE CASCADE` (une seule instruction `ALTER TABLE`, nom conservé) ; aucune ligne modifiée ; `down` remet le refus de 000003. |

Les tables s'appellent `user_models` et `task_models` (noms de GORM pour les
modèles `pgstore`) ; la 1.0.0 utilisait `users` et `tasks` (règle 7). Un
renommage serait une migration séparée.

## Échec d'une migration

PostgreSQL annule le fichier (transaction implicite), mais golang-migrate marque
la base « dirty » et refuse d'avancer. Le serveur ne démarre pas (voulu) et
l'erreur donne la commande : corriger la cause (doublons, tâches orphelines),
puis `make migrate-force version=N` avec N la dernière version entièrement
appliquée, puis relancer.

## Conséquences

- Le démarrage dépend de la base et de l'état des migrations.
- Toute évolution du schéma passe par une nouvelle migration et, si elle change
  le contrat, par `docs/specs/api/openapi.yaml` d'abord.
- Les tests d'intégration appliquent ces mêmes migrations.
