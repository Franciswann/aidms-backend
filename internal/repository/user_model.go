package repository

import (
	"time"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
)

type UserModel struct {
	ID             string `gorm:"primaryKey"`
	Email          string `gorm:"unique"`
	HashedPassword string
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

func (UserModel) TableName() string {
	return "users"
}

func (u *UserModel) ToDomain() *entity.User {
	return &entity.User{
		ID:             u.ID,
		Email:          u.Email,
		HashedPassword: u.HashedPassword,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

func UserFromDomain(u *entity.User) *UserModel {
	return &UserModel{
		ID:             u.ID,
		Email:          u.Email,
		HashedPassword: u.HashedPassword,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}
