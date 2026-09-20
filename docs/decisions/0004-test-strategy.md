# ADR 0004 : Stratégie de tests par catégories (Unit, Integration, E2E)

- **Statut** : accepté
- **Périmètre** : `inbackend`
- **Date** : 2026-09-20

## Contexte

Le backend actuel manquait d'une structure claire pour les tests. Les tests
étaient fragmentés, sans catégorisation, sans conventions de nommage, et sans
documentation d'intent explicit.

Cela ralentit les développeurs et rend difficile l'ajout de nouveaux tests
suivant les mêmes patterns.

## Décision

Structurer TOUS les tests autour de trois catégories :

1. **Unit** (~5ms)  
   - Logique métier pure, pas d'I/O
   - Fichiers d'entités de domaine
   - Location : `tests/units/domain/`

2. **Integration** (~100ms)  
   - Tests de contracts de repository
   - Contre une vraie base de données (SQLite en local, PostgreSQL en CI)
   - Location : `tests/integration/persistence/`

3. **E2E** (~500ms)  
   - Flux HTTP complet : controller → repo → DB
   - Contre testcontainers (Podman/Docker)
   - Location : `tests/e2e/`

### Structure des répertoires

```
tests/
├── units/
│   └── domain/              # Entity tests
│
├── integration/
│   └── persistence/         # Repository tests + shared fixtures
│
└── e2e/
    └── controllers_test.go  # HTTP endpoint tests
```

### Convention de nommage

**Format des tests :**
- `Test<Component>_<Scenario>_<Result>`
- Exemples :
  - `TestUserRepository_Create_HappyPath` — cas nominal
  - `TestUserEntity_NewUser_EmptyName` — validation échouée
  - `TestUserRepository_GetByID_SadPath_NotFound` — erreur attendue

**Fichiers :**
- `*_entity_test.go` → tests d'entités domaine
- `*_repository_test.go` → tests de repository
- `setup_test.go` → fixtures partagées

### Given/When/Then

Tous les tests suivent cette structure explicite :

```go
func TestX_Y_Z(t *testing.T) {
	// GIVEN: Préparation et préconditions
	
	// WHEN: Exécution de l'action testée
	
	// THEN: Vérification des résultats
}
```

### Chemins heureux et tristes

- **Happy path** : comportement normal attendu  
  `TestUserRepository_Create_HappyPath`
  
- **Sad path** : conditions d'erreur, cas limites  
  `TestUserRepository_GetByID_SadPath_NotFound`

## Conséquences positives

- **Clarté** : intention du test évidente par le nom et la structure
- **Maintenabilité** : fixture partagée au lieu de duplication
- **Feedback rapide** : catégories permettent `make test-unit` en ~1s
- **Couverture** : good path + sad path explicites et nommés
- **Isolation** : chaque test a sa propre DB fraîche

## Conséquences acceptées

- Plus de fichiers (un par composant testé)
- Fixtures partagées nécessitent coordination
- Tests E2E nécessitent Podman/Docker

## Alternatives rejetées

- Tests colocalisés dans `internal/` (difficile à naviguer)
- Mocks complexes de repository (on teste contre vraie DB)
- Convention unique "TestCreate" sans contexte (pas claire)
- Sauter les sad paths (pas de garantie d'erreur handling)
