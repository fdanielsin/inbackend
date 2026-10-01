# Déploiement

Le service est un binaire Go statique, livré seul ou dans une image Docker.
Les variables d'environnement sont décrites dans le [README](../README.md).

## Binaire

```bash
make build
DATABASE_URL="postgres://user:pass@host:5432/inbackend?sslmode=require" ENV=production ./app
```

Les migrations en attente sont appliquées au démarrage
([ADR 0007](./decisions/0007-versioned-migrations.md)). En production :

- `ENV=production` : logs JSON au niveau Info (texte au niveau Debug sinon) ;
- `CORS_ALLOWED_ORIGINS` : lister les origines autorisées (`*` par défaut) ;
- `DATABASE_URL` : URL `postgres://` ou `postgresql://` (le format `clé=valeur`
  est refusé), avec `sslmode=require` si la base l'impose.

SIGINT et SIGTERM arrêtent le serveur proprement : 10 s pour les requêtes en
cours, puis fermeture du pool SQL.

## Image Docker

```bash
docker build -t inbackend:latest .
docker run -p 8080:8080 -e DATABASE_URL="postgres://user:pass@host:5432/inbackend?sslmode=disable" inbackend:latest
```

L'image est construite en deux étapes (`golang:1.26.8-alpine` puis
`alpine:3.24.2`), sans CGO, et s'exécute avec un utilisateur non root
(uid 10001). `make docker-build` et `make docker-run` utilisent Podman.

## Docker Compose

```bash
docker compose up --build --wait   # PostgreSQL 16 et le backend, attend les healthchecks
curl http://localhost:8080/ready
docker compose down -v             # arrête et supprime le volume de la base
```

Compose utilise des identifiants de développement (`inbackend` / `inbackend`),
modifiables avec `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` et `PORT`.
Le job `docker` de la CI exécute cette même séquence. Compose est testé avec
Docker seulement ; le chemin Podman n'est pas testé.

## Sondes

- `GET /health` : vivacité, sans interroger la base ;
- `GET /ready` : disponibilité, la base doit répondre en moins de 2 s, sinon 503
  `DATABASE_UNAVAILABLE`.

Le healthcheck de Compose utilise `/ready`. Les contrats sont dans
[`openapi.yaml`](./specs/api/openapi.yaml).
