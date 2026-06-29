package repository

import "github.com/Franciswann/aidms-backend/internal/domain/entity"

// JobRepository persists and retrieves asynchronous job records.
type JobRepository interface {
	Save(job *entity.Job) error
	Update(job *entity.Job) error
	FindByID(id string) (*entity.Job, error)
	FindByUserID(userID string) ([]*entity.Job, error)
}
