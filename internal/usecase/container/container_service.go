package container

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
	"github.com/google/uuid"
)

var (
	// ErrForbidden means the container exists, but does not belong to the
	// requesting user.
	ErrForbidden = errors.New("you do not have access to this container")
)

type ContainerService struct {
	runtime domainrepo.ContainerRuntime
	repo    domainrepo.ContainerRepository
	jobRepo domainrepo.JobRepository
	wg      sync.WaitGroup
}

func NewContainerService(runtime domainrepo.ContainerRuntime, repo domainrepo.ContainerRepository, jobRepo domainrepo.JobRepository) *ContainerService {
	return &ContainerService{runtime: runtime, repo: repo, jobRepo: jobRepo}
}

// CreateAsync records a pending Job and starts the actual container creation
// in the background, returning immediately. The caller polls the Job by ID
// to find out when it finishes and whether it succeeded.
func (s *ContainerService) CreateAsync(userID, image, name string) (*entity.Job, error) {
	job := &entity.Job{
		ID:        uuid.NewString(),
		UserID:    userID,
		Action:    entity.JobActionCreate,
		Status:    entity.JobStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.jobRepo.Save(job); err != nil {
		return nil, err
	}

	// The goroutine gets its own copy to mutate. The caller's `job` is never
	// touched again after this point, so reading it back (e.g. to build an
	// HTTP response) is race-free - callers that want the latest state poll
	// via FindByID, which reads whatever the goroutine last persisted.
	jobCopy := *job
	s.wg.Add(1)
	go s.runCreateJob(&jobCopy, userID, image, name)

	return job, nil
}

func (s *ContainerService) runCreateJob(job *entity.Job, userID, image, name string) {
	defer s.wg.Done()

	job.Status = entity.JobStatusRunning
	job.UpdatedAt = time.Now()
	if err := s.jobRepo.Update(job); err != nil {
		log.Printf("job %s: failed to mark running: %v", job.ID, err)
	}

	ctr, err := s.Create(userID, image, name)
	if err != nil {
		job.Status = entity.JobStatusFailed
		job.ErrorMessage = err.Error()
	} else {
		job.Status = entity.JobStatusSuccess
		job.ContainerID = ctr.ID
	}
	job.UpdatedAt = time.Now()

	if err := s.jobRepo.Update(job); err != nil {
		log.Printf("job %s: failed to record final status: %v", job.ID, err)
	}
}

// Wait blocks until all in-flight background jobs finish. Intended for use
// during graceful shutdown so the process doesn't exit mid-job.
func (s *ContainerService) Wait() {
	s.wg.Wait()
}

func (s *ContainerService) Create(userID, image, name string) (*entity.Container, error) {
	dockerID, err := s.runtime.Create(image, name)
	if err != nil {
		return nil, err
	}

	c := &entity.Container{
		ID:        uuid.NewString(),
		UserID:    userID,
		DockerID:  dockerID,
		Name:      name,
		Image:     image,
		Status:    entity.ContainerStatusCreated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Save(c); err != nil {
		if rmErr := s.runtime.Remove(dockerID); rmErr != nil {
			log.Printf("failed to roll back orphaned container %s after save error %v: %v",
				dockerID, err, rmErr)
		}
		return nil, err
	}
	return c, nil
}

// getOwned looks up a container by ID and verifies it belongs to userID.
// Start/Stop/Delete all need this exact check, so it's factored out once.
func (s *ContainerService) getOwned(userID, containerID string) (*entity.Container, error) {
	c, err := s.repo.FindByID(containerID)
	if err != nil {
		return nil, err
	}
	if c.UserID != userID {
		return nil, ErrForbidden
	}
	return c, nil
}

func (s *ContainerService) Start(userID, containerID string) error {
	c, err := s.getOwned(userID, containerID)
	if err != nil {
		return err
	}

	if err := s.runtime.Start(c.DockerID); err != nil {
		return err
	}

	c.Status = entity.ContainerStatusRunning
	return s.repo.Update(c)
}

func (s *ContainerService) List(userID string) ([]*entity.Container, error) {
	return s.repo.FindByUserID(userID)
}

func (s *ContainerService) Get(userID, containerID string) (*entity.Container, error) {
	return s.getOwned(userID, containerID)
}

func (s *ContainerService) Stop(userID, containerID string) error {
	c, err := s.getOwned(userID, containerID)
	if err != nil {
		return err
	}

	if err = s.runtime.Stop(c.DockerID); err != nil {
		return err
	}
	c.Status = entity.ContainerStatusExited
	return s.repo.Update(c)
}

func (s *ContainerService) Delete(userID, containerID string) error {
	c, err := s.getOwned(userID, containerID)
	if err != nil {
		return err
	}

	if err := s.runtime.Remove(c.DockerID); err != nil {
		return err
	}

	return s.repo.Delete(c.ID)
}
