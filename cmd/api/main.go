package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Franciswann/aidms-backend/configs"
	_ "github.com/Franciswann/aidms-backend/docs"
	"github.com/Franciswann/aidms-backend/internal/docker"
	"github.com/Franciswann/aidms-backend/internal/handler"
	"github.com/Franciswann/aidms-backend/internal/middleware"
	"github.com/Franciswann/aidms-backend/internal/repository"
	"github.com/Franciswann/aidms-backend/internal/usecase/container"
	"github.com/Franciswann/aidms-backend/internal/usecase/user"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @title           AIDMS Backend API
// @version         1.0
// @description     Backend API for the container management system
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     JWT token, format: "Bearer {token}"
func main() {
	cfg := configs.Load()

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(
		&repository.UserModel{},
		&repository.ContainerModel{},
		&repository.FileModel{},
		&repository.JobModel{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	userService := user.NewUserService(userRepo, cfg.JWTSecret)
	userHandler := handler.NewUserHandler(userService)

	dockerRuntime, err := docker.NewDockerRuntime()
	if err != nil {
		log.Fatalf("failed to create docker runtime: %v", err)
	}
	containerRepo := repository.NewContainerRepository(db)
	containerService := container.NewContainerService(dockerRuntime, containerRepo)
	containerHandler := handler.NewContainerHandler(containerService)

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	auth := v1.Group("/auth")
	auth.POST("/register", userHandler.Register)
	auth.POST("/login", userHandler.Login)

	containers := v1.Group("/containers")
	containers.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	containers.POST("", containerHandler.Create)
	containers.GET("", containerHandler.List)
	containers.GET("/:id", containerHandler.Get)
	containers.POST("/:id/start", containerHandler.Start)
	containers.POST("/:id/stop", containerHandler.Stop)
	containers.DELETE("/:id", containerHandler.Delete)

	if err := r.Run(fmt.Sprintf(":%s", cfg.ServerPort)); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
