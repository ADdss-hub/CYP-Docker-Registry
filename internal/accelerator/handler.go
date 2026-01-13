// Package accelerator provides image acceleration and caching functionality.
package accelerator

import (
	"container-registry/internal/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for accelerator operations.
type Handler struct {
	proxy *ProxyService
}

// NewHandler creates a new accelerator handler.
func NewHandler(proxy *ProxyService) *Handler {
	return &Handler{
		proxy: proxy,
	}
}

// RegisterRoutes registers accelerator routes on the given router group.
func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	// Proxy pull endpoints
	group.GET("/pull/:name/blobs/:digest", h.proxyPullBlob)
	group.GET("/pull/:name/manifests/:reference", h.proxyPullManifest)

	// Cache management endpoints
	cache := group.Group("/cache")
	{
		cache.GET("/stats", h.getCacheStats)
		cache.DELETE("", h.clearCache)
		cache.DELETE("/:digest", h.deleteCacheEntry)
		cache.GET("/entries", h.listCacheEntries)
	}

	// Upstream management endpoints
	upstreams := group.Group("/upstreams")
	{
		upstreams.GET("", h.listUpstreams)
		upstreams.POST("", h.addUpstream)
		upstreams.PUT("/:name", h.updateUpstream)
		upstreams.DELETE("/:name", h.removeUpstream)
		upstreams.POST("/:name/enable", h.enableUpstream)
		upstreams.POST("/:name/disable", h.disableUpstream)
		upstreams.GET("/:name/health", h.checkUpstreamHealth)
	}
}


// ============================================================================
// Proxy Pull Handlers
// ============================================================================

// proxyPullBlob handles GET /api/accel/pull/:name/blobs/:digest
func (h *Handler) proxyPullBlob(c *gin.Context) {
	name := c.Param("name")
	digest := c.Param("digest")

	reader, size, err := h.proxy.ProxyPull(name, digest)
	if err != nil {
		common.ErrorResponse(c, common.ErrUpstreamError, gin.H{
			"name":   name,
			"digest": digest,
			"error":  err.Error(),
		})
		return
	}
	defer reader.Close()

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Docker-Content-Digest", digest)
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.DataFromReader(200, size, "application/octet-stream", reader, nil)
}

// proxyPullManifest handles GET /api/accel/pull/:name/manifests/:reference
func (h *Handler) proxyPullManifest(c *gin.Context) {
	name := c.Param("name")
	reference := c.Param("reference")

	data, contentType, err := h.proxy.ProxyPullManifest(name, reference)
	if err != nil {
		common.ErrorResponse(c, common.ErrUpstreamError, gin.H{
			"name":      name,
			"reference": reference,
			"error":     err.Error(),
		})
		return
	}

	if contentType == "" {
		contentType = "application/vnd.docker.distribution.manifest.v2+json"
	}

	c.Header("Content-Type", contentType)
	c.Data(200, contentType, data)
}

// ============================================================================
// Cache Management Handlers
// ============================================================================

// getCacheStats handles GET /api/accel/cache/stats
func (h *Handler) getCacheStats(c *gin.Context) {
	stats := h.proxy.GetCache().Stats()
	common.SuccessResponse(c, stats)
}

// clearCache handles DELETE /api/accel/cache
func (h *Handler) clearCache(c *gin.Context) {
	if err := h.proxy.GetCache().Clear(); err != nil {
		common.ErrorResponse(c, common.ErrInternalError, gin.H{
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "缓存已清空",
	})
}

// deleteCacheEntry handles DELETE /api/accel/cache/:digest
func (h *Handler) deleteCacheEntry(c *gin.Context) {
	digest := c.Param("digest")

	if err := h.proxy.GetCache().Delete(digest); err != nil {
		common.ErrorResponse(c, common.ErrNotFound, gin.H{
			"digest": digest,
			"error":  err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "缓存条目已删除",
		"digest":  digest,
	})
}

// listCacheEntries handles GET /api/accel/cache/entries
func (h *Handler) listCacheEntries(c *gin.Context) {
	entries := h.proxy.GetCache().GetEntries()
	common.SuccessResponse(c, gin.H{
		"entries": entries,
		"count":   len(entries),
	})
}


// ============================================================================
// Upstream Management Handlers
// ============================================================================

// listUpstreams handles GET /api/accel/upstreams
func (h *Handler) listUpstreams(c *gin.Context) {
	upstreams := h.proxy.GetUpstreams()
	common.SuccessResponse(c, gin.H{
		"upstreams": upstreams,
		"count":     len(upstreams),
	})
}

// addUpstream handles POST /api/accel/upstreams
func (h *Handler) addUpstream(c *gin.Context) {
	var upstream UpstreamSource
	if err := c.ShouldBindJSON(&upstream); err != nil {
		common.ErrorResponse(c, common.ErrInvalidRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Set default enabled state
	upstream.Enabled = true

	if err := h.proxy.AddUpstream(upstream); err != nil {
		common.ErrorResponse(c, common.ErrInvalidRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message":  "上游源添加成功",
		"upstream": upstream,
	})
}

// updateUpstream handles PUT /api/accel/upstreams/:name
func (h *Handler) updateUpstream(c *gin.Context) {
	name := c.Param("name")

	var upstream UpstreamSource
	if err := c.ShouldBindJSON(&upstream); err != nil {
		common.ErrorResponse(c, common.ErrInvalidRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := h.proxy.UpdateUpstream(name, upstream); err != nil {
		common.ErrorResponse(c, common.ErrNotFound, gin.H{
			"name":  name,
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message":  "上游源更新成功",
		"upstream": upstream,
	})
}

// removeUpstream handles DELETE /api/accel/upstreams/:name
func (h *Handler) removeUpstream(c *gin.Context) {
	name := c.Param("name")

	if err := h.proxy.RemoveUpstream(name); err != nil {
		common.ErrorResponse(c, common.ErrNotFound, gin.H{
			"name":  name,
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "上游源删除成功",
		"name":    name,
	})
}

// enableUpstream handles POST /api/accel/upstreams/:name/enable
func (h *Handler) enableUpstream(c *gin.Context) {
	name := c.Param("name")

	if err := h.proxy.EnableUpstream(name, true); err != nil {
		common.ErrorResponse(c, common.ErrNotFound, gin.H{
			"name":  name,
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "上游源已启用",
		"name":    name,
	})
}

// disableUpstream handles POST /api/accel/upstreams/:name/disable
func (h *Handler) disableUpstream(c *gin.Context) {
	name := c.Param("name")

	if err := h.proxy.EnableUpstream(name, false); err != nil {
		common.ErrorResponse(c, common.ErrNotFound, gin.H{
			"name":  name,
			"error": err.Error(),
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "上游源已禁用",
		"name":    name,
	})
}

// checkUpstreamHealth handles GET /api/accel/upstreams/:name/health
func (h *Handler) checkUpstreamHealth(c *gin.Context) {
	name := c.Param("name")

	healthy, err := h.proxy.CheckUpstreamHealth(name)
	if err != nil {
		common.ErrorResponse(c, common.ErrNotFound, gin.H{
			"name":  name,
			"error": err.Error(),
		})
		return
	}

	status := "healthy"
	if !healthy {
		status = "unreachable"
	}

	common.SuccessResponse(c, gin.H{
		"name":    name,
		"healthy": healthy,
		"status":  status,
	})
}
