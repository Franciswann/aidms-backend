package job

import (
	"testing"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
	"github.com/stretchr/testify/require"
)

type mockJobRepository struct {
	byID map[string]*entity.Job
}

var _ domainrepo.JobRepository = (*mockJobRepository)(nil)

func (m *mockJobRepository) Save(j *entity.Job) error   { m.byID[j.ID] = j; return nil }
func (m *mockJobRepository) Update(j *entity.Job) error { m.byID[j.ID] = j; return nil }

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

func TestJobService_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockJobRepository{byID: map[string]*entity.Job{
			"job-1": {ID: "job-1", UserID: "user-1", Status: entity.JobStatusSuccess},
		}}
		svc := NewJobService(repo)

		j, err := svc.Get("user-1", "job-1")

		require.NoError(t, err)
		require.Equal(t, entity.JobStatusSuccess, j.Status)
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockJobRepository{byID: map[string]*entity.Job{}}
		svc := NewJobService(repo)

		_, err := svc.Get("user-1", "does-not-exist")

		require.ErrorIs(t, err, domainrepo.ErrNotFound)
	})

	t.Run("belongs to a different user", func(t *testing.T) {
		repo := &mockJobRepository{byID: map[string]*entity.Job{
			"job-1": {ID: "job-1", UserID: "owner"},
		}}
		svc := NewJobService(repo)

		_, err := svc.Get("someone-else", "job-1")

		require.ErrorIs(t, err, ErrForbidden)
	})
}
