// Package handler provides HTTP handlers for CYP-Docker-Registry.
package handler

import (
	"net/http"
	"strconv"

	"cyp-docker-registry/internal/service"

	"github.com/gin-gonic/gin"
)

// SignatureHandler handles signature requests.
type SignatureHandler struct {
	signatureService *service.SignatureService
	auditService     *service.AuditService
}

// NewSignatureHandler creates a new SignatureHandler instance.
func NewSignatureHandler(sigSvc *service.SignatureService, auditSvc *service.AuditService) *SignatureHandler {
	return &SignatureHandler{
		signatureService: sigSvc,
		auditService:     auditSvc,
	}
}

// RegisterRoutes registers signature routes.
func (h *SignatureHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("", h.ListSignatures)
	r.POST("", h.SignImage)
	r.GET("/:imageRef", h.GetSignature)
	r.POST("/verify", h.VerifyImage)
	r.DELETE("/:imageRef", h.DeleteSignature)
}

// ListSignatures lists all signatures.
func (h *SignatureHandler) ListSignatures(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	signatures, total, err := h.signatureService.ListSignatures(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取签名列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"signatures": signatures,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// SignImage signs an image.
func (h *SignatureHandler) SignImage(c *gin.Context) {
	var req service.SignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
		return
	}

	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	signature, err := h.signatureService.SignImage(&req, user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "签名失败"})
		return
	}

	// Log signature event
	if h.auditService != nil {
		h.auditService.LogAuditEvent(&service.AuditLog{
			Level:     "info",
			Event:     "image_signed",
			UserID:    user.ID,
			Username:  user.Username,
			IPAddress: c.ClientIP(),
			Action:    "sign",
			Status:    "success",
			Details: map[string]interface{}{
				"image_ref": req.ImageRef,
			},
		})
	}

	c.JSON(http.StatusCreated, gin.H{
		"signature": signature,
		"message":   "镜像签名成功",
	})
}

// GetSignature retrieves a signature.
func (h *SignatureHandler) GetSignature(c *gin.Context) {
	imageRef := c.Param("imageRef")

	signature, err := h.signatureService.GetSignature(imageRef)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "签名不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"signature": signature})
}

// VerifyImage verifies an image signature.
func (h *SignatureHandler) VerifyImage(c *gin.Context) {
	var req service.VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
		return
	}

	result, err := h.signatureService.VerifyImage(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证失败"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DeleteSignature deletes a signature.
func (h *SignatureHandler) DeleteSignature(c *gin.Context) {
	imageRef := c.Param("imageRef")

	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	if err := h.signatureService.DeleteSignature(imageRef); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "删除签名失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "签名已删除"})
}
