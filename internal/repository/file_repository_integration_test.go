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
)

func TestFileRepository_SaveAndFindByID(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewFileRepository(tx)

	f := &entity.File{
		ID:        uuid.NewString(),
		UserID:    uuid.NewString(),
		Name:      "report.csv",
		Path:      "/uploads/some-user/some-uuid.csv",
		MimeType:  "text/csv",
		Size:      1024,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repo.Save(f))

	got, err := repo.FindByID(f.ID)
	require.NoError(t, err)
	assert.Equal(t, f.Name, got.Name)
	assert.Equal(t, f.Size, got.Size)
}

func TestFileRepository_FindByUserID(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewFileRepository(tx)

	userID := uuid.NewString()
	f1 := &entity.File{ID: uuid.NewString(), UserID: userID, Name: "a.csv", Path: "/a", MimeType: "text/csv", Size: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f2 := &entity.File{ID: uuid.NewString(), UserID: userID, Name: "b.json", Path: "/b", MimeType: "application/json", Size: 2, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, repo.Save(f1))
	require.NoError(t, repo.Save(f2))

	got, err := repo.FindByUserID(userID)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestFileRepository_Delete(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewFileRepository(tx)

	f := &entity.File{
		ID:        uuid.NewString(),
		UserID:    uuid.NewString(),
		Name:      "report.csv",
		Path:      "/uploads/some-user/some-uuid.csv",
		MimeType:  "text/csv",
		Size:      1024,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repo.Save(f))
	require.NoError(t, repo.Delete(f.ID))

	_, err := repo.FindByID(f.ID)
	assert.ErrorIs(t, err, domainrepo.ErrNotFound)
}

func TestFileRepository_FindByID_NotFound(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewFileRepository(tx)

	_, err := repo.FindByID(uuid.NewString())
	assert.ErrorIs(t, err, domainrepo.ErrNotFound)
}
