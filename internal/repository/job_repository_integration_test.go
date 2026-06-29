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

func TestJobRepository_SaveAndFindByID(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewJobRepository(tx)

	j := &entity.Job{
		ID:          uuid.NewString(),
		UserID:      uuid.NewString(),
		ContainerID: uuid.NewString(),
		Status:      entity.JobStatusPending,
		Action:      entity.JobActionCreate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(t, repo.Save(j))

	got, err := repo.FindByID(j.ID)
	require.NoError(t, err)
	assert.Equal(t, j.Status, got.Status)
	assert.Equal(t, j.Action, got.Action)
}

func TestJobRepository_Update(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewJobRepository(tx)

	j := &entity.Job{
		ID:          uuid.NewString(),
		UserID:      uuid.NewString(),
		ContainerID: uuid.NewString(),
		Status:      entity.JobStatusPending,
		Action:      entity.JobActionCreate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(t, repo.Save(j))

	j.Status = entity.JobStatusSuccess
	require.NoError(t, repo.Update(j))

	got, err := repo.FindByID(j.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.JobStatusSuccess, got.Status)
}

func TestJobRepository_FindByUserID(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewJobRepository(tx)

	userID := uuid.NewString()
	j1 := &entity.Job{ID: uuid.NewString(), UserID: userID, ContainerID: uuid.NewString(), Status: entity.JobStatusPending, Action: entity.JobActionCreate, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	j2 := &entity.Job{ID: uuid.NewString(), UserID: userID, ContainerID: uuid.NewString(), Status: entity.JobStatusRunning, Action: entity.JobActionStart, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, repo.Save(j1))
	require.NoError(t, repo.Save(j2))

	got, err := repo.FindByUserID(userID)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestJobRepository_FindByID_NotFound(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewJobRepository(tx)

	_, err := repo.FindByID(uuid.NewString())
	assert.ErrorIs(t, err, domainrepo.ErrNotFound)
}
