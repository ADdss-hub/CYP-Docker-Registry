// Package middleware provides security middleware for CYP-Registry.
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

		// Allow unlock endpoint
		if c.Request.URL.Path == "/api/v1/system/lock/unlock" {
			c.Next()
			return
		}

		// Allow health check
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/api/v1/system/health" {
			c.Next()
			return
		}

		// Allow lock status check
		if c.Request.URL.Path == "/api/v1/system/lock/status" {
			c.Next()
			return
		}

		// Check if system is locked
		if m.lockService.IsSystemLocked() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":       "System is locked",
				"details":     "system_locked",
				"lock_reason": m.lockService.GetLockReason(),
			})
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
			"error":   "System is in read-only mode",
			"details": "readonly_mode",
		})
	}
}
