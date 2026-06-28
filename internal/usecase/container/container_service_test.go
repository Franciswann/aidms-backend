package container

import (
	"errors"
	"testing"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockContainerRuntime struct {
	createID    string
	createErr   error
	startErr    error
	stopErr     error
	removeErr   error
	removeCalls []string
}

var _ domainrepo.ContainerRuntime = (*mockContainerRuntime)(nil)

func (m *mockContainerRuntime) Create(image, name string) (string, error) {
	if m.createErr != nil {
		return "", m.createErr
	}
	return m.createID, nil
}

func (m *mockContainerRuntime) Start(dockerID string) error { return m.startErr }
func (m *mockContainerRuntime) Stop(dockerID string) error  { return m.stopErr }

func (m *mockContainerRuntime) Remove(dockerID string) error {
	m.removeCalls = append(m.removeCalls, dockerID)
	return m.removeErr
}

type mockContainerRepository struct {
	byID          map[string]*entity.Container
	saveErr       error
	updateErr     error
	deleteErr     error
	saveCallCount int
}

var _ domainrepo.ContainerRepository = (*mockContainerRepository)(nil)

func newMockContainerRepository() *mockContainerRepository {
	return &mockContainerRepository{byID: make(map[string]*entity.Container)}
}

func (m *mockContainerRepository) Save(c *entity.Container) error {
	m.saveCallCount++
	if m.saveErr != nil {
		return m.saveErr
	}
	m.byID[c.ID] = c
	return nil
}

func (m *mockContainerRepository) Update(c *entity.Container) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.byID[c.ID] = c
	return nil
}

func (m *mockContainerRepository) Delete(id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.byID, id)
	return nil
}

func (m *mockContainerRepository) FindByID(id string) (*entity.Container, error) {
	c, ok := m.byID[id]
	if !ok {
		return nil, domainrepo.ErrNotFound
	}
	return c, nil
}

func (m *mockContainerRepository) FindByDockerID(dockerID string) (*entity.Container, error) {
	for _, c := range m.byID {
		if c.DockerID == dockerID {
			return c, nil
		}
	}
	return nil, domainrepo.ErrNotFound
}

func (m *mockContainerRepository) FindByUserID(userID string) ([]*entity.Container, error) {
	var out []*entity.Container
	for _, c := range m.byID {
		if c.UserID == userID {
			out = append(out, c)
		}
	}
	return out, nil
}

type mockJobRepository struct {
	byID map[string]*entity.Job
}

var _ domainrepo.JobRepository = (*mockJobRepository)(nil)

func newMockJobRepository() *mockJobRepository {
	return &mockJobRepository{byID: make(map[string]*entity.Job)}
}

func (m *mockJobRepository) Save(j *entity.Job) error {
	m.byID[j.ID] = j
	return nil
}

func (m *mockJobRepository) Update(j *entity.Job) error {
	m.byID[j.ID] = j
	return nil
}

func (m *mockJobRepository) FindByID(id string) (*entity.Job, error) {
	j, ok := m.byID[id]
	if !ok {
		return nil, domainrepo.ErrNotFound
	}
	return j, nil
}

func (m *mockJobRepository) FindByUserID(userID string) ([]*entity.Job, error) {
	var out []*entity.Job
	for _, j := range m.byID {
		if j.UserID == userID {
			out = append(out, j)
		}
	}
	return out, nil
}

var errSimulated = errors.New("simulated failure")

func TestContainerService_CreateAsync(t *testing.T) {
	t.Run("success - job ends up succeeded with the new container's ID", func(t *testing.T) {
		rt := &mockContainerRuntime{createID: "docker-789"}
		repo := newMockContainerRepository()
		jobRepo := newMockJobRepository()
		svc := NewContainerService(rt, repo, jobRepo)

		j, err := svc.CreateAsync("user-1", "alpine:latest", "my-container")
		require.NoError(t, err)
		require.NotNil(t, j)
		assert.Equal(t, entity.JobStatusPending, j.Status)

		svc.Wait()

		final, err := jobRepo.FindByID(j.ID)
		require.NoError(t, err)
		assert.Equal(t, entity.JobStatusSuccess, final.Status)
		assert.NotEmpty(t, final.ContainerID)
	})

	t.Run("runtime create fails - job ends up failed with the error recorded", func(t *testing.T) {
		rt := &mockContainerRuntime{createErr: errSimulated}
		repo := newMockContainerRepository()
		jobRepo := newMockJobRepository()
		svc := NewContainerService(rt, repo, jobRepo)

		j, err := svc.CreateAsync("user-1", "alpine:latest", "my-container")
		require.NoError(t, err)

		svc.Wait()

		final, err := jobRepo.FindByID(j.ID)
		require.NoError(t, err)
		assert.Equal(t, entity.JobStatusFailed, final.Status)
		assert.Equal(t, errSimulated.Error(), final.ErrorMessage)
	})
}

