# Déploiement

Le service est un binaire Go statique, livré seul ou dans une image Docker.
Les variables d'environnement sont décrites dans le [README](../README.md).

## Binaire

```bash
make build
DATABASE_URL="postgres://user:pass@host:5432/inbackend?sslmode=require" ENV=production \
CORS_ALLOWED_ORIGINS=https://app.example.com ./app
```

Les migrations en attente sont appliquées au démarrage
([ADR 0007](./decisions/0007-versioned-migrations.md)). En production :

- `ENV=production` : logs JSON au niveau Info (texte au niveau Debug sinon) ;
- `CORS_ALLOWED_ORIGINS` : lister les origines autorisées, obligatoire quand `ENV` vaut exactement
  `production` : absente, vide ou contenant `*`, le serveur refuse de démarrer (`*` par défaut
  dans les autres cas). Chaque origine s'écrit `https://hôte[:port]` (ou `http://`) en
  minuscules, sans chemin, slash final, identifiants, query, fragment, port par défaut (80 pour
  http, 443 pour https) ni motif `https://*.example.com`. Un hôte dont le dernier label est
  numérique doit être une IPv4 canonique (quatre octets décimaux de 0 à 255, sans zéro de tête :
  `http://10.0.0.1`, pas `http://010.0.0.1`) ; une IPv6 mappée IPv4 est refusée. Rien n'est corrigé :
  le serveur refuse de démarrer en donnant la raison et le rang de l'entrée, compté sur les positions
  séparées par des virgules, éléments vides compris (`,, https://a.example, a.example` échoue à
  l'entrée 4), sans recopier la valeur ;
- `DATABASE_URL` : URL `postgres://` ou `postgresql://` (le format `clé=valeur`
  est refusé), avec `sslmode=require` si la base l'impose.

SIGINT et SIGTERM arrêtent le serveur proprement : 10 s pour les requêtes en
cours, puis fermeture du pool SQL.

## Mise à jour depuis la 1.0.0

Le serveur refuse de démarrer sur une base qui contient les tables `users` et
`tasks` de la 1.0.0 : copier d'abord les données avec la procédure SQL de
[`troubleshooting.md`](./troubleshooting.md).

## Image Docker

```bash
docker build -t inbackend:latest .
docker run -p 8080:8080 -e DATABASE_URL="postgres://user:pass@host:5432/inbackend?sslmode=disable" inbackend:latest
```

L'image est construite en deux étapes (`golang:1.26.8-alpine` puis
`alpine:3.24.2`), sans CGO, et s'exécute avec un utilisateur non root
(uid 10001). `make docker-build` et `make docker-run` utilisent Podman.

## Image publiée (GHCR)

Le job `publish-image` de la CI pousse l'image sur `ghcr.io/fdanielsin/inbackend`
(paquet privé), destinée au futur déploiement du backend, après un run vert, pour
chaque push sur `develop` et sur un tag `v*`, jamais depuis une PR, une autre
branche ni un `workflow_dispatch`. Tags : sha complet du commit, `develop` (push
sur la branche) et le nom du tag (push de tag). L'image est un manifeste unique
linux/amd64, sans attestation, et porte les labels OCI `source`, `title` et
`description`. Elle fait environ 37 Mo décompressée et 10 Mo compressée (mesure
locale `docker image ls` et `docker save | gzip`, arm64 ; la CI construit en
amd64, taille proche).

Le job n'utilise que le `GITHUB_TOKEN` du run avec `packages: write`, limité à
ce job et à `prune-image` ; le reste du workflow n'a que `contents: read`. Lire
l'image demande un jeton avec `read:packages` (ou l'accès en lecture au paquet,
réglages du paquet). Aucun autre dépôt ne la tire aujourd'hui : elle sert au
futur déploiement.

Pour la démarrer, il faut un PostgreSQL ; le contrat de ce commit se lit sur
`GET /openapi.yaml`.

