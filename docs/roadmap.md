# Roadmap d'implémentation

## Phase 0 — état initial

- Lire le code actuel et les tests.
- Recenser les routes et les contrats de réponse.
- Ne pas modifier le frontend.

## Phase 1 — fondations du domaine

- **Statut : conforme après remise en conformité**
- Créer les packages `domain/user` et `domain/task` (renommés `users` et `tasks`, ADR 0008).
- Définir les entités et les erreurs métier.
- Définir les interfaces de repository.
- Ajouter les règles métier réellement identifiées.

## Phase 2 — application et CQRS

- **Statut : conforme après remise en conformité**
- Créer les commands utilisateur et tâche.
- Créer les queries utilisateur et tâche.
- Déplacer l'orchestration hors des handlers HTTP.
- Tester les use cases avec des repositories de test.

## Phase 3 — infrastructure

- **Statut : conforme après remise en conformité**
- Créer les modèles PostgreSQL/GORM.
- Implémenter les repositories.
- Isoler la connexion et les migrations.
- Conserver les transactions PostgreSQL locales.

## Phase 4 — controllers et composition

- **Statut : conforme après remise en conformité**
- Remplacer les handlers monolithiques par des controllers HTTP.
- Conserver les routes existantes.
- Construire les dépendances dans `internal/app` (`app.NewRouter`, lot 3 ; `main.go` ne garde que la configuration).
- Centraliser la conversion des erreurs HTTP.

## Phase 5 — validation

- **Statut : CI corrigée ; intégration PostgreSQL exécutée avec succès via
  Docker/Podman pour le lot 1 (2026-09-29, voir Lots de l'audit) ; sans provider,
  elle reste ignorée explicitement, jamais réussie par défaut.**
- Adapter les tests unitaires.
- Conserver et corriger les tests d'intégration.
- Exécuter les tests end-to-end.
- Exécuter `go test ./...` depuis `inbackend`.
- Vérifier le formatage et le build.

## Lots de l'audit du 2026-09-29 (backend)

Le détail des identifiants et le plan complet (lots frontend et QA compris) sont
dans `docs/roadmap.md` à la racine du workspace. Ici, l'état du backend à la
release 1.2.0 (voir le [CHANGELOG](../CHANGELOG.md)) :

- **Fait et mergé dans `develop`** : CI verte (lot 0) ; `Q-01` (mise à jour
  partielle d'une tâche) ; `B-03` et `B-04` (409 `EMAIL_TAKEN`, 404
  `TASK_NOT_FOUND`) ; `Q-06`, `Q-09`, `Q-10`, `Q-12` (tests honnêtes, tri par
  `id`) ; `B-05` et `B-02` (migrations versionnées, clé étrangère, ADR 0007) ;
  `B-08` (plus de donnée personnelle dans les logs) ; `Q-04`, `Q-05`, `Q-11`,
  `B-06` (couverture minimale, lint, govulncheck, Redocly, hook pre-push) ;
  `Q-03` (Docker et compose).
- **Fait hors audit** : sondes `/health` et `/ready`, logs `slog` seul, messages
  français et anglais, contrat et Swagger UI servis par l'API, corps de requête
  validés, CORS obligatoire en production avec origines validées, paquets
  renommés (ADR 0008), lint
  strict, image publiée sur GHCR, hook pre-push qui protège `main`.
- **À faire ou à décider** : `B-07` (pagination de `ListUsers` et
  `ListUserTasks`, décision de contrat ouverte) ; `Q-13` ; lot 8 des
  fonctionnalités (`FP-01` à `FP-04`, filtre `?completed=` `BP-02`), sur
  décision utilisateur.

## Règle de progression

Chaque phase doit préserver le comportement existant. Toute modification
fonctionnelle ou décision nouvelle doit être ajoutée à `docs/decisions/`.
