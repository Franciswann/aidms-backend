package middleware

import (
	"time"

	"github.com/Franciswann/aidms-backend/internal/logger"
	"github.com/gin-gonic/gin"
)

// LoggingMiddleware records one structured entry per request via the
// internal/logger LogManager, replacing Gin's default access log.
func LoggingMiddleware(manager *logger.LogManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		status := c.Writer.Status()
		level := "info"
		switch {
		case status >= 500:
			level = "error"
		case status >= 400:
			level = "warn"
		}

		manager.WriteLogFields(level, "http request", map[string]interface{}{
			"method":    c.Request.Method,
			"path":      c.Request.URL.Path,
			"status":    status,
			"latency":   latency.String(),
			"client_ip": c.ClientIP(),
		})
	}
}
