package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Franciswann/aidms-backend/configs"
	_ "github.com/Franciswann/aidms-backend/docs"
	"github.com/Franciswann/aidms-backend/internal/docker"
	"github.com/Franciswann/aidms-backend/internal/handler"
	"github.com/Franciswann/aidms-backend/internal/logger"
	"github.com/Franciswann/aidms-backend/internal/middleware"
	"github.com/Franciswann/aidms-backend/internal/repository"
	"github.com/Franciswann/aidms-backend/internal/usecase/container"
	"github.com/Franciswann/aidms-backend/internal/usecase/file"
	"github.com/Franciswann/aidms-backend/internal/usecase/job"
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

	jobRepo := repository.NewJobRepository(db)
	jobService := job.NewJobService(jobRepo)
	jobHandler := handler.NewJobHandler(jobService)

	dockerRuntime, err := docker.NewDockerRuntime()
	if err != nil {
		log.Fatalf("failed to create docker runtime: %v", err)
	}
	containerRepo := repository.NewContainerRepository(db)
	containerService := container.NewContainerService(dockerRuntime, containerRepo, jobRepo)
	containerHandler := handler.NewContainerHandler(containerService)

	fileRepo := repository.NewFileRepository(db)
	fileService := file.NewFileService(fileRepo, cfg.FileStoragePath)
	fileHandler := handler.NewFileHandler(fileService)

	logStore, err := logger.NewFileLogStore(cfg.LogFilePath)
	if err != nil {
		log.Fatalf("failed to create log store: %v", err)
	}
	logManager := logger.NewLogManager(logStore, logStore, logger.LogLevel(cfg.LogMinLevel))

	r := gin.Default()
	r.Use(middleware.LoggingMiddleware(logManager))
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

	files := v1.Group("/files")
	files.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	files.POST("", fileHandler.Upload)
	files.GET("", fileHandler.List)
	files.DELETE("/:id", fileHandler.Delete)

	jobs := v1.Group("/jobs")
	jobs.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	jobs.GET("/:id", jobHandler.Get)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.ServerPort),
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutdown signal received, draining in-flight work...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown did not complete cleanly: %v", err)
	}

	containerService.Wait()
	logManager.Close()

	log.Println("shutdown complete")
}
