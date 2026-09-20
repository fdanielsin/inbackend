# Spécification non fonctionnelle — baseline actuelle

## Statut

`active` — exigences observées dans l'implémentation et le CI; les seuils
ci-dessous constituent la baseline actuelle, pas une promesse de production.

## Qualité et validation

- Le code doit être formaté avec `gofmt`.
- `go vet ./...`, `go build ./...` et `go test ./...` doivent passer.
- Les tests PostgreSQL doivent s'exécuter avec un provider Docker/Podman en
  CI ou lors d'une validation locale complète.
- Les tests sans provider doivent être explicitement signalés comme ignorés,
  jamais présentés comme des tests PostgreSQL exécutés.

## Observabilité

- Les logs sont structurés avec `slog`, délégué à `zap`.
- Les détails techniques restent dans les logs serveur.
- Les réponses HTTP 500 ne doivent pas exposer les erreurs internes.

## Sécurité et disponibilité

- L'API est actuellement ouverte; l'authentification et l'autorisation sont
  hors de la baseline actuelle.
- Le middleware Echo de récupération d'erreur et le health check sont activés.
- Aucun objectif chiffré de latence, disponibilité ou capacité n'est défini.
