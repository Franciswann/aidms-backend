package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Franciswann/aidms-backend/configs"
	"github.com/Franciswann/aidms-backend/internal/docker"
	"github.com/Franciswann/aidms-backend/internal/handler"
	"github.com/Franciswann/aidms-backend/internal/middleware"
	"github.com/Franciswann/aidms-backend/internal/repository"
	"github.com/Franciswann/aidms-backend/internal/usecase/container"
	"github.com/Franciswann/aidms-backend/internal/usecase/user"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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

	v1 := r.Group("/api/v1")
	v1.POST("/auth/register", userHandler.Register)
	v1.POST("/auth/login", userHandler.Login)

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
