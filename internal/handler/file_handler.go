package handler

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
	"github.com/Franciswann/aidms-backend/internal/usecase/file"
	"github.com/gin-gonic/gin"
)

// FileHandler exposes the file upload/list/delete endpoints.
type FileHandler struct {
	fileService *file.FileService
}

func NewFileHandler(fileService *file.FileService) *FileHandler {
	return &FileHandler{fileService: fileService}
}

type fileResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	MimeType  string    `json:"mime_type"`
	CreatedAt time.Time `json:"created_at"`
}

func toFileResponse(f *entity.File) fileResponse {
	return fileResponse{
		ID:        f.ID,
		Name:      f.Name,
		Size:      f.Size,
		MimeType:  f.MimeType,
		CreatedAt: f.CreatedAt,
	}
}

func handleFileServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainrepo.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
	case errors.Is(err, file.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		log.Printf("file handler: unexpected error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

// Upload godoc
// @Summary      Upload a file
// @Description  Upload a file into the authenticated user's storage
// @Tags         files
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file formData file true "File to upload"
// @Success      201 {object} fileResponse
// @Failure      400 {object} errorResponse
// @Failure      401 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /files [post]
func (h *FileHandler) Upload(c *gin.Context) {
	userID := c.GetString("userID")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		log.Printf("file handler: failed to open uploaded file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	defer src.Close()

	uploaded, err := h.fileService.Upload(userID, fileHeader.Filename, fileHeader.Size, fileHeader.Header.Get("Content-Type"), src)
	if err != nil {
		handleFileServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toFileResponse(uploaded))
}

// List godoc
// @Summary      List files
// @Description  List all files owned by the authenticated user
// @Tags         files
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} fileResponse
// @Failure      401 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /files [get]
func (h *FileHandler) List(c *gin.Context) {
	userID := c.GetString("userID")

	files, err := h.fileService.List(userID)
	if err != nil {
		handleFileServiceError(c, err)
		return
	}

	resp := make([]fileResponse, len(files))
	for i, f := range files {
		resp[i] = toFileResponse(f)
	}
	c.JSON(http.StatusOK, resp)
}

// Delete godoc
// @Summary      Delete a file
// @Description  Remove a file (from disk and the database) owned by the authenticated user
// @Tags         files
// @Security     BearerAuth
// @Param        id path string true "File ID"
// @Success      204 "No Content"
// @Failure      401 {object} errorResponse
// @Failure      403 {object} errorResponse
// @Failure      404 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /files/{id} [delete]
func (h *FileHandler) Delete(c *gin.Context) {
	userID := c.GetString("userID")
	fileID := c.Param("id")

	if err := h.fileService.Delete(userID, fileID); err != nil {
		handleFileServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
