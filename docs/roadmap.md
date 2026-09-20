# Roadmap d'implémentation

## Phase 0 — état initial

- Lire le code actuel et les tests.
- Recenser les routes et les contrats de réponse.
- Ne pas modifier le frontend.

## Phase 1 — fondations du domaine

- **Statut : conforme après remise en conformité**
- Créer les packages `domain/user` et `domain/task`.
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
- Construire les dépendances dans `main.go`.
- Centraliser la conversion des erreurs HTTP.

## Phase 5 — validation

- **Statut : CI corrigée; intégration PostgreSQL ignorée localement sans Docker/Podman**
- Adapter les tests unitaires.
- Conserver et corriger les tests d'intégration.
- Exécuter les tests end-to-end.
- Exécuter `go test ./...` depuis `inbackend`.
- Vérifier le formatage et le build.

La validation PostgreSQL réelle reste la prochaine étape obligatoire dans un
environnement disposant d'un provider.

## Étape obligatoire avant la suite

Réaliser une remise en conformité de l'implémentation existante avant toute
nouvelle fonctionnalité :

- vérifier les écarts entre le code et l'architecture ;
- confirmer ou supprimer les doublons techniques ;
- clarifier les modèles métier, persistance et API ;
- clarifier le logging et la gestion des erreurs ;
- vérifier la stratégie de tests ;
- mettre à jour le handoff avec des faits vérifiés.

## Règle de progression

Chaque phase doit préserver le comportement existant. Toute modification
fonctionnelle ou décision nouvelle doit être ajoutée à `docs/decisions/`.
