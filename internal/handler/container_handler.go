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

func (h *ContainerHandler) Create(c *gin.Context) {
	userID := c.GetString("userID")

	var req createContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctr, err := h.containerService.Create(userID, req.Image, req.Name)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toContainerResponse(ctr))
}

func (h *ContainerHandler) Start(c *gin.Context) {
	userID := c.GetString("userID")
	containerID := c.Param("id")

	if err := h.containerService.Start(userID, containerID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

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

func (h *ContainerHandler) Stop(c *gin.Context) {
	userID := c.GetString("userID")
	containerID := c.Param("id")

	if err := h.containerService.Stop(userID, containerID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

func (h *ContainerHandler) Delete(c *gin.Context) {
	userID := c.GetString("userID")
	containerID := c.Param("id")

	if err := h.containerService.Delete(userID, containerID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
