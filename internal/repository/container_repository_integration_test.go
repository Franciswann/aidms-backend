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

func TestContainerRepository_SaveAndFindByID(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewContainerRepository(tx)

	c := &entity.Container{
		ID:        uuid.NewString(),
		UserID:    uuid.NewString(),
		DockerID:  uuid.NewString(),
		Name:      "test-container",
		Image:     "nginx:latest",
		Status:    entity.ContainerStatusCreated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repo.Save(c))

	got, err := repo.FindByID(c.ID)
	require.NoError(t, err)
	assert.Equal(t, c.Name, got.Name)
	assert.Equal(t, c.Status, got.Status)
}

func TestContainerRepository_FindByDockerID(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewContainerRepository(tx)

	c := &entity.Container{
		ID:        uuid.NewString(),
		UserID:    uuid.NewString(),
		DockerID:  uuid.NewString(),
		Name:      "test-container",
		Image:     "nginx:latest",
		Status:    entity.ContainerStatusCreated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repo.Save(c))

	got, err := repo.FindByDockerID(c.DockerID)
	require.NoError(t, err)
	assert.Equal(t, c.ID, got.ID)
}

func TestContainerRepository_FindByUserID(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewContainerRepository(tx)

	userID := uuid.NewString()
	c1 := &entity.Container{ID: uuid.NewString(), UserID: userID, DockerID: uuid.NewString(), Name: "c1", Image: "nginx", Status: entity.ContainerStatusCreated, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	c2 := &entity.Container{ID: uuid.NewString(), UserID: userID, DockerID: uuid.NewString(), Name: "c2", Image: "nginx", Status: entity.ContainerStatusCreated, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, repo.Save(c1))
	require.NoError(t, repo.Save(c2))

	got, err := repo.FindByUserID(userID)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestContainerRepository_Update(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewContainerRepository(tx)

	c := &entity.Container{
		ID:        uuid.NewString(),
		UserID:    uuid.NewString(),
		DockerID:  uuid.NewString(),
		Name:      "test-container",
		Image:     "nginx:latest",
		Status:    entity.ContainerStatusCreated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repo.Save(c))

	c.Status = entity.ContainerStatusRunning
	require.NoError(t, repo.Update(c))

	got, err := repo.FindByID(c.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.ContainerStatusRunning, got.Status)
}

func TestContainerRepository_Delete(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewContainerRepository(tx)

	c := &entity.Container{
		ID:        uuid.NewString(),
		UserID:    uuid.NewString(),
		DockerID:  uuid.NewString(),
		Name:      "test-container",
		Image:     "nginx:latest",
		Status:    entity.ContainerStatusCreated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repo.Save(c))
	require.NoError(t, repo.Delete(c.ID))

	_, err := repo.FindByID(c.ID)
	assert.ErrorIs(t, err, domainrepo.ErrNotFound)
}

func TestContainerRepository_FindByID_NotFound(t *testing.T) {
	db := newIntegrationTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	repo := NewContainerRepository(tx)

	_, err := repo.FindByID(uuid.NewString())
	assert.ErrorIs(t, err, domainrepo.ErrNotFound)
}
