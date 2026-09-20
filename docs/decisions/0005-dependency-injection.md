# ADR 0005 : Pattern d'Injection de Dépendances (main.go)

- **Statut** : accepté
- **Périmètre** : `inbackend`
- **Date** : 2026-09-20

## Contexte

Le backend utilisait initialement une configuration implicite et fragmentée des
dépendances. Chaque handler créait sa propre instance de repository. Cette
approche rend difficile le test et l'évolution.

L'injection de dépendances doit être centralisée mais rester simple et explicite.

## Décision

**Centraliser l'assemblage de TOUTES les dépendances dans `main.go`.**

### Structure d'assemblage

```go
main()
  ↓
InitDB(dsn) → *gorm.DB
  ↓
NewUserRepository(db) → UserRepository
NewTaskRepository(db) → TaskRepository
  ↓
NewCreateUserHandler(userRepo) → CreateUserHandler
NewGetUserHandler(userRepo) → GetUserHandler
... (repeat for all handlers)
  ↓
NewUserController(createUserHandler, getUserHandler, ...) → UserController
... (repeat for all controllers)
  ↓
echo.New() → *echo.Echo
register routes with wired controllers
  ↓
e.Start(":8080")
```

### Hiérarchie de construction

```
Database
  ↓
Repositories (depend on Database)
  ↓
Handlers (depend on Repositories)
  ↓
Controllers (depend on Handlers)
  ↓
Routes (depend on Controllers)
  ↓
Server (depends on Routes)
```

### Règles d'assemblage

1. **Une seule source de vérité** : `main.go` = point unique d'assemblage
2. **Pas de constructeur complexe** : Chaque composant a un `New*()` simple
3. **Pas de service locator** : Pas de global registry ou singleton factory
4. **Dépendances explicites** : Chaque composant déclare ses dépendances en paramètres
5. **Pas de cycles** : Domain → Application → Controllers → Infrastructure (DAG)

### Exemple réel de wiring

```go
// Database
db := InitDB(dsn, log)

// Repositories
userRepo := postgres.NewUserRepository(db)
taskRepo := postgres.NewTaskRepository(db)

// Handlers (Commands)
createUserHandler := usercommands.NewCreateUserHandler(userRepo)
createTaskHandler := commands.NewCreateTaskHandler(taskRepo)

// Handlers (Queries)
getUserHandler := userqueries.NewGetUserHandler(userRepo)
getStatsHandler := queries.NewGetStatsHandler(userRepo, taskRepo)

// Controllers
userCtrl := ctrlhttp.NewUserController(
  createUserHandler,
  getUserHandler,
  listUsersHandler,
)

// Routes
e.POST("/api/v1/users", userCtrl.CreateUser)
```

## Avantages

- **Testabilité** : Facile d'injecter des mocks dans les tests
- **Clarté** : Toutes les dépendances visibles en un seul fichier
- **Flexibilité** : Changer une implémentation = un seul endroit à modifier
- **Maintenabilité** : Nouveaux développeurs comprennent le flow en 5 min
- **Pas de magie** : Zero frameworks/decorators/annotations

## Inconvénients acceptés

- **Verbeux** : Beaucoup de lignes de wiring dans `main.go`
- **Pas de lazy loading** : Toutes les dépendances créées au démarrage
- **Durée de démarrage** : Plus de setup avant que le serveur soit prêt

## Alternatives rejetées

- **Dependency Injection frameworks** (Wire, dig, fx)
  - Ajoute complexité sans bénéfice pour cette taille de projet
  - Magie difficile à déboguer pour les nouveaux devs

- **Global singletons** (e.g., `var UserRepo = ...`)
  - Rend les tests difficiles (state partagé)
  - Couples les composants implicitement

- **Configuration files** (YAML, JSON)
  - Redondant quand Go code peut l'exprimer clairement
  - Ajoute fichiers externes à synchroniser

- **Package-level initialization** (init())
  - Ordre d'initialisation imprévisible
  - Bugs difficiles à reproduire

## Conséquences

### Pour les tests

Tests unitaires injectent les mocks directement :

```go
func TestCreateUserHandler(t *testing.T) {
  mockRepo := &MockUserRepository{}
  handler := commands.NewCreateUserHandler(mockRepo)
  // Test with mock
}
```

Tests d'intégration/E2E réutilisent l'assemblage mais avec SQLite :

```go
func TestUserController_CreateUser(t *testing.T) {
  db := setupTestDB(t)  // SQLite in-memory
  userRepo := postgres.NewUserRepository(db)
  // ... rest of wiring
  userCtrl := ctrlhttp.NewUserController(...)
  // Test via HTTP
}
```

### Pour l'évolution

Ajouter un nouveau feature :

1. Créer le handler dans `application/`
2. Créer le controller dans `controllers/http/`
3. Wirer dans `main.go`
4. Enregistrer la route dans `main.go`

Zéro boilerplate externe. Zéro annotations. Zéro magic.

## Décisions liées

- ADR 0001 : Architecture en 4 couches (Domain, Application, Controllers, Infrastructure)
- ADR 0002 : Code cleanup et gouvernance
- ADR 0004 : Test strategy (unit/integration/E2E avec Given/When/Then)

## Références

- [Dependency Injection in Go](https://www.golang-book.com/books/intro/13)
- [Functional Options Pattern](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis)
- `main.go` dans ce dépôt (implementation de référence)
