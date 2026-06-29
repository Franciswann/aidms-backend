package handler

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
	"github.com/Franciswann/aidms-backend/internal/usecase/job"
	"github.com/gin-gonic/gin"
)

// JobHandler exposes the asynchronous job status endpoint.
type JobHandler struct {
	jobService *job.JobService
}

func NewJobHandler(jobService *job.JobService) *JobHandler {
	return &JobHandler{jobService: jobService}
}

type jobResponse struct {
	ID           string    `json:"id"`
	ContainerID  string    `json:"container_id,omitempty"`
	Status       string    `json:"status"`
	Action       string    `json:"action"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func toJobResponse(j *entity.Job) jobResponse {
	return jobResponse{
		ID:           j.ID,
		ContainerID:  j.ContainerID,
		Status:       string(j.Status),
		Action:       string(j.Action),
		ErrorMessage: j.ErrorMessage,
		CreatedAt:    j.CreatedAt,
		UpdatedAt:    j.UpdatedAt,
	}
}

func handleJobServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainrepo.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
	case errors.Is(err, job.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		log.Printf("job handler: unexpected error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

// Get godoc
// @Summary      Get a job's status
// @Description  Poll the status of an asynchronous job (e.g. container creation) owned by the authenticated user
// @Tags         jobs
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Job ID"
// @Success      200 {object} jobResponse
// @Failure      401 {object} errorResponse
// @Failure      403 {object} errorResponse
// @Failure      404 {object} errorResponse
// @Router       /jobs/{id} [get]
func (h *JobHandler) Get(c *gin.Context) {
	userID := c.GetString("userID")
	jobID := c.Param("id")

	j, err := h.jobService.Get(userID, jobID)
	if err != nil {
		handleJobServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toJobResponse(j))
}
