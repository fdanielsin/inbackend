# État de reprise du backend

## Mission

Réorganiser `inbackend` selon l'architecture documentée, sans modifier
`infrontend` et sans casser les routes ou réponses HTTP existantes.

## État vérifié

Les packages suivants existent dans le code :

```text
internal/domain/
internal/application/
internal/controllers/
internal/infrastructure/
```

Le CQRS applicatif est présent avec des `commands` et des `queries`. Les
repositories PostgreSQL/GORM sont assemblés dans `main.go`.

Les validations locales exécutées depuis `inbackend` et vérifiables dans cet
état sont :

```text
make lint                                         → réussi (gofmt et go vet)
make test                                         → réussi ; 7 tests PostgreSQL ignorés sans provider
go test ./...                                     → réussi
go vet ./...                                      → réussi
go build ./...                                    → réussi
go test -count=1 -v ./tests/integration/persistence/... → réussi ; 7 tests ignorés
go test -count=1 -v ./tests/e2e/...               → réussi ; 8 tests exécutés
TESTCONTAINERS_RYUK_DISABLED=true DOCKER_HOST=... go test -count=1 -v ./tests/integration/persistence/... → réussi ; 7 tests PostgreSQL exécutés
TESTCONTAINERS_RYUK_DISABLED=true DOCKER_HOST=... go test -count=1 ./... → réussi avec PostgreSQL actif
```

Les tests d'intégration PostgreSQL sont explicitement ignorés lorsque
Testcontainers ne trouve aucun provider Docker/Podman local. Avec un provider
disponible, ils restent bloquants en cas d'échec. La validation avec Podman
ci-dessus constitue une preuve locale d'exécution PostgreSQL, sans constituer
à elle seule une preuve de préparation production.

Une première exécution sans socket explicite a produit
`rootless Docker not found, failed to create Docker provider`. La validation
réussie a ensuite utilisé le socket Podman macOS explicite et
`TESTCONTAINERS_RYUK_DISABLED=true`.

## Résultats de la remise en conformité

- Une branche dédiée `chore/backend-compliance-audit` est utilisée.
- Les modèles GORM sont maintenant dans
  `internal/infrastructure/persistence/postgres/models.go`; `APIResponse`
  reste dans `internal/shared/response.go`.
- Les comparaisons d'erreurs utilisent `errors.Is`.
- Les contrôleurs ne renvoient plus les messages d'erreur techniques bruts
  pour les erreurs 500; ils journalisent le détail et renvoient un message
  générique.
- La spécification API OpenAPI 3.1 est disponible dans
  `docs/specs/api/openapi.yaml` et couvre les neuf routes enregistrées.
- Les spécifications fonctionnelles, data, non fonctionnelles et de test sont
  indexées dans `docs/specs/`.
- Le logging `slog` délégué à `zap` est conservé comme choix explicite de la
  phase 6.

## Corrections confirmées

- Les jobs CI d'intégration et E2E ciblent désormais leurs packages respectifs.
- Le filtre `-run 'E2E'`, qui ne correspondait à aucun nom de test, a été
  supprimé afin d'éviter une réussite sans exécution.
- `make test` couvre maintenant `./...`, y compris les packages internes.
- Les références README à l'ancien chemin `internal/db` ont été corrigées.

## Contournement documenté

- Sans provider, les tests PostgreSQL sont ignorés explicitement.
- Dans l'environnement validé, le socket Podman macOS doit être fourni via
  `DOCKER_HOST` et `TESTCONTAINERS_RYUK_DISABLED=true` doit être activé.

## Validation CI distante

- L'API GitHub confirme que le run `35479583554` est `completed/success` pour
  `chore/backend-compliance-audit`, commit `5a76cb3`.
- Le run le plus récent observé est `35480322121`, également
  `completed/success`, pour le commit `a44832c` actuellement en tête de la
  branche.
- Ces résultats valident les commits distants concernés ; le diff documentaire
  local actuel reste non couvert par ces runs tant qu'il n'est pas committé.

## Audit du workspace et de Git

- Les seuls éléments à la racine sont `docs/`, `inbackend/` et `infrontend/`.
  `README.md`, `scripts/`, `docker-compose.yml` et `.github/` sont absents de
  la racine comme prévu.
- `inbackend` et `infrontend` sont deux dépôts Git indépendants avec leurs
  propres README et fichiers de projet. Aucun fichier frontend n'a été modifié.
- Le backend est sur `chore/backend-compliance-audit`; les modifications
  documentaires de cette reprise restent non committées.

## Classification de l'état

- **Corrigé et vérifié** : structure des couches, routes enregistrées,
  documentation indexée, formatage, vet, build, tests unitaires, E2E et
  PostgreSQL avec Podman, ainsi que les deux derniers runs CI distants observés.
- **Contournement documenté** : détection automatique du socket Podman ;
  la commande locale doit fournir `DOCKER_HOST` explicitement.
- **Non résolu** : les limites documentées de relation GORM, migrations
  versionnées, authentification et autorisation restent hors de cette mission.
- **Hors périmètre** : nouvelle fonctionnalité, modification du frontend,
  création de fichiers à la racine, commit ou push.

## Prochaine mission

La conformité locale est vérifiée pour les commandes exécutées ci-dessus,
y compris PostgreSQL avec Podman. Une nouvelle décision n'est nécessaire que
si l'architecture ou le workflow change.

## Documentation à lire

Un agent doit commencer par :

1. `../docs/README.md`
2. `../docs/context.md`
3. `../docs/workflow.md`
4. `../docs/documentation-guide.md`
5. `docs/README.md`
6. `docs/architecture.md`
7. `docs/roadmap.md`
8. `docs/decisions/`

## Règles de reprise

- Ne pas considérer un ancien handoff comme une preuve.
- Vérifier les affirmations dans le code, les tests et Git.
- Ne pas commencer une nouvelle fonctionnalité si une remise en conformité
  bloquante reste ouverte, sauf décision explicitement confirmée.
- Documenter les erreurs corrigées, contournées ou non résolues.
- Ne pas créer de nouveaux documents si un document canonique existe déjà.
- Ne pas créer de commit ou pousser sans autorisation explicite.

## Transmission

À la fin d'une tâche, indiquer :

```text
Branche :
Fichiers modifiés :
Corrigé :
Contourné temporairement :
Non résolu :
Tests exécutés :
Build :
Commit :
Push :
CI :
Décisions en attente :
Prochaine étape :
```

## Dernière mise à jour

2026-09-20 — reprise documentaire et technique vérifiée ; CI distante et
validation PostgreSQL locale avec Podman confirmées.
