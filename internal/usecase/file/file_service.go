package file

import (
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
	"github.com/google/uuid"
)

var ErrForbidden = errors.New("you do not have access to this file")

type FileService struct {
	repo        domainrepo.FileRepository
	storagePath string
}

func NewFileService(repo domainrepo.FileRepository, storagePath string) *FileService {
	return &FileService{repo: repo, storagePath: storagePath}
}

func (s *FileService) Upload(userID, originalName string, size int64, mimeType string, content io.Reader) (*entity.File, error) {
	id := uuid.NewString()
	userDir := filepath.Join(s.storagePath, userID)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return nil, err
	}
	destPath := filepath.Join(userDir, id)

	dest, err := os.Create(destPath)
	if err != nil {
		return nil, err
	}
	defer dest.Close()

	if _, err := io.Copy(dest, content); err != nil {
		if rmErr := os.Remove(destPath); rmErr != nil {
			log.Printf("failed to clean up partial file %s after copy error %v: %v", destPath, err, rmErr)
		}
		return nil, err
	}

	f := &entity.File{
		ID:        id,
		UserID:    userID,
		Name:      originalName,
		Path:      destPath,
		Size:      size,
		MimeType:  mimeType,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Save(f); err != nil {
		if rmErr := os.Remove(destPath); rmErr != nil {
			log.Printf("failed to roll back orphaned file %s after save error %v: %v", destPath, err, rmErr)
		}
		return nil, err
	}

	return f, nil
}

func (s *FileService) getOwned(userID, fileID string) (*entity.File, error) {
	f, err := s.repo.FindByID(fileID)
	if err != nil {
		return nil, err
	}
	if f.UserID != userID {
		return nil, ErrForbidden
	}
	return f, nil
}

func (s *FileService) List(userID string) ([]*entity.File, error) {
	return s.repo.FindByUserID(userID)
}

func (s *FileService) Get(userID, fileID string) (*entity.File, error) {
	return s.getOwned(userID, fileID)
}

func (s *FileService) Delete(userID, fileID string) error {
	f, err := s.getOwned(userID, fileID)
	if err != nil {
		return err
	}

	if err := os.Remove(f.Path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return s.repo.Delete(f.ID)
}
