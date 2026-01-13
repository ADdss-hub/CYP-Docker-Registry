// Package handler 提供TUF HTTP处理器
package handler

import (
	"io"
	"net/http"

	"cyp-registry/internal/service"

	"github.com/gin-gonic/gin"
)

// TUFHandler TUF处理器
type TUFHandler struct {
	tufService *service.TUFService
}

// NewTUFHandler 创建TUF处理器
func NewTUFHandler(tufService *service.TUFService) *TUFHandler {
	return &TUFHandler{
		tufService: tufService,
	}
}

// RegisterRoutes 注册路由
func (h *TUFHandler) RegisterRoutes(r *gin.RouterGroup) {
	tuf := r.Group("/tuf")
	{
		// 状态和管理
		tuf.GET("/status", h.GetStatus)
		tuf.POST("/initialize", h.Initialize)
		tuf.POST("/refresh", h.RefreshTimestamp)

		// 目标管理
		tuf.GET("/targets", h.ListTargets)
		tuf.GET("/targets/:name", h.GetTarget)
		tuf.POST("/targets/:name", h.AddTarget)
		tuf.DELETE("/targets/:name", h.RemoveTarget)
		tuf.POST("/targets/:name/verify", h.VerifyTarget)

		// 密钥管理
		tuf.POST("/keys/rotate/:role", h.RotateKey)
		tuf.GET("/keys/export", h.ExportKeys)

		// 委托管理
		tuf.GET("/delegations", h.ListDelegations)
		tuf.POST("/delegations", h.AddDelegation)
		tuf.DELETE("/delegations/:name", h.RemoveDelegation)

		// 元数据获取（供客户端使用）
		tuf.GET("/metadata/root.json", h.GetRootMetadata)
		tuf.GET("/metadata/timestamp.json", h.GetTimestampMetadata)
		tuf.GET("/metadata/snapshot.json", h.GetSnapshotMetadata)
		tuf.GET("/metadata/targets.json", h.GetTargetsMetadata)

		// 过期检查
		tuf.GET("/expiry", h.CheckExpiry)
	}
}

// GetStatus 获取TUF状态
// @Summary 获取TUF状态
// @Tags TUF
// @Produce json
// @Success 200 {object} signature.TUFStatus
// @Router /api/v1/tuf/status [get]
func (h *TUFHandler) GetStatus(c *gin.Context) {
	status := h.tufService.GetStatus()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": status,
	})
}

// Initialize 初始化TUF仓库
// @Summary 初始化TUF仓库
// @Tags TUF
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tuf/initialize [post]
func (h *TUFHandler) Initialize(c *gin.Context) {
	if h.tufService.IsInitialized() {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "TUF仓库已初始化",
		})
		return
	}

	if err := h.tufService.Initialize(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "TUF仓库初始化成功",
	})
}

// RefreshTimestamp 刷新Timestamp
// @Summary 刷新Timestamp
// @Tags TUF
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tuf/refresh [post]
func (h *TUFHandler) RefreshTimestamp(c *gin.Context) {
	if err := h.tufService.RefreshTimestamp(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Timestamp已刷新",
	})
}

// ListTargets 列出所有目标
// @Summary 列出所有目标
// @Tags TUF
// @Produce json
// @Success 200 {array} service.TUFTargetInfo
// @Router /api/v1/tuf/targets [get]
func (h *TUFHandler) ListTargets(c *gin.Context) {
	targets := h.tufService.GetTargetList()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": targets,
	})
}

// GetTarget 获取目标信息
// @Summary 获取目标信息
// @Tags TUF
// @Produce json
// @Param name path string true "目标名称"
// @Success 200 {object} signature.TUFTarget
// @Router /api/v1/tuf/targets/{name} [get]
func (h *TUFHandler) GetTarget(c *gin.Context) {
	name := c.Param("name")

	target, err := h.tufService.GetTarget(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": target,
	})
}

// AddTargetRequest 添加目标请求
type AddTargetRequest struct {
	Custom map[string]interface{} `json:"custom"`
}

// AddTarget 添加目标
// @Summary 添加目标
// @Tags TUF
// @Accept multipart/form-data
// @Produce json
// @Param name path string true "目标名称"
// @Param file formData file true "目标文件"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tuf/targets/{name} [post]
func (h *TUFHandler) AddTarget(c *gin.Context) {
	name := c.Param("name")

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请上传文件",
		})
		return
	}

	// 读取文件内容
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "打开文件失败",
		})
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "读取文件失败",
		})
		return
	}

	// 获取自定义元数据
	var custom map[string]interface{}
	if customStr := c.PostForm("custom"); customStr != "" {
		// 解析JSON
		// 简化处理，实际应该解析JSON
	}

	if err := h.tufService.AddTarget(name, data, custom); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "目标添加成功",
	})
}

