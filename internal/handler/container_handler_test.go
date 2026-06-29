package handler

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
	"github.com/Franciswann/aidms-backend/internal/usecase/container"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// mockContainerUsecase is a fake implementation of containerUsecase. Each
// test only sets the func field it actually needs; calling an unset field
// panics with a nil pointer, which surfaces immediately as a test failure.
type mockContainerUsecase struct {
	createAsyncFunc func(userID, image, name string) (*entity.Job, error)
	listFunc        func(userID string) ([]*entity.Container, error)
	getFunc         func(userID, containerID string) (*entity.Container, error)
	startFunc       func(userID, containerID string) error
	stopFunc        func(userID, containerID string) error
	deleteFunc      func(userID, containerID string) error
}

func (m *mockContainerUsecase) CreateAsync(userID, image, name string) (*entity.Job, error) {
	return m.createAsyncFunc(userID, image, name)
}
func (m *mockContainerUsecase) List(userID string) ([]*entity.Container, error) {
	return m.listFunc(userID)
}
func (m *mockContainerUsecase) Get(userID, containerID string) (*entity.Container, error) {
	return m.getFunc(userID, containerID)
}
func (m *mockContainerUsecase) Start(userID, containerID string) error {
	return m.startFunc(userID, containerID)
}
func (m *mockContainerUsecase) Stop(userID, containerID string) error {
	return m.stopFunc(userID, containerID)
}
func (m *mockContainerUsecase) Delete(userID, containerID string) error {
	return m.deleteFunc(userID, containerID)
}

// newTestRouter wires the given mock straight into ContainerHandler's real
// routes, with a stub middleware standing in for AuthMiddleware (it just sets
// "userID" the same way the real middleware does after verifying a JWT).
func newTestRouter(mock *mockContainerUsecase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", "user-1")
		c.Next()
	})

	h := NewContainerHandler(mock)
	r.POST("/containers", h.Create)
	r.GET("/containers", h.List)
	r.GET("/containers/:id", h.Get)
	r.POST("/containers/:id/start", h.Start)
	r.POST("/containers/:id/stop", h.Stop)
	r.DELETE("/containers/:id", h.Delete)
	return r
}

func TestContainerHandler_Create_Success(t *testing.T) {
	mock := &mockContainerUsecase{
		createAsyncFunc: func(userID, image, name string) (*entity.Job, error) {
			assert.Equal(t, "user-1", userID)
			assert.Equal(t, "nginx:latest", image)
			assert.Equal(t, "web", name)
			return &entity.Job{ID: "job-1", Status: entity.JobStatusPending, Action: entity.JobActionCreate}, nil
		},
	}
	r := newTestRouter(mock)

	body := bytes.NewBufferString(`{"image":"nginx:latest","name":"web"}`)
	req := httptest.NewRequest(http.MethodPost, "/containers", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"job-1"`)
	assert.Contains(t, w.Body.String(), `"status":"pending"`)
}

func TestContainerHandler_Create_InvalidBody(t *testing.T) {
	mock := &mockContainerUsecase{}
	r := newTestRouter(mock)

	// Missing the required "name" field - should fail binding before the
	// usecase is ever called.
	body := bytes.NewBufferString(`{"image":"nginx:latest"}`)
	req := httptest.NewRequest(http.MethodPost, "/containers", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestContainerHandler_Get_Success(t *testing.T) {
	mock := &mockContainerUsecase{
		getFunc: func(userID, containerID string) (*entity.Container, error) {
			return &entity.Container{
				ID:        containerID,
				Name:      "web",
				Image:     "nginx:latest",
				Status:    entity.ContainerStatusRunning,
				CreatedAt: time.Now(),
			}, nil
		},
	}
	r := newTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/containers/c-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"c-1"`)
}

func TestContainerHandler_Get_NotFound(t *testing.T) {
	mock := &mockContainerUsecase{
		getFunc: func(userID, containerID string) (*entity.Container, error) {
			return nil, domainrepo.ErrNotFound
		},
	}
	r := newTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/containers/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestContainerHandler_Get_Forbidden(t *testing.T) {
	mock := &mockContainerUsecase{
		getFunc: func(userID, containerID string) (*entity.Container, error) {
			return nil, container.ErrForbidden
		},
	}
	r := newTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/containers/someone-elses", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestContainerHandler_Get_InternalError(t *testing.T) {
	mock := &mockContainerUsecase{
		getFunc: func(userID, containerID string) (*entity.Container, error) {
			return nil, errors.New("docker daemon unreachable")
		},
	}
	r := newTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/containers/c-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	// The real error message must never leak to the client.
	assert.NotContains(t, w.Body.String(), "docker daemon unreachable")
}

func TestContainerHandler_List_Success(t *testing.T) {
	mock := &mockContainerUsecase{
		listFunc: func(userID string) ([]*entity.Container, error) {
			return []*entity.Container{
				{ID: "c-1", Name: "web"},
				{ID: "c-2", Name: "db"},
			}, nil
		},
	}
	r := newTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/containers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"c-1"`)
	assert.Contains(t, w.Body.String(), `"id":"c-2"`)
}

func TestContainerHandler_Start_Success(t *testing.T) {
	mock := &mockContainerUsecase{
		startFunc: func(userID, containerID string) error {
			return nil
		},
	}
	r := newTestRouter(mock)

	req := httptest.NewRequest(http.MethodPost, "/containers/c-1/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestContainerHandler_Stop_Success(t *testing.T) {
	mock := &mockContainerUsecase{
		stopFunc: func(userID, containerID string) error {
			return nil
		},
	}
	r := newTestRouter(mock)

	req := httptest.NewRequest(http.MethodPost, "/containers/c-1/stop", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestContainerHandler_Delete_Success(t *testing.T) {
	mock := &mockContainerUsecase{
		deleteFunc: func(userID, containerID string) error {
			return nil
		},
	}
	r := newTestRouter(mock)

	req := httptest.NewRequest(http.MethodDelete, "/containers/c-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
