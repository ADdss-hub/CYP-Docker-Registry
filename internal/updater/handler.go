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
	group.GET("/config", h.getConfig)
	group.PUT("/config", h.updateConfig)
	group.POST("/download", h.downloadUpdate)
	group.POST("/apply", h.applyUpdate)
	group.POST("/rollback", h.rollback)
	group.GET("/docker-command", h.getDockerCommand)
	group.GET("/watchtower-config", h.getWatchtowerConfig)
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
		"status":    status,
		"is_docker": h.service.IsDocker(),
	}

	if lastVersion != nil {
		response["version_info"] = lastVersion
	}

	common.SuccessResponse(c, response)
}

// getConfig handles GET /api/update/config
func (h *Handler) getConfig(c *gin.Context) {
	config := h.service.GetConfig()
	common.SuccessResponse(c, gin.H{
		"config":    config,
		"is_docker": h.service.IsDocker(),
	})
}

// updateConfig handles PUT /api/update/config
func (h *Handler) updateConfig(c *gin.Context) {
	var config UpdateConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		common.ErrorResponse(c, common.ErrInvalidRequest, gin.H{
			"error": "无效的配置参数",
		})
		return
	}

	h.service.SetConfig(config)

	common.SuccessResponse(c, gin.H{
		"message": "配置已更新",
		"config":  config,
	})
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

	// Check if running in Docker
	if h.service.IsDocker() {
		common.SuccessResponse(c, gin.H{
			"message":        "Docker 容器无法直接下载更新",
			"is_docker":      true,
			"docker_command": h.service.GetDockerUpdateCommand(),
			"tip":            "请使用 docker pull 命令更新镜像，或配置 Watchtower 实现自动更新",
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
	// Check if running in Docker
	if h.service.IsDocker() {
		common.SuccessResponse(c, gin.H{
			"message":           "Docker 容器无法自动应用更新",
			"is_docker":         true,
			"docker_command":    h.service.GetDockerUpdateCommand(),
			"watchtower_config": h.service.GetWatchtowerConfig(),
			"instructions": []string{
				"方式一: 手动更新",
				"  1. 停止容器: docker-compose down",
				"  2. 拉取新镜像: docker pull cyp/docker-registry:latest",
				"  3. 启动容器: docker-compose up -d",
				"",
				"方式二: 使用 Watchtower 自动更新",
				"  将 Watchtower 配置添加到 docker-compose.yaml",
			},
		})
		return
	}

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
	if h.service.IsDocker() {
		common.SuccessResponse(c, gin.H{
			"message":   "Docker 容器请手动回滚",
			"is_docker": true,
			"instructions": []string{
				"1. 停止容器: docker-compose down",
				"2. 拉取指定版本: docker pull cyp/docker-registry:<version>",
				"3. 修改 docker-compose.yaml 中的镜像版本",
				"4. 启动容器: docker-compose up -d",
			},
		})
		return
	}

	if err := h.service.Rollback(); err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "已回滚到之前版本，请重启服务",
	})
}

// getDockerCommand handles GET /api/update/docker-command
func (h *Handler) getDockerCommand(c *gin.Context) {
	if !h.service.IsDocker() {
		common.SuccessResponse(c, gin.H{
			"is_docker": false,
			"message":   "当前不是 Docker 环境",
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"is_docker":      true,
		"docker_command": h.service.GetDockerUpdateCommand(),
	})
}

// getWatchtowerConfig handles GET /api/update/watchtower-config
func (h *Handler) getWatchtowerConfig(c *gin.Context) {
	common.SuccessResponse(c, gin.H{
		"config":      h.service.GetWatchtowerConfig(),
		"description": "将此配置添加到 docker-compose.yaml 以启用自动更新",
	})
}
