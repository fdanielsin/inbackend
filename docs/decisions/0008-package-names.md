# ADR 0008 : Noms de paquets sans collision

- **Statut** : accepted
- **Périmètre** : `inbackend`
- **Date** : 2026-10-02 (décision utilisateur)

## Contexte

Quatre paquets portaient le nom d'un paquet connu de la bibliothèque standard ou
d'une dépendance : `domain/user` (`os/user`, `moby/sys/user`),
`infrastructure/database` (`golang-migrate/migrate/v4/database`),
`infrastructure/logger` et `tests/units/logger` (`gorm.io/gorm/logger`). Chaque
collision impose un alias à l'import ou masque un nom, ce que les
[recommandations de style Go](https://google.github.io/styleguide/go/best-practices.html#package-names)
déconseillent : un nom de paquet court, qui dit ce qu'il fait, et qui ne se
confond avec aucun autre.

## Décision

- Renommer : `infrastructure/logger` en `applog` (et `tests/units/logger` en
  `tests/units/applog`), `infrastructure/database` en `pgsetup` (connexion,
  pool et migrations PostgreSQL, voisin de `persistence/pgstore`), `domain/user`
  en `users` et, par symétrie avec les routes `/users` et `/tasks`, `domain/task`
  en `tasks`. Les entités gardent leur nom (`users.User`, `tasks.Task`).
- Règle : un paquet du projet ne porte pas le nom d'un paquet de la
  bibliothèque standard ni d'une dépendance.
- Le contrôle est réalisé (#30) : `scripts/check-package-names.sh`, lancé par
  `make lint` (aussi seul par `make lint-packages`), compare chaque paquet du
  module à la bibliothèque standard et aux dépendances. Les exceptions, versionnées
  et justifiées, vivent dans `scripts/package-name-exceptions.txt` (vide aujourd'hui).

## Conséquences

- Renommage sans effet sur l'API ni sur le comportement : chemins d'import,
  clauses `package` et qualificatifs seulement, dans `internal/`, `main.go` et
  `tests/`. Quelques variables locales nommées `users` ou `tasks` ont changé de
  nom là où elles masquaient le nouveau paquet.
- Les chemins des anciens paquets cités dans les ADR précédentes se lisent avec
  les nouveaux noms.
