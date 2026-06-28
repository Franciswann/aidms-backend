package configs

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	ServerPort      string
	JWTSecret       string
	FileStoragePath string
}

func Load() *Config {
	loadErr := godotenv.Load(".env")
	if loadErr != nil {
		log.Printf("Warning: .env not found")
	}

	return &Config{
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "aidms"),
		DBPassword:      getEnv("DB_PASSWORD", "aidms_secret"),
		DBName:          getEnv("DB_NAME", "aidms_db"),
		ServerPort:      getEnv("SERVER_PORT", "8080"),
		JWTSecret:       mustGetEnv("JWT_SECRET"),
		FileStoragePath: getEnv("FILE_STORAGE_PATH", "./uploads"),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName)
}

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s is required but not set", key)
	}
	return value
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
