// Package handler provides HTTP handlers for the container registry.
package handler

import (
	"net/http"

	"cyp-docker-registry/internal/service"

	"github.com/gin-gonic/gin"
)

// DNSHandler handles DNS resolution requests.
type DNSHandler struct {
	dnsService *service.DNSService
}

// NewDNSHandler creates a new DNSHandler instance.
func NewDNSHandler(dnsSvc *service.DNSService) *DNSHandler {
	return &DNSHandler{
		dnsService: dnsSvc,
	}
}

// RegisterRoutes registers DNS routes.
func (h *DNSHandler) RegisterRoutes(r *gin.RouterGroup) {
	dns := r.Group("/dns")
	{
		dns.POST("/resolve", h.Resolve)
		dns.GET("/resolve", h.ResolveGet)
	}
}

// ResolveRequest represents a DNS resolve request.
type ResolveRequest struct {
	Domain string `json:"domain" binding:"required"`
}

// Resolve handles DNS resolution via POST.
func (h *DNSHandler) Resolve(c *gin.Context) {
	var req ResolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数无效",
			"code":  "invalid_request",
		})
		return
	}

	result, err := h.dnsService.Resolve(req.Domain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"code":  "resolve_failed",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ResolveGet handles DNS resolution via GET.
func (h *DNSHandler) ResolveGet(c *gin.Context) {
	domain := c.Query("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请提供域名参数",
			"code":  "missing_domain",
		})
		return
	}

	result, err := h.dnsService.Resolve(domain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"code":  "resolve_failed",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}
