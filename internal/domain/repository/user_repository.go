package repository

import "github.com/Franciswann/aidms-backend/internal/domain/entity"

// UserRepository persists and retrieves user accounts.
type UserRepository interface {
	Save(user *entity.User) error
	FindByID(id string) (*entity.User, error)
	FindByEmail(email string) (*entity.User, error)
}
