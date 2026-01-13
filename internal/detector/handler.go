// Package detector provides host system detection functionality.
package detector

import (
	"cyp-docker-registry/internal/common"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for system detection operations.
type Handler struct {
	service *DetectorService
}

// NewHandler creates a new detector handler.
func NewHandler(service *DetectorService) *Handler {
	return &Handler{
		service: service,
	}
}

// RegisterRoutes registers detector routes on the given router group.
func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/info", h.getSystemInfo)
	group.GET("/compatibility", h.checkCompatibility)
	group.GET("/refresh", h.refreshSystemInfo)
}

// getSystemInfo handles GET /api/system/info
func (h *Handler) getSystemInfo(c *gin.Context) {
	// Try to return cached info first for faster response
	if cached := h.service.GetCachedInfo(); cached != nil {
		common.SuccessResponse(c, cached)
		return
	}

	info, err := h.service.GetSystemInfo()
	if err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, info)
}

// checkCompatibility handles GET /api/system/compatibility
func (h *Handler) checkCompatibility(c *gin.Context) {
	report, err := h.service.CheckCompatibility()
	if err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, report)
}

// refreshSystemInfo handles GET /api/system/refresh
// Forces a refresh of system information.
func (h *Handler) refreshSystemInfo(c *gin.Context) {
	info, err := h.service.GetSystemInfo()
	if err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "系统信息已刷新",
		"info":    info,
	})
}
