// file_repository.go
package repository

import (
	"github.com/Franciswann/aidms-backend/internal/domain/entity"
	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
	"gorm.io/gorm"
)

var _ domainrepo.FileRepository = (*FileRepository)(nil)

type FileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) *FileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) Save(file *entity.File) error {
	model := FileFromDomain(file)
	return r.db.Create(model).Error
}

func (r *FileRepository) FindByID(id string) (*entity.File, error) {
	var model FileModel
	if err := r.db.Where("id = ?", id).First(&model).Error; err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *FileRepository) FindByUserID(userID string) ([]*entity.File, error) {
	var models []FileModel
	if err := r.db.Where("user_id = ?", userID).Find(&models).Error; err != nil {
		return nil, err
	}
	files := make([]*entity.File, len(models))
	for i, m := range models {
		files[i] = m.ToDomain()
	}
	return files, nil
}

func (r *FileRepository) Delete(id string) error {
	return r.db.Delete(&FileModel{}, "id = ?", id).Error
}
