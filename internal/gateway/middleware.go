package gateway

import (
	"container-registry/internal/common"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// logger is the package-level logger instance.
var logger *zap.Logger

// InitLogger initializes the package logger.
func InitLogger(l *zap.Logger) {
	logger = l
}

// LoggingMiddleware returns a middleware that logs HTTP requests.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Log request details
		if logger != nil {
			logger.Info("HTTP Request",
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.String("query", query),
				zap.Int("status", c.Writer.Status()),
				zap.Duration("latency", latency),
				zap.String("client_ip", c.ClientIP()),
				zap.Int("body_size", c.Writer.Size()),
			)
		}
	}
}

// ErrorHandlingMiddleware returns a middleware that handles panics and errors.
func ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				if logger != nil {
					logger.Error("Panic recovered",
						zap.Any("error", err),
						zap.String("path", c.Request.URL.Path),
					)
				}

				// Return internal error response
				common.ErrorResponse(c, common.ErrInternalError, map[string]interface{}{
					"recovered": true,
				})
				c.Abort()
			}
		}()

		c.Next()

		// Handle errors set during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			if logger != nil {
				logger.Error("Request error",
					zap.Error(err.Err),
					zap.String("path", c.Request.URL.Path),
				)
			}
		}
	}
}

// CORSMiddleware returns a middleware that handles CORS.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
