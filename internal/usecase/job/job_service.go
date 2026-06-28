package job

import (
	"errors"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
)

var ErrForbidden = errors.New("you do not have access to this job")

type JobService struct {
	repo domainrepo.JobRepository
}

func NewJobService(repo domainrepo.JobRepository) *JobService {
	return &JobService{repo: repo}
}

func (s *JobService) Get(userID, jobID string) (*entity.Job, error) {
	j, err := s.repo.FindByID(jobID)
	if err != nil {
		return nil, err
	}
	if j.UserID != userID {
		return nil, ErrForbidden
	}
	return j, nil
}
