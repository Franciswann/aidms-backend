// job_repository.go
package repository

import (
	"github.com/Franciswann/aidms-backend/internal/domain/entity"
	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
	"gorm.io/gorm"
)

type JobRepository struct {
	db *gorm.DB
}

var _ domainrepo.JobRepository = (*JobRepository)(nil)

func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) Save(job *entity.Job) error {
	model := JobFromDomain(job)
	return r.db.Create(model).Error
}

func (r *JobRepository) Update(job *entity.Job) error {
	model := JobFromDomain(job)
	return r.db.Save(model).Error
}

func (r *JobRepository) FindByID(id string) (*entity.Job, error) {
	var model JobModel
	if err := r.db.Where("id = ?", id).First(&model).Error; err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *JobRepository) FindByUserID(userID string) ([]*entity.Job, error) {
	var models []JobModel
	if err := r.db.Where("user_id = ?", userID).Find(&models).Error; err != nil {
		return nil, err
	}
	jobs := make([]*entity.Job, len(models))
	for i, m := range models {
		jobs[i] = m.ToDomain()
	}
	return jobs, nil
}
