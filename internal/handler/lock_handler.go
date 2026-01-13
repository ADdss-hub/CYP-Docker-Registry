// Package handler provides HTTP handlers for the container registry.
package handler

import (
	"net/http"

	"cyp-docker-registry/internal/service"

	"github.com/gin-gonic/gin"
)

// LockHandler handles system lock requests.
type LockHandler struct {
	lockService  *service.LockService
	auditService *service.AuditService
}

// NewLockHandler creates a new LockHandler instance.
func NewLockHandler(lockSvc *service.LockService, auditSvc *service.AuditService) *LockHandler {
	return &LockHandler{
		lockService:  lockSvc,
		auditService: auditSvc,
	}
}

// RegisterRoutes registers lock routes.
func (h *LockHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/status", h.GetLockStatus)
	r.POST("/unlock", h.Unlock)
	r.POST("/lock", h.Lock)
}

// GetLockStatus returns the current lock status.
func (h *LockHandler) GetLockStatus(c *gin.Context) {
	if h.lockService == nil {
		c.JSON(http.StatusOK, gin.H{
			"is_locked": false,
		})
		return
	}

	status := h.lockService.GetLockStatus()
	c.JSON(http.StatusOK, status)
}

// UnlockRequest represents an unlock request.
type UnlockRequest struct {
	Password    string `json:"password" binding:"required"`
	RecoveryKey string `json:"recovery_key,omitempty"`
}

// Unlock handles system unlock requests.
func (h *LockHandler) Unlock(c *gin.Context) {
	var req UnlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
			"code":  "invalid_request",
		})
		return
	}

	if h.lockService == nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "System is not locked",
		})
		return
	}

	// Verify admin password or recovery key
	err := h.lockService.UnlockSystem(req.Password)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Invalid password",
			"code":  "invalid_password",
		})
		return
	}

	// Log unlock event
	if h.auditService != nil {
		user, _ := c.Get("currentUser")
		username := ""
		if user != nil {
			username = user.(*service.User).Username
		}
		h.auditService.LogUnlockEvent(c.ClientIP(), username)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "System unlocked successfully",
	})
}

// LockRequest represents a manual lock request.
type LockRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// Lock handles manual system lock requests.
func (h *LockHandler) Lock(c *gin.Context) {
	var req LockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
			"code":  "invalid_request",
		})
		return
	}

	if h.lockService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Lock service not available",
		})
		return
	}

	err := h.lockService.LockSystem(req.Reason, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to lock system",
		})
		return
	}

	// Log lock event
	if h.auditService != nil {
		h.auditService.LogLockEvent(c.ClientIP(), req.Reason, "manual")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "System locked successfully",
	})
}
