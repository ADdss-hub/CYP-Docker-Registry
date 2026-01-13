// Package handler provides HTTP handlers for the container registry.
package handler

import (
	"net/http"
	"strconv"

	"container-registry/internal/service"

	"github.com/gin-gonic/gin"
)

// TokenHandler handles personal access token requests.
type TokenHandler struct {
	tokenService *service.TokenService
	auditService *service.AuditService
}

// NewTokenHandler creates a new TokenHandler instance.
func NewTokenHandler(tokenSvc *service.TokenService, auditSvc *service.AuditService) *TokenHandler {
	return &TokenHandler{
		tokenService: tokenSvc,
		auditService: auditSvc,
	}
}

// RegisterRoutes registers token routes.
func (h *TokenHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("", h.ListTokens)
	r.POST("", h.CreateToken)
	r.DELETE("/:id", h.DeleteToken)
}

// ListTokens lists all tokens for the current user.
func (h *TokenHandler) ListTokens(c *gin.Context) {
	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tokens, err := h.tokenService.ListTokens(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

// CreateToken creates a new personal access token.
func (h *TokenHandler) CreateToken(c *gin.Context) {
	var req service.CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	resp, err := h.tokenService.CreateToken(&req, user.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log token creation
	if h.auditService != nil {
		h.auditService.LogAuditEvent(&service.AuditLog{
			Level:     "info",
			Event:     "token_created",
			UserID:    user.ID,
			Username:  user.Username,
			IPAddress: c.ClientIP(),
			Action:    "create",
			Status:    "success",
			Details: map[string]interface{}{
				"token_name": req.Name,
				"scopes":     req.Scopes,
			},
		})
	}

	c.JSON(http.StatusCreated, gin.H{
		"token":       resp.Token,
		"plain_token": resp.PlainToken,
		"message":     "Token created successfully. Please save the token now, it won't be shown again.",
	})
}

// DeleteToken deletes a personal access token.
func (h *TokenHandler) DeleteToken(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}

	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.tokenService.DeleteToken(id, user.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log token deletion
	if h.auditService != nil {
		h.auditService.LogAuditEvent(&service.AuditLog{
			Level:     "info",
			Event:     "token_deleted",
			UserID:    user.ID,
			Username:  user.Username,
			IPAddress: c.ClientIP(),
			Action:    "delete",
			Status:    "success",
			Details: map[string]interface{}{
				"token_id": id,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token deleted successfully"})
}
