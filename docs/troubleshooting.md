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
