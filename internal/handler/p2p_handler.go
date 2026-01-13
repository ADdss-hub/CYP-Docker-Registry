// Package handler 提供P2P HTTP处理器
package handler

import (
	"net/http"

	"container-registry/internal/service"

	"github.com/gin-gonic/gin"
)

// P2PHandler P2P处理器
type P2PHandler struct {
	p2pService *service.P2PService
}

// NewP2PHandler 创建P2P处理器
func NewP2PHandler(p2pService *service.P2PService) *P2PHandler {
	return &P2PHandler{
		p2pService: p2pService,
	}
}

// RegisterRoutes 注册路由
func (h *P2PHandler) RegisterRoutes(r *gin.RouterGroup) {
	p2p := r.Group("/p2p")
	{
		p2p.GET("/status", h.GetStatus)
		p2p.GET("/peers", h.GetPeers)
		p2p.POST("/peers/connect", h.ConnectPeer)
		p2p.DELETE("/peers/:id", h.DisconnectPeer)
		p2p.GET("/blobs", h.ListBlobs)
		p2p.GET("/blobs/:digest", h.GetBlob)
		p2p.POST("/blobs/:digest/announce", h.AnnounceBlob)
		p2p.POST("/enable", h.Enable)
		p2p.POST("/disable", h.Disable)
	}
}

// GetStatus 获取P2P状态
// @Summary 获取P2P状态
// @Tags P2P
// @Produce json
// @Success 200 {object} service.P2PStatus
// @Router /api/v1/p2p/status [get]
func (h *P2PHandler) GetStatus(c *gin.Context) {
	status := h.p2pService.GetStatus()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": status,
	})
}

// GetPeers 获取对等节点列表
// @Summary 获取对等节点列表
// @Tags P2P
// @Produce json
// @Success 200 {array} service.P2PPeerInfo
// @Router /api/v1/p2p/peers [get]
func (h *P2PHandler) GetPeers(c *gin.Context) {
	peers := h.p2pService.GetPeers()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": peers,
	})
}

// ConnectPeerRequest 连接节点请求
type ConnectPeerRequest struct {
	Address string `json:"address" binding:"required"`
}

// ConnectPeer 连接指定节点
// @Summary 连接指定节点
// @Tags P2P
// @Accept json
// @Produce json
// @Param request body ConnectPeerRequest true "连接请求"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/p2p/peers/connect [post]
func (h *P2PHandler) ConnectPeer(c *gin.Context) {
	var req ConnectPeerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求参数",
		})
		return
	}

	if err := h.p2pService.ConnectPeer(c.Request.Context(), req.Address); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "连接成功",
	})
}

// DisconnectPeer 断开指定节点
// @Summary 断开指定节点
// @Tags P2P
// @Produce json
// @Param id path string true "节点ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/p2p/peers/{id} [delete]
func (h *P2PHandler) DisconnectPeer(c *gin.Context) {
	peerID := c.Param("id")

	if err := h.p2pService.DisconnectPeer(peerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "已断开连接",
	})
}

// ListBlobs 列出本地Blob
// @Summary 列出本地Blob
// @Tags P2P
// @Produce json
// @Success 200 {array} string
// @Router /api/v1/p2p/blobs [get]
func (h *P2PHandler) ListBlobs(c *gin.Context) {
	blobs, err := h.p2pService.ListBlobs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": blobs,
	})
}

// GetBlob 获取Blob信息
// @Summary 获取Blob信息
// @Tags P2P
// @Produce json
// @Param digest path string true "Blob摘要"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/p2p/blobs/{digest} [get]
func (h *P2PHandler) GetBlob(c *gin.Context) {
	digest := c.Param("digest")

	// 检查本地是否有
	hasLocal := h.p2pService.HasLocalBlob(digest)

	// 检查P2P网络是否有
	hasP2P := h.p2pService.HasBlob(c.Request.Context(), digest)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"digest":    digest,
			"has_local": hasLocal,
			"has_p2p":   hasP2P,
		},
	})
}

// AnnounceBlob 宣布拥有Blob
// @Summary 宣布拥有Blob
// @Tags P2P
// @Produce json
// @Param digest path string true "Blob摘要"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/p2p/blobs/{digest}/announce [post]
func (h *P2PHandler) AnnounceBlob(c *gin.Context) {
	digest := c.Param("digest")

	if err := h.p2pService.AnnounceBlob(c.Request.Context(), digest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "已宣布",
	})
}

// Enable 启用P2P
// @Summary 启用P2P
// @Tags P2P
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/p2p/enable [post]
func (h *P2PHandler) Enable(c *gin.Context) {
	if err := h.p2pService.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "P2P已启用",
	})
}

// Disable 禁用P2P
// @Summary 禁用P2P
// @Tags P2P
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/p2p/disable [post]
func (h *P2PHandler) Disable(c *gin.Context) {
	if err := h.p2pService.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "P2P已禁用",
	})
}
