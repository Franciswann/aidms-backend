//go:build integration

package repository

import (
	"testing"
	"time"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newIntegrationTestDB opens a connection to the local docker-compose Postgres
// and makes sure the schema exists. Each test gets its own transaction (see
// tests below) so they never need to clean up rows themselves.
func newIntegrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "host=localhost port=5432 user=aidms password=aidms_secret dbname=aidms_db sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&UserModel{}, &ContainerModel{}, &FileModel{}, &JobModel{}))
	return db
}

func TestUserRepository_SaveAndFindByID(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewUserRepository(tx)

	user := &entity.User{
		ID:             uuid.NewString(),
		Email:          "francis@example.com",
		HashedPassword: "hashed-password",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.Save(user))

	got, err := repo.FindByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.Email, got.Email)
	assert.Equal(t, user.HashedPassword, got.HashedPassword)
}

func TestUserRepository_FindByEmail(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewUserRepository(tx)

	user := &entity.User{
		ID:             uuid.NewString(),
		Email:          "francis-by-email@example.com",
		HashedPassword: "hashed-password",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.Save(user))

	got, err := repo.FindByEmail(user.Email)
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.ID)
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewUserRepository(tx)

	_, err := repo.FindByID(uuid.NewString())
	assert.ErrorIs(t, err, domainrepo.ErrNotFound)
}

func TestUserRepository_FindByEmail_NotFound(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewUserRepository(tx)

	_, err := repo.FindByEmail("nobody@example.com")
	assert.ErrorIs(t, err, domainrepo.ErrNotFound)
}
