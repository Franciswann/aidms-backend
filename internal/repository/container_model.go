package repository

import (
	"time"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
)

type ContainerModel struct {
	ID        string `gorm:"primaryKey"`
	UserID    string `gorm:"index"`
	DockerID  string `gorm:"unique"`
	Name      string
	Image     string
	Status    entity.ContainerStatus
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (ContainerModel) TableName() string {
	return "containers"
}

func (c *ContainerModel) ToDomain() *entity.Container {
	return &entity.Container{
		ID:        c.ID,
		UserID:    c.UserID,
		DockerID:  c.DockerID,
		Name:      c.Name,
		Image:     c.Image,
		Status:    c.Status,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func ContainerFromDomain(c *entity.Container) *ContainerModel {
	return &ContainerModel{
		ID:        c.ID,
		UserID:    c.UserID,
		DockerID:  c.DockerID,
		Name:      c.Name,
		Image:     c.Image,
		Status:    c.Status,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
