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
// 问题9修复：系统锁定后不允许手动解锁，只能联系管理员或重新安装
func (h *LockHandler) Unlock(c *gin.Context) {
	var req UnlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求参数",
			"code":  "invalid_request",
		})
		return
	}

	if h.lockService == nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "系统未锁定",
		})
		return
	}

	// 检查系统是否锁定
	status := h.lockService.GetLockStatus()
	if !status.IsLocked {
		c.JSON(http.StatusOK, gin.H{
			"message": "系统未锁定",
		})
		return
	}

	// 系统锁定后不允许手动解锁
	// 只能通过以下方式解锁：
	// 1. 联系管理员进行后台操作
	// 2. 重新安装系统
	c.JSON(http.StatusForbidden, gin.H{
		"error":   "系统已锁定，不允许手动解锁",
		"code":    "manual_unlock_disabled",
		"message": "系统锁定后不允许手动解锁。请联系管理员或重新安装系统。",
		"details": gin.H{
			"lock_reason":   status.LockReason,
			"lock_type":     status.LockType,
			"locked_at":     status.LockedAt,
			"require_admin": true,
		},
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
			"error": "无效的请求参数",
			"code":  "invalid_request",
		})
		return
	}

	if h.lockService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "锁定服务不可用",
		})
		return
	}

	err := h.lockService.LockSystem(req.Reason, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "系统锁定失败",
		})
		return
	}

	// Log lock event
	if h.auditService != nil {
		h.auditService.LogLockEvent(c.ClientIP(), req.Reason, "manual")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "系统锁定成功",
	})
}
