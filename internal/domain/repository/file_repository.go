package repository

import "github.com/Franciswann/aidms-backend/internal/domain/entity"

type FileRepository interface {
	Save(file *entity.File) error
	Delete(id string) error
	FindByID(id string) (*entity.File, error)
	FindByUserID(userID string) ([]*entity.File, error)
}
