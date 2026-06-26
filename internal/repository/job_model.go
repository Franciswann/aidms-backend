package repository

import (
	"time"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
)

type JobModel struct {
	ID           string `gorm:"primaryKey"`
	UserID       string `gorm:"index"`
	ContainerID  string
	ErrorMessage string
	Status       entity.JobStatus
	Action       entity.JobAction
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

func (JobModel) TableName() string {
	return "jobs"
}

func (j *JobModel) ToDomain() *entity.Job {
	return &entity.Job{
		ID:           j.ID,
		UserID:       j.UserID,
		ContainerID:  j.ContainerID,
		ErrorMessage: j.ErrorMessage,
		Status:       j.Status,
		Action:       j.Action,
		CreatedAt:    j.CreatedAt,
		UpdatedAt:    j.UpdatedAt,
	}
}
