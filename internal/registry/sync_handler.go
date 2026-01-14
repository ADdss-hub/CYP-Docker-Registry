// Package registry provides container image registry functionality.
package registry

import (
	"cyp-docker-registry/internal/common"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SyncHandler provides HTTP handlers for sync operations.
type SyncHandler struct {
	syncService       *SyncService
	credentialManager *CredentialManager
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(syncService *SyncService, credentialManager *CredentialManager) *SyncHandler {
	return &SyncHandler{
		syncService:       syncService,
		credentialManager: credentialManager,
	}
}

// RegisterRoutes registers sync routes on the given router group.
func (h *SyncHandler) RegisterRoutes(apiGroup *gin.RouterGroup) {
	// Credential management routes
	creds := apiGroup.Group("/credentials")
	{
		creds.GET("", h.listCredentials)
		creds.POST("", h.saveCredential)
		creds.GET("/:registry", h.getCredential)
		creds.DELETE("/:registry", h.deleteCredential)
	}

	// Sync routes
	sync := apiGroup.Group("/sync")
	{
		sync.POST("", h.syncImage)
		sync.GET("/history", h.getSyncHistory)
		sync.GET("/history/:id", h.getSyncRecord)
		sync.POST("/retry/:id", h.retrySync)
		sync.GET("/image/:name/:tag", h.getImageSyncHistory)
	}
}

// ============================================================================
// Credential Handlers
// ============================================================================

// listCredentials handles GET /api/credentials
func (h *SyncHandler) listCredentials(c *gin.Context) {
	credentials, err := h.credentialManager.ListCredentials()
	if err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"credentials": credentials,
	})
}

// CredentialRequest represents a request to save a credential.
type CredentialRequest struct {
	Registry string `json:"registry" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// saveCredential handles POST /api/credentials
func (h *SyncHandler) saveCredential(c *gin.Context) {
	var req CredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, common.ErrInvalidRequest, gin.H{
			"error": "仓库地址、用户名和密码为必填项",
		})
		return
	}

	if err := h.credentialManager.SaveCredential(req.Registry, req.Username, req.Password); err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message":  "凭证保存成功",
		"registry": req.Registry,
	})
}

// getCredential handles GET /api/credentials/:registry
func (h *SyncHandler) getCredential(c *gin.Context) {
	registry := c.Param("registry")

	// Return encrypted credential (don't expose password)
	cred, err := h.credentialManager.GetCredentialEncrypted(registry)
	if err != nil {
		common.ErrorResponse(c, common.ErrNotFound, gin.H{
			"error":    "凭证不存在",
			"registry": registry,
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"registry":   registry,
		"username":   cred.Username,
		"password":   "********", // Mask password
		"created_at": cred.CreatedAt,
		"updated_at": cred.UpdatedAt,
	})
}

// deleteCredential handles DELETE /api/credentials/:registry
func (h *SyncHandler) deleteCredential(c *gin.Context) {
	registry := c.Param("registry")

	if err := h.credentialManager.DeleteCredential(registry); err != nil {
		common.ErrorResponse(c, common.ErrNotFound, gin.H{
			"error":    "凭证不存在",
			"registry": registry,
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message":  "凭证删除成功",
		"registry": registry,
	})
}

// ============================================================================
// Sync Handlers
// ============================================================================

// syncImage handles POST /api/sync
func (h *SyncHandler) syncImage(c *gin.Context) {
	var req SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, common.ErrInvalidRequest, gin.H{
			"error": "镜像名称、镜像标签和目标仓库为必填项",
		})
		return
	}

	record, err := h.syncService.SyncImage(&req)
	if err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, common.Response{
		Success: true,
		Data: gin.H{
			"message": "同步任务已启动",
			"record":  record,
		},
	})
}

// getSyncHistory handles GET /api/sync/history
func (h *SyncHandler) getSyncHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	records, total, err := h.syncService.GetSyncHistory(page, pageSize)
	if err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	common.SuccessResponse(c, gin.H{
		"records":     records,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

// getSyncRecord handles GET /api/sync/history/:id
func (h *SyncHandler) getSyncRecord(c *gin.Context) {
	id := c.Param("id")

	record, err := h.syncService.GetSyncRecord(id)
	if err != nil {
		common.ErrorResponse(c, common.ErrNotFound, gin.H{
			"error": "同步记录不存在",
			"id":    id,
		})
		return
	}

	common.SuccessResponse(c, record)
}

// retrySync handles POST /api/sync/retry/:id
func (h *SyncHandler) retrySync(c *gin.Context) {
	id := c.Param("id")

	record, err := h.syncService.RetrySync(id)
	if err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, common.Response{
		Success: true,
		Data: gin.H{
			"message": "重试任务已启动",
			"record":  record,
		},
	})
}

// getImageSyncHistory handles GET /api/sync/image/:name/:tag
func (h *SyncHandler) getImageSyncHistory(c *gin.Context) {
	name := c.Param("name")
	tag := c.Param("tag")

	records, err := h.syncService.GetSyncHistoryByImage(name, tag)
	if err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"image_name": name,
		"image_tag":  tag,
		"records":    records,
	})
}
