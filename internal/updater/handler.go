// Package updater provides auto-update functionality for CYP-Docker-Registry.
package updater

import (
	"cyp-docker-registry/internal/common"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for update operations.
type Handler struct {
	service *UpdaterService
}

// NewHandler creates a new updater handler.
func NewHandler(service *UpdaterService) *Handler {
	return &Handler{
		service: service,
	}
}

// RegisterRoutes registers updater routes on the given router group.
func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/check", h.checkUpdate)
	group.GET("/status", h.getStatus)
	group.POST("/download", h.downloadUpdate)
	group.POST("/apply", h.applyUpdate)
	group.POST("/rollback", h.rollback)
}

// checkUpdate handles GET /api/update/check
func (h *Handler) checkUpdate(c *gin.Context) {
	info, err := h.service.CheckUpdate()
	if err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, info)
}

// getStatus handles GET /api/update/status
func (h *Handler) getStatus(c *gin.Context) {
	status := h.service.GetStatus()
	lastVersion := h.service.GetLastVersionInfo()

	response := gin.H{
		"status": status,
	}

	if lastVersion != nil {
		response["version_info"] = lastVersion
	}

	common.SuccessResponse(c, response)
}

// downloadUpdate handles POST /api/update/download
func (h *Handler) downloadUpdate(c *gin.Context) {
	var req struct {
		Version string `json:"version"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// If no version specified, use latest
		lastVersion := h.service.GetLastVersionInfo()
		if lastVersion != nil {
			req.Version = lastVersion.Latest
		}
	}

	if req.Version == "" {
		common.ErrorResponse(c, common.ErrInvalidRequest, gin.H{
			"error": "未指定版本号，请先检查更新",
		})
		return
	}

	// Start download in background
	go func() {
		h.service.DownloadUpdate(req.Version)
	}()

	common.SuccessResponse(c, gin.H{
		"message": "开始下载更新",
		"version": req.Version,
	})
}

// applyUpdate handles POST /api/update/apply
func (h *Handler) applyUpdate(c *gin.Context) {
	if err := h.service.ApplyUpdate(); err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "更新已应用，请重启服务",
	})
}

// rollback handles POST /api/update/rollback
func (h *Handler) rollback(c *gin.Context) {
	if err := h.service.Rollback(); err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "已回滚到之前版本",
	})
}
