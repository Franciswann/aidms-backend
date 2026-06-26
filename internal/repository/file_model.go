package repository

import (
	"time"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
)

type FileModel struct {
	ID        string `gorm:"primaryKey"`
	UserID    string `gorm:"index"`
	Name      string
	Path      string
	MimeType  string
	Size      int64
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (FileModel) TableName() string {
	return "files"
}

func (f *FileModel) ToDomain() *entity.File {
	return &entity.File{
		ID:        f.ID,
		UserID:    f.UserID,
		Name:      f.Name,
		Path:      f.Path,
		MimeType:  f.MimeType,
		Size:      f.Size,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}
