package handler

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
	"github.com/Franciswann/aidms-backend/internal/usecase/container"
	"github.com/gin-gonic/gin"
)

type ContainerHandler struct {
	containerService *container.ContainerService
}

func NewContainerHandler(containerService *container.ContainerService) *ContainerHandler {
	return &ContainerHandler{containerService: containerService}
}

type createContainerRequest struct {
	Image string `json:"image" binding:"required"`
	Name  string `json:"name" binding:"required"`
}

type containerResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Image     string    `json:"image"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func toContainerResponse(c *entity.Container) containerResponse {
	return containerResponse{
		ID:        c.ID,
		Name:      c.Name,
		Image:     c.Image,
		Status:    string(c.Status),
		CreatedAt: c.CreatedAt,
	}
}

// handleServiceError maps the small set of errors ContainerService can
// return to the right HTTP status code. Centralized here so the six
// handler methods below don't each repeat this same if/else chain.
func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainrepo.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
	case errors.Is(err, container.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		log.Printf("container handler: unexpected error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

// Create godoc
// @Summary      Create a new container (asynchronous)
// @Description  Starts pulling the given image and creating a Docker container in the background, owned by the authenticated user. Returns a Job immediately - poll GET /jobs/{id} for completion.
// @Tags         containers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body createContainerRequest true "Container creation payload"
// @Success      202 {object} jobResponse
// @Failure      400 {object} errorResponse
// @Failure      401 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /containers [post]
func (h *ContainerHandler) Create(c *gin.Context) {
	userID := c.GetString("userID")

	var req createContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	j, err := h.containerService.CreateAsync(userID, req.Image, req.Name)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, toJobResponse(j))
}

// Start godoc
// @Summary      Start a container
// @Description  Start a previously created container owned by the authenticated user
// @Tags         containers
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Container ID"
// @Success      200 {object} map[string]string
// @Failure      401 {object} errorResponse
// @Failure      403 {object} errorResponse
// @Failure      404 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /containers/{id}/start [post]
func (h *ContainerHandler) Start(c *gin.Context) {
	userID := c.GetString("userID")
	containerID := c.Param("id")

	if err := h.containerService.Start(userID, containerID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

// List godoc
// @Summary      List containers
// @Description  List all containers owned by the authenticated user
// @Tags         containers
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} containerResponse
// @Failure      401 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /containers [get]
func (h *ContainerHandler) List(c *gin.Context) {
	userID := c.GetString("userID")

	containers, err := h.containerService.List(userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	resp := make([]containerResponse, len(containers))
	for i, ctr := range containers {
		resp[i] = toContainerResponse(ctr)
	}
	c.JSON(http.StatusOK, resp)
}

// Get godoc
// @Summary      Get a container
// @Description  Get a single container owned by the authenticated user
// @Tags         containers
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Container ID"
// @Success      200 {object} containerResponse
// @Failure      401 {object} errorResponse
// @Failure      403 {object} errorResponse
// @Failure      404 {object} errorResponse
// @Router       /containers/{id} [get]
func (h *ContainerHandler) Get(c *gin.Context) {
	userID := c.GetString("userID")
	containerID := c.Param("id")

	ctr, err := h.containerService.Get(userID, containerID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toContainerResponse(ctr))
}

// Stop godoc
// @Summary      Stop a container
// @Description  Stop a running container owned by the authenticated user
// @Tags         containers
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Container ID"
// @Success      200 {object} map[string]string
// @Failure      401 {object} errorResponse
// @Failure      403 {object} errorResponse
// @Failure      404 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /containers/{id}/stop [post]
func (h *ContainerHandler) Stop(c *gin.Context) {
	userID := c.GetString("userID")
	containerID := c.Param("id")

	if err := h.containerService.Stop(userID, containerID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

// Delete godoc
// @Summary      Delete a container
// @Description  Remove a container (from Docker and the database) owned by the authenticated user
// @Tags         containers
// @Security     BearerAuth
// @Param        id path string true "Container ID"
// @Success      204 "No Content"
// @Failure      401 {object} errorResponse
// @Failure      403 {object} errorResponse
// @Failure      404 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /containers/{id} [delete]
func (h *ContainerHandler) Delete(c *gin.Context) {
	userID := c.GetString("userID")
	containerID := c.Param("id")

	if err := h.containerService.Delete(userID, containerID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
