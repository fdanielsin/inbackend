package persistence

import (
	"context"
	"fmt"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/users"
	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/pgstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// GIVEN: UserRepository / WHEN: Create valid user / THEN: User saved with ID
func TestUserRepository_Create_HappyPath(t *testing.T) {
	db := setupTestDB(t)
	repo := pgstore.NewUserRepository(db)

	u := &users.User{Name: "John Doe", Email: "john@example.com"}
	id, err := repo.Create(context.Background(), u)

	require.NoError(t, err)
	assert.NotZero(t, id)

	// THEN: Verify user was persisted
	retrieved, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "John Doe", retrieved.Name)
	assert.Equal(t, "john@example.com", retrieved.Email)
}

// GIVEN: UserRepository / WHEN: Get non-existent user / THEN: Error ErrUserNotFound
func TestUserRepository_GetByID_SadPath_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := pgstore.NewUserRepository(db)

	retrieved, err := repo.GetByID(context.Background(), 999)

	require.Error(t, err)
	assert.Nil(t, retrieved)
	assert.Equal(t, users.ErrUserNotFound, err)
}

// GIVEN: UserRepository with multiple users / WHEN: Get by email / THEN: Return matching user
func TestUserRepository_GetByEmail_HappyPath(t *testing.T) {
	db := setupTestDB(t)
	repo := pgstore.NewUserRepository(db)

	u1 := &users.User{Name: "John", Email: "john@example.com"}
	u2 := &users.User{Name: "Jane", Email: "jane@example.com"}

	_, err := repo.Create(context.Background(), u1)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), u2)
	require.NoError(t, err)

	// THEN: Find by email
	found, err := repo.GetByEmail(context.Background(), "jane@example.com")
	require.NoError(t, err)
	assert.Equal(t, "Jane", found.Name)
	assert.Equal(t, "jane@example.com", found.Email)
}

// GIVEN: UserRepository / WHEN: Get by non-existent email / THEN: ErrUserNotFound
func TestUserRepository_GetByEmail_SadPath_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := pgstore.NewUserRepository(db)

	found, err := repo.GetByEmail(context.Background(), "nonexistent@example.com")

	require.ErrorIs(t, err, users.ErrUserNotFound)
	assert.Nil(t, found)
}

// createUsers stores count users whose names and emails sort in the reverse order of their creation,
// so that an order by name or email cannot pass for the order by id. It returns the ids in creation order.
func createUsers(t *testing.T, repo *pgstore.UserRepository, count int) []uint {
	t.Helper()
	ids := make([]uint, count)
	for i := range ids {
		id, err := repo.Create(context.Background(), &users.User{
			Name:  fmt.Sprintf("User %03d", count-i),
			Email: fmt.Sprintf("u%03d@example.com", count-i),
		})
		require.NoError(t, err)
		ids[i] = id
	}
	return ids
}

// moveToHeapEnd rewrites the row of id: PostgreSQL (MVCC) writes the new row version after the
// others, so a sequential scan without ORDER BY returns it last.
func moveToHeapEnd(t *testing.T, db *gorm.DB, id uint) {
	t.Helper()
	require.NoError(t, db.Exec("UPDATE user_models SET name = name WHERE id = ?", id).Error)
}

// requirePhysicalOrderDiffers proves the setup: read without ORDER BY, the rows come in another order
// than want, so the test can only pass thanks to the ORDER BY of List.
func requirePhysicalOrderDiffers(t *testing.T, db *gorm.DB, want []uint) {
	t.Helper()
	var physical []uint
	require.NoError(t, db.Raw("SELECT id FROM user_models").Scan(&physical).Error)
	require.ElementsMatch(t, want, physical)
	require.NotEqual(t, want, physical, "the setup must change the physical order")
}

func listedIDs(t *testing.T, repo *pgstore.UserRepository) []uint {
	t.Helper()
	listed, err := repo.List(context.Background())
	require.NoError(t, err)
	ids := make([]uint, len(listed))
	for i, u := range listed {
		ids[i] = u.ID
	}
	return ids
}

// GIVEN: Three users, the first one rewritten after the others / WHEN: List
// THEN: Ascending ids, not the physical order (B-07 decided: id order, stable)
func TestUserRepository_List_OrderedByIDAfterUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := pgstore.NewUserRepository(db)
	ids := createUsers(t, repo, 3)
	moveToHeapEnd(t, db, ids[0])
	requirePhysicalOrderDiffers(t, db, ids)

	assert.Equal(t, ids, listedIDs(t, repo))
}

// GIVEN: Exactly two users, physically swapped / WHEN: List / THEN: Ascending ids
func TestUserRepository_List_TwoUsersPhysicallySwapped(t *testing.T) {
	db := setupTestDB(t)
	repo := pgstore.NewUserRepository(db)
	ids := createUsers(t, repo, 2)
	moveToHeapEnd(t, db, ids[0])
	requirePhysicalOrderDiffers(t, db, ids)

	assert.Equal(t, ids, listedIDs(t, repo))
}

// GIVEN: Thirty users, several rewritten in a mixed order / WHEN: List / THEN: Strictly ascending ids
func TestUserRepository_List_ManyUsersStayOrdered(t *testing.T) {
	db := setupTestDB(t)
	repo := pgstore.NewUserRepository(db)
	ids := createUsers(t, repo, 30)
	for _, position := range []int{17, 3, 25, 0, 9} {
		moveToHeapEnd(t, db, ids[position])
	}
	requirePhysicalOrderDiffers(t, db, ids)

	assert.Equal(t, ids, listedIDs(t, repo))
}

// GIVEN: Three users, the middle one deleted and the first rewritten / WHEN: List
// THEN: The two remaining ids in ascending order, the gap kept
func TestUserRepository_List_KeepsGapsInIDs(t *testing.T) {
	db := setupTestDB(t)
	repo := pgstore.NewUserRepository(db)
	ids := createUsers(t, repo, 3)
	require.NoError(t, db.Exec("DELETE FROM user_models WHERE id = ?", ids[1]).Error)
	moveToHeapEnd(t, db, ids[0])
	remaining := []uint{ids[0], ids[2]}
	requirePhysicalOrderDiffers(t, db, remaining)

	assert.Equal(t, remaining, listedIDs(t, repo))
}
