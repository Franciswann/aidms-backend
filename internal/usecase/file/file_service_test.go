package file

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFileRepository struct {
	byID    map[string]*entity.File
	saveErr error
}

var _ domainrepo.FileRepository = (*mockFileRepository)(nil)

func newMockFileRepository() *mockFileRepository {
	return &mockFileRepository{byID: make(map[string]*entity.File)}
}

func (m *mockFileRepository) Save(f *entity.File) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.byID[f.ID] = f
	return nil
}

func (m *mockFileRepository) Delete(id string) error {
	delete(m.byID, id)
	return nil
}

func (m *mockFileRepository) FindByID(id string) (*entity.File, error) {
	f, ok := m.byID[id]
	if !ok {
		return nil, domainrepo.ErrNotFound
	}
	return f, nil
}

func (m *mockFileRepository) FindByUserID(userID string) ([]*entity.File, error) {
	var out []*entity.File
	for _, f := range m.byID {
		if f.UserID == userID {
			out = append(out, f)
		}
	}
	return out, nil
}

var errSimulated = errors.New("simulated failure")

func TestFileService_Upload(t *testing.T) {
	t.Run("success - writes to a per-user directory and saves metadata", func(t *testing.T) {
		dir := t.TempDir()
		repo := newMockFileRepository()
		svc := NewFileService(repo, dir)

		f, err := svc.Upload("user-1", "report.csv", 11, "text/csv", strings.NewReader("hello world"))

		require.NoError(t, err)
		assert.Equal(t, "report.csv", f.Name)
		assert.Equal(t, "user-1", f.UserID)
		assert.NotEmpty(t, f.ID)

		// the file must live under the user's own subdirectory, named by its
		// own ID (not the original filename)
		assert.Equal(t, filepath.Join(dir, "user-1", f.ID), f.Path)
		content, err := os.ReadFile(f.Path)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(content))

		_, ok := repo.byID[f.ID]
		assert.True(t, ok)
	})

	t.Run("save fails - rolls back the orphaned file on disk", func(t *testing.T) {
		dir := t.TempDir()
		repo := newMockFileRepository()
		repo.saveErr = errSimulated
		svc := NewFileService(repo, dir)

		f, err := svc.Upload("user-1", "report.csv", 11, "text/csv", strings.NewReader("hello world"))

		require.ErrorIs(t, err, errSimulated)
		assert.Nil(t, f)

		entries, err := os.ReadDir(filepath.Join(dir, "user-1"))
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

func TestFileService_Get(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		svc := NewFileService(newMockFileRepository(), t.TempDir())

		_, err := svc.Get("user-1", "does-not-exist")

		require.ErrorIs(t, err, domainrepo.ErrNotFound)
	})

	t.Run("belongs to a different user", func(t *testing.T) {
		repo := newMockFileRepository()
		repo.byID["f-1"] = &entity.File{ID: "f-1", UserID: "owner"}
		svc := NewFileService(repo, t.TempDir())

		_, err := svc.Get("someone-else", "f-1")

		require.ErrorIs(t, err, ErrForbidden)
	})
}

func TestFileService_Delete(t *testing.T) {
	t.Run("success - removes the file from disk and the repository", func(t *testing.T) {
		dir := t.TempDir()
		repo := newMockFileRepository()
		path := filepath.Join(dir, "f-1")
		require.NoError(t, os.WriteFile(path, []byte("data"), 0644))
		repo.byID["f-1"] = &entity.File{ID: "f-1", UserID: "user-1", Path: path}
		svc := NewFileService(repo, dir)

		err := svc.Delete("user-1", "f-1")

		require.NoError(t, err)
		_, statErr := os.Stat(path)
		assert.True(t, os.IsNotExist(statErr))
		_, stillExists := repo.byID["f-1"]
		assert.False(t, stillExists)
	})

	t.Run("file already missing from disk - still deletes the DB record", func(t *testing.T) {
		dir := t.TempDir()
		repo := newMockFileRepository()
		repo.byID["f-1"] = &entity.File{ID: "f-1", UserID: "user-1", Path: filepath.Join(dir, "already-gone")}
		svc := NewFileService(repo, dir)

		err := svc.Delete("user-1", "f-1")

		require.NoError(t, err)
		_, stillExists := repo.byID["f-1"]
		assert.False(t, stillExists)
	})
}
