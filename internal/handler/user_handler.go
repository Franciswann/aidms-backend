package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/Franciswann/aidms-backend/internal/usecase/user"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *user.UserService
}

func NewUserHandler(userService *user.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	Token string `json:"token"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body registerRequest true "Registration payload"
// @Success      201 {object} userResponse
// @Failure      400 {object} errorResponse
// @Failure      409 {object} errorResponse
// @Router       /auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.userService.Register(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, user.ErrEmailAlreadyRegistered) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		log.Printf("user handler: unexpected error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, userResponse{ID: u.ID, Email: u.Email})
}

// Login godoc
// @Summary      User login
// @Description  Validate credentials and return a JWT token on success
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body loginRequest true "Login payload"
// @Success      200 {object} loginResponse
// @Failure      400 {object} errorResponse
// @Failure      401 {object} errorResponse
// @Router       /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.userService.Login(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, user.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		log.Printf("user handler: unexpected error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, loginResponse{Token: token})
}
