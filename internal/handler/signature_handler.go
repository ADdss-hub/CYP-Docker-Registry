// Package handler provides HTTP handlers for CYP-Registry.
package handler

import (
	"net/http"
	"strconv"

	"cyp-registry/internal/service"

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	signature, err := h.signatureService.SignImage(&req, user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		"message":   "Image signed successfully",
	})
}

// GetSignature retrieves a signature.
func (h *SignatureHandler) GetSignature(c *gin.Context) {
	imageRef := c.Param("imageRef")

	signature, err := h.signatureService.GetSignature(imageRef)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"signature": signature})
}

// VerifyImage verifies an image signature.
func (h *SignatureHandler) VerifyImage(c *gin.Context) {
	var req service.VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	result, err := h.signatureService.VerifyImage(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DeleteSignature deletes a signature.
func (h *SignatureHandler) DeleteSignature(c *gin.Context) {
	imageRef := c.Param("imageRef")

	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.signatureService.DeleteSignature(imageRef); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Signature deleted successfully"})
}