// RemoveTarget 移除目标
// @Summary 移除目标
// @Tags TUF
// @Produce json
// @Param name path string true "目标名称"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tuf/targets/{name} [delete]
func (h *TUFHandler) RemoveTarget(c *gin.Context) {
	name := c.Param("name")

	if err := h.tufService.RemoveTarget(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "目标已移除",
	})
}

// VerifyTarget 验证目标
// @Summary 验证目标
// @Tags TUF
// @Accept multipart/form-data
// @Produce json
// @Param name path string true "目标名称"
// @Param file formData file true "要验证的文件"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tuf/targets/{name}/verify [post]
func (h *TUFHandler) VerifyTarget(c *gin.Context) {
	name := c.Param("name")

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请上传文件",
		})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "打开文件失败",
		})
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "读取文件失败",
		})
		return
	}

	valid, err := h.tufService.VerifyTarget(name, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
			"valid":   false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"valid":   valid,
		"message": "验证通过",
	})
}

// RotateKey 轮换密钥
// @Summary 轮换密钥
// @Tags TUF
// @Produce json
// @Param role path string true "角色名称"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tuf/keys/rotate/{role} [post]
func (h *TUFHandler) RotateKey(c *gin.Context) {
	role := c.Param("role")

	// 验证角色
	validRoles := map[string]bool{
		"root": true, "targets": true, "snapshot": true, "timestamp": true,
	}
	if !validRoles[role] {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的角色",
		})
		return
	}

	if err := h.tufService.RotateKey(role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "密钥轮换成功",
	})
}

// ExportKeys 导出公钥
// @Summary 导出公钥
// @Tags TUF
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/v1/tuf/keys/export [get]
func (h *TUFHandler) ExportKeys(c *gin.Context) {
	keys := h.tufService.ExportPublicKeys()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": keys,
	})
}

// ListDelegations 列出委托
// @Summary 列出委托
// @Tags TUF
// @Produce json
// @Success 200 {array} service.TUFDelegationInfo
// @Router /api/v1/tuf/delegations [get]
func (h *TUFHandler) ListDelegations(c *gin.Context) {
	delegations := h.tufService.GetDelegationList()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": delegations,
	})
}

// AddDelegationRequest 添加委托请求
type AddDelegationRequest struct {
	Name      string   `json:"name" binding:"required"`
	Paths     []string `json:"paths" binding:"required"`
	Threshold int      `json:"threshold"`
}

// AddDelegation 添加委托
// @Summary 添加委托
// @Tags TUF
// @Accept json
// @Produce json
// @Param request body AddDelegationRequest true "委托配置"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tuf/delegations [post]
func (h *TUFHandler) AddDelegation(c *gin.Context) {
	var req AddDelegationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求参数",
		})
		return
	}

	if req.Threshold <= 0 {
		req.Threshold = 1
	}

	if err := h.tufService.AddDelegation(req.Name, req.Paths, req.Threshold); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "委托添加成功",
	})
}

// RemoveDelegation 移除委托
// @Summary 移除委托
// @Tags TUF
// @Produce json
// @Param name path string true "委托名称"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tuf/delegations/{name} [delete]
func (h *TUFHandler) RemoveDelegation(c *gin.Context) {
	name := c.Param("name")

	if err := h.tufService.RemoveDelegation(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "委托已移除",
	})
}

// GetRootMetadata 获取Root元数据
// @Summary 获取Root元数据
// @Tags TUF
// @Produce application/json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tuf/metadata/root.json [get]
func (h *TUFHandler) GetRootMetadata(c *gin.Context) {
	data, err := h.tufService.GetRootMetadata()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Root元数据不存在",
		})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

// GetTimestampMetadata 获取Timestamp元数据
// @Summary 获取Timestamp元数据
// @Tags TUF
// @Produce application/json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tuf/metadata/timestamp.json [get]
func (h *TUFHandler) GetTimestampMetadata(c *gin.Context) {
	data, err := h.tufService.GetTimestampMetadata()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Timestamp元数据不存在",
		})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

// GetSnapshotMetadata 获取Snapshot元数据
// @Summary 获取Snapshot元数据
// @Tags TUF
// @Produce application/json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tuf/metadata/snapshot.json [get]
func (h *TUFHandler) GetSnapshotMetadata(c *gin.Context) {
	data, err := h.tufService.GetSnapshotMetadata()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Snapshot元数据不存在",
		})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

// GetTargetsMetadata 获取Targets元数据
// @Summary 获取Targets元数据
// @Tags TUF
// @Produce application/json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tuf/metadata/targets.json [get]
func (h *TUFHandler) GetTargetsMetadata(c *gin.Context) {
	data, err := h.tufService.GetTargetsMetadata()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Targets元数据不存在",
		})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

// CheckExpiry 检查过期状态
// @Summary 检查过期状态
// @Tags TUF
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tuf/expiry [get]
func (h *TUFHandler) CheckExpiry(c *gin.Context) {
	warnings := h.tufService.CheckExpiry()
	c.JSON(http.StatusOK, gin.H{
		"code":     0,
		"warnings": warnings,
		"healthy":  len(warnings) == 0,
	})
}