func TestContainerService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		rt := &mockContainerRuntime{createID: "docker-123"}
		repo := newMockContainerRepository()
		svc := NewContainerService(rt, repo, newMockJobRepository())

		c, err := svc.Create("user-1", "alpine:latest", "my-container")

		require.NoError(t, err)
		assert.Equal(t, "docker-123", c.DockerID)
		assert.Equal(t, "user-1", c.UserID)
		assert.Equal(t, entity.ContainerStatusCreated, c.Status)
		assert.Equal(t, 1, repo.saveCallCount)
	})

	t.Run("runtime create fails - never touches the repository", func(t *testing.T) {
		rt := &mockContainerRuntime{createErr: errSimulated}
		repo := newMockContainerRepository()
		svc := NewContainerService(rt, repo, newMockJobRepository())

		c, err := svc.Create("user-1", "alpine:latest", "my-container")

		require.ErrorIs(t, err, errSimulated)
		assert.Nil(t, c)
		assert.Equal(t, 0, repo.saveCallCount)
	})

	t.Run("save fails - rolls back the orphaned docker container", func(t *testing.T) {
		rt := &mockContainerRuntime{createID: "docker-456"}
		repo := newMockContainerRepository()
		repo.saveErr = errSimulated
		svc := NewContainerService(rt, repo, newMockJobRepository())

		c, err := svc.Create("user-1", "alpine:latest", "my-container")

		require.ErrorIs(t, err, errSimulated)
		assert.Nil(t, c)
		// the original Save error must be what's returned, not swallowed by
		// the rollback - and the rollback must actually target the right ID
		require.Len(t, rt.removeCalls, 1)
		assert.Equal(t, "docker-456", rt.removeCalls[0])
	})
}

func TestContainerService_Start(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		rt := &mockContainerRuntime{}
		repo := newMockContainerRepository()
		repo.byID["c-1"] = &entity.Container{ID: "c-1", UserID: "user-1", DockerID: "docker-1", Status: entity.ContainerStatusCreated}
		svc := NewContainerService(rt, repo, newMockJobRepository())

		err := svc.Start("user-1", "c-1")

		require.NoError(t, err)
		assert.Equal(t, entity.ContainerStatusRunning, repo.byID["c-1"].Status)
	})

	t.Run("container not found", func(t *testing.T) {
		rt := &mockContainerRuntime{}
		repo := newMockContainerRepository()
		svc := NewContainerService(rt, repo, newMockJobRepository())

		err := svc.Start("user-1", "does-not-exist")

		require.ErrorIs(t, err, domainrepo.ErrNotFound)
	})

	t.Run("container belongs to a different user", func(t *testing.T) {
		rt := &mockContainerRuntime{}
		repo := newMockContainerRepository()
		repo.byID["c-1"] = &entity.Container{ID: "c-1", UserID: "owner", DockerID: "docker-1"}
		svc := NewContainerService(rt, repo, newMockJobRepository())

		err := svc.Start("someone-else", "c-1")

		require.ErrorIs(t, err, ErrForbidden)
	})
}

func TestContainerService_Stop(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		rt := &mockContainerRuntime{}
		repo := newMockContainerRepository()
		repo.byID["c-1"] = &entity.Container{ID: "c-1", UserID: "user-1", DockerID: "docker-1", Status: entity.ContainerStatusRunning}
		svc := NewContainerService(rt, repo, newMockJobRepository())

		err := svc.Stop("user-1", "c-1")

		require.NoError(t, err)
		assert.Equal(t, entity.ContainerStatusExited, repo.byID["c-1"].Status)
	})
}

func TestContainerService_Delete(t *testing.T) {
	t.Run("success - removes from both docker and the repository", func(t *testing.T) {
		rt := &mockContainerRuntime{}
		repo := newMockContainerRepository()
		repo.byID["c-1"] = &entity.Container{ID: "c-1", UserID: "user-1", DockerID: "docker-1"}
		svc := NewContainerService(rt, repo, newMockJobRepository())

		err := svc.Delete("user-1", "c-1")

		require.NoError(t, err)
		require.Len(t, rt.removeCalls, 1)
		assert.Equal(t, "docker-1", rt.removeCalls[0])
		_, stillExists := repo.byID["c-1"]
		assert.False(t, stillExists)
	})

	t.Run("runtime remove fails - repository record is kept, not deleted", func(t *testing.T) {
		rt := &mockContainerRuntime{removeErr: errSimulated}
		repo := newMockContainerRepository()
		repo.byID["c-1"] = &entity.Container{ID: "c-1", UserID: "user-1", DockerID: "docker-1"}
		svc := NewContainerService(rt, repo, newMockJobRepository())

		err := svc.Delete("user-1", "c-1")

		require.ErrorIs(t, err, errSimulated)
		_, stillExists := repo.byID["c-1"]
		assert.True(t, stillExists)
	})
}