| Variable | À donner | Remarque |
| --- | --- | --- |
| `DATABASE_URL` | URL du PostgreSQL (`sslmode=require` si la base l'impose) | obligatoire ; les migrations s'appliquent au démarrage |
| `PORT` | `8080` par défaut | le `HEALTHCHECK` de l'image suit cette variable |
| `CORS_ALLOWED_ORIGINS` | origines autorisées, séparées par des virgules | `*` par défaut hors production |
| `ENV` | `production` pour un déploiement, `development` par défaut | avec `production`, `/openapi.yaml` est coupé sauf `API_DOCS_ENABLED=true` et les origines CORS sont obligatoires |
| `API_DOCS_ENABLED` | `true` pour forcer | actif par défaut hors production |

Le `HEALTHCHECK` appelle `/ready`, donc l'image devient saine quand la base
répond. Pour figer le contrat, tirer le tag sha : `develop` change à chaque push.

### Coût et droits

- Le stockage et la bande passante du Container registry (`ghcr.io`) sont
  actuellement gratuits, et GitHub annonce un préavis d'un mois avant tout
  changement ([GitHub Packages,
  facturation](https://docs.github.com/en/billing/concepts/product-billing/github-packages)
  : « Container image storage and bandwidth for the Container registry is
  currently free. If you use Container registry, you'll be informed at least one
  month in advance of any change to this policy. »). La rétention reste utile par
  hygiène.
- La suppression de versions par un workflow (job `prune-image`) est une fonction
  en préversion publique, donc sujette à changement : « The ability for GitHub
  Actions workflows to delete and restore packages using the REST API is currently
  in public preview and subject to change. » ([suppression et restauration d'un
  paquet](https://docs.github.com/en/packages/learn-github-packages/deleting-and-restoring-a-package),
  même phrase sur la page [Working with the Container
  registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry),
  lues le 2026-10-03). Si l'API change, le job échoue et la suppression se fait à
  la main ; la publication de l'image n'en dépend pas.
- Le `GITHUB_TOKEN` peut supprimer des versions parce que le dépôt dont le workflow
  publie le paquet reçoit par défaut le rôle admin sur lui
  ([suppression et restauration d'un paquet](https://docs.github.com/en/packages/learn-github-packages/deleting-and-restoring-a-package)).
  Si le paquet est un jour recréé autrement, redonner ce rôle au dépôt dans les
  réglages du paquet.

### Ordre des publications

Le workflow porte `concurrency: group <workflow>-<ref>, cancel-in-progress: true` :
un push plus récent sur la même ref annule le run en cours (aussi pour une PR :
un nouveau push annule l'ancien run ; un `workflow_dispatch` sur la même ref aussi). Sans cela, deux
pushes rapprochés sur `develop` lanceraient deux runs complets, et celui du plus
ancien, s'il finissait en dernier, repousserait le tag `develop` sur une image
périmée. Une conséquence : le commit annulé n'a ni run complet ni image sha. Choix
retenu plutôt que de vérifier en fin de run que `github.sha` est encore la tête de
`develop` (plus de code, et une fenêtre de course restante).

### Rétention des versions

Le job `prune-image` suit `publish-image` et lance `scripts/prune-image-versions.sh`
(shell POSIX, `gh api --paginate` et `jq`). Il garde :

- toute version qui porte un tag `develop` ou `v*`, jamais supprimée ;
- les 10 versions taguées sha les plus récentes (`created_at` décroissant).

Il supprime le reste, versions sans tag comprises. Une version que `develop`
quitte garde son tag sha et compte parmi les 10 ; une version sans tag vient d'une
reconstruction du même commit qui déplace le tag sha (relance d'un run, tag `v*`
posé sur un commit déjà publié). L'image n'a pas d'attestation, donc aucune version
sans tag n'est l'enfant d'une version taguée.

Sans la variable de dépôt `PRUNE_IMAGE_VERSIONS=true` (réglages du dépôt, Actions,
Variables), le script n'affiche que ce qu'il supprimerait : lire cette liste dans
le journal d'un run réel avant d'activer la suppression. Un échec de `gh api`, de
liste ou de suppression, fait échouer le job.

Le propriétaire vient de `github.repository_owner` et le script appelle
`users/<propriétaire>/packages/container/inbackend/versions` : pour un dépôt
d'organisation, remplacer `users/` par `orgs/` dans le script. Essai sans GitHub :
`OWNER=x scripts/prune-image-versions.sh versions.json` lit un JSON de versions
(`id`, `created_at`, `metadata.container.tags`) et ne supprime jamais : avec
`PRUNE_IMAGE_VERSIONS=true` il refuse (exit 2), sinon il fait un dry-run sans appeler `gh`.

## Publier une version

`main` ne change que par une release. Le hook `.githooks/pre-push` (garde dans
`scripts/check-push-main.sh`, testée par `tests/units/githooks`) refuse tout push
vers `main`, suppression comprise, sauf si `RELEASE_TAG` nomme un tag qui pointe
sur le commit poussé. Activation, une fois par clone :
`git config core.hooksPath .githooks`.

```bash
git tag v1.2.0 <sha>                                   # tag sur le commit de la release
RELEASE_TAG=v1.2.0 git push origin <sha>:refs/heads/main
git push origin v1.2.0                                 # la CI publie l'image v1.2.0
```

Limites : le hook est local, s'active à la main et se contourne avec
`--no-verify` ; il ne voit ni un merge fait sur GitHub ni un autre poste (`main`
n'est pas protégée côté GitHub). La règle d'équipe reste : `main` ne change que
par une PR de release `develop` vers `main`, CI verte.

Un tag `v*` lance toute la CI puis `publish-image` : l'image reçoit les tags sha
complet et `v1.2.0` (pas `develop`). Le push de `main` lance la CI sans rien
publier. Aligner la date de la section du CHANGELOG, `info.version` du contrat et
la `version` de `GET /health` sur le tag avant de le poser.

## Docker Compose

```bash
docker compose up --build --wait   # PostgreSQL 16 et le backend, attend les healthchecks
curl http://localhost:8080/ready
docker compose down -v             # arrête et supprime le volume de la base
```

Compose utilise des identifiants de développement (`inbackend` / `inbackend`),
modifiables avec `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` et `PORT`.
`ENV` et `CORS_ALLOWED_ORIGINS` sont transmis au backend : avec `ENV=production`,
`CORS_ALLOWED_ORIGINS` doit lister les origines, sinon le backend ne démarre pas ;
vide ou absente, elle vaut `*` hors production.
Le job `docker` de la CI exécute cette même séquence. Compose est testé avec
Docker seulement ; le chemin Podman n'est pas testé.

## Contrat et Swagger UI

`GET /openapi.yaml` sert le contrat embarqué dans le binaire et `GET /docs` la
page Swagger UI. `API_DOCS_ENABLED` les active : par défaut oui, sauf avec
`ENV=production` ; `true` ou `false` force le choix, toute autre valeur fait
refuser le démarrage. Désactivées, ces routes répondent 404 `ROUTE_NOT_FOUND`.

`/docs` charge Swagger UI depuis jsDelivr à une version exacte, avec une
empreinte d'intégrité par fichier ; une `Content-Security-Policy` limite la page
à son origine et à ce CDN. Elle demande donc un accès à Internet côté navigateur
et n'est pas faite pour être exposée en production.

## Sondes

- `GET /health` : vivacité, sans interroger la base ;
- `GET /ready` : disponibilité, la base doit répondre en moins de 2 s, sinon 503
  `DATABASE_UNAVAILABLE`.

Le healthcheck de Compose utilise `/ready`. Les contrats sont dans
[`openapi.yaml`](./specs/api/openapi.yaml).
