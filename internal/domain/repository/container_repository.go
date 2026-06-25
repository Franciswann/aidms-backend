package repository

import "github.com/Franciswann/aidms-backend/internal/domain/entity"

type ContainerRepository interface {
	Save(container *entity.Container) error
	Update(container *entity.Container) error
	Delete(id string) error
	FindByID(id string) (*entity.Container, error)
	FindByUserID(userID string) ([]*entity.Container, error)
	FindByDockerID(dockerID string) (*entity.Container, error)
}
