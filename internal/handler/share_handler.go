// Package handler provides HTTP handlers for the container registry.
package handler

import (
	"net/http"
	"strconv"

	"container-registry/internal/service"

	"github.com/gin-gonic/gin"
)

// ShareHandler handles share link requests.
type ShareHandler struct {
	shareService *service.ShareService
	auditService *service.AuditService
}

// NewShareHandler creates a new ShareHandler instance.
func NewShareHandler(shareSvc *service.ShareService, auditSvc *service.AuditService) *ShareHandler {
	return &ShareHandler{
		shareService: shareSvc,
		auditService: auditSvc,
	}
}

// RegisterRoutes registers share routes.
func (h *ShareHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("", h.ListShareLinks)
	r.POST("", h.CreateShareLink)
	r.GET("/:code", h.GetShareLink)
	r.POST("/:code/verify", h.VerifyPassword)
	r.DELETE("/:code", h.RevokeShareLink)
}

// ListShareLinks lists share links for the current user.
func (h *ShareHandler) ListShareLinks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	links, total, err := h.shareService.ListShareLinks(user.ID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"links":     links,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// CreateShareLink creates a new share link.
func (h *ShareHandler) CreateShareLink(c *gin.Context) {
	var req service.CreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	link, code, err := h.shareService.CreateShareLink(&req, user.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build share URL
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	shareURL := scheme + "://" + c.Request.Host + "/s/" + code

	c.JSON(http.StatusCreated, gin.H{
		"link":      link,
		"share_url": shareURL,
		"message":   "Share link created successfully",
	})
}

// GetShareLink retrieves a share link by code.
func (h *ShareHandler) GetShareLink(c *gin.Context) {
	code := c.Param("code")

	link, err := h.shareService.GetShareLink(code)
	if err != nil {
		status := http.StatusNotFound
		if err.Error() == "share link expired" {
			status = http.StatusGone
		} else if err.Error() == "share link usage limit exceeded" {
			status = http.StatusGone
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, link)
}

// VerifyPassword verifies the password for a share link.
func (h *ShareHandler) VerifyPassword(c *gin.Context) {
	code := c.Param("code")

	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := h.shareService.VerifySharePassword(code, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Increment usage count
	h.shareService.IncrementUsage(code)

	c.JSON(http.StatusOK, gin.H{"message": "Password verified"})
}

// RevokeShareLink revokes a share link.
func (h *ShareHandler) RevokeShareLink(c *gin.Context) {
	code := c.Param("code")

	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.shareService.RevokeShareLink(code, user.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Share link revoked successfully"})
}
