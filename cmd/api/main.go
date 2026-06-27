package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Franciswann/aidms-backend/configs"
	"github.com/Franciswann/aidms-backend/internal/handler"
	"github.com/Franciswann/aidms-backend/internal/repository"
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

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	v1.POST("/auth/register", userHandler.Register)
	v1.POST("/auth/login", userHandler.Login)

	if err := r.Run(fmt.Sprintf(":%s", cfg.ServerPort)); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
