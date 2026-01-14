// Package middleware provides security middleware for CYP-Docker-Registry.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// LockMiddleware provides system lock detection middleware.
type LockMiddleware struct {
	lockService LockServiceInterface
}

// NewLockMiddleware creates a new LockMiddleware instance.
func NewLockMiddleware(lockSvc LockServiceInterface) *LockMiddleware {
	return &LockMiddleware{
		lockService: lockSvc,
	}
}

// CheckLock returns a middleware that checks if the system is locked.
func (m *LockMiddleware) CheckLock() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.lockService == nil {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		// Allow unlock endpoint
		if path == "/api/v1/system/lock/unlock" {
			c.Next()
			return
		}

		// Allow health check
		if path == "/health" || path == "/api/v1/system/health" {
			c.Next()
			return
		}

		// Allow lock status check
		if path == "/api/v1/system/lock/status" {
			c.Next()
			return
		}

		// Allow frontend static resources (for locked page to render)
		if path == "/" || path == "/locked" || path == "/login" ||
			len(path) > 7 && path[:7] == "/assets" ||
			path == "/favicon.ico" || path == "/vite.svg" {
			c.Next()
			return
		}

		// Check if system is locked
		if m.lockService.IsSystemLocked() {
			// For API requests, return JSON
			if len(path) > 4 && path[:4] == "/api" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":       "系统已锁定",
					"details":     "system_locked",
					"lock_reason": m.lockService.GetLockReason(),
				})
				return
			}
			// For page requests, let frontend handle the redirect
			c.Next()
			return
		}

		c.Next()
	}
}

// ReadOnlyMode returns a middleware that enforces read-only mode.
func ReadOnlyMode(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled {
			c.Next()
			return
		}

		// Allow read operations
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Block write operations
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":   "系统处于只读模式",
			"details": "readonly_mode",
		})
	}
}
