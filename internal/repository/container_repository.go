// container_repository.go
package repository

import (
	"errors"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
	"gorm.io/gorm"
)

var _ domainrepo.ContainerRepository = (*ContainerRepository)(nil)

type ContainerRepository struct {
	db *gorm.DB
}

func NewContainerRepository(db *gorm.DB) *ContainerRepository {
	return &ContainerRepository{db: db}
}

func (r *ContainerRepository) Save(container *entity.Container) error {
	model := ContainerFromDomain(container)
	return r.db.Create(model).Error
}

func (r *ContainerRepository) Update(container *entity.Container) error {
	model := ContainerFromDomain(container)
	return r.db.Save(model).Error
}

func (r *ContainerRepository) Delete(id string) error {
	return r.db.Delete(&ContainerModel{}, "id = ?", id).Error
}

func (r *ContainerRepository) FindByID(id string) (*entity.Container, error) {
	var model ContainerModel
	if err := r.db.Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainrepo.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *ContainerRepository) FindByDockerID(dockerID string) (*entity.Container, error) {
	var model ContainerModel
	if err := r.db.Where("docker_id = ?", dockerID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainrepo.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *ContainerRepository) FindByUserID(userID string) ([]*entity.Container, error) {
	var models []ContainerModel
	if err := r.db.Where("user_id = ?", userID).Find(&models).Error; err != nil {
		return nil, err
	}
	containers := make([]*entity.Container, len(models))
	for i, m := range models {
		containers[i] = m.ToDomain()
	}
	return containers, nil
}
