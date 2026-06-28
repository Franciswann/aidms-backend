package container

import (
	"errors"
	"log"
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
}

func NewContainerService(runtime domainrepo.ContainerRuntime, repo domainrepo.ContainerRepository) *ContainerService {
	return &ContainerService{runtime: runtime, repo: repo}
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
