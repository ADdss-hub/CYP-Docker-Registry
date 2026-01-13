// Package handler provides HTTP handlers for the container registry.
package handler

import (
	"net/http"
	"strconv"

	"cyp-docker-registry/internal/service"

	"github.com/gin-gonic/gin"
)

// SBOMHandler handles SBOM requests.
type SBOMHandler struct {
	sbomService  *service.SBOMService
	auditService *service.AuditService
}

// NewSBOMHandler creates a new SBOMHandler instance.
func NewSBOMHandler(sbomSvc *service.SBOMService, auditSvc *service.AuditService) *SBOMHandler {
	return &SBOMHandler{
		sbomService:  sbomSvc,
		auditService: auditSvc,
	}
}

// RegisterRoutes registers SBOM routes.
func (h *SBOMHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("", h.ListSBOMs)
	r.POST("/generate", h.GenerateSBOM)
	r.GET("/:imageRef", h.GetSBOM)
	r.GET("/:imageRef/export", h.ExportSBOM)
	r.POST("/scan", h.ScanVulnerabilities)
	r.DELETE("/:imageRef", h.DeleteSBOM)
}

// ListSBOMs lists all SBOMs.
func (h *SBOMHandler) ListSBOMs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	sboms, total, err := h.sbomService.ListSBOMs(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sboms":     sboms,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GenerateSBOM generates a SBOM for an image.
func (h *SBOMHandler) GenerateSBOM(c *gin.Context) {
	var req service.GenerateSBOMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	sbom, err := h.sbomService.GenerateSBOM(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log SBOM generation
	if h.auditService != nil {
		user := getCurrentUser(c)
		var userID int64
		var username string
		if user != nil {
			userID = user.ID
			username = user.Username
		}

		h.auditService.LogAuditEvent(&service.AuditLog{
			Level:     "info",
			Event:     "sbom_generated",
			UserID:    userID,
			Username:  username,
			IPAddress: c.ClientIP(),
			Action:    "generate",
			Status:    "success",
			Details: map[string]interface{}{
				"image_ref": req.ImageRef,
				"format":    sbom.Format,
			},
		})
	}

	c.JSON(http.StatusCreated, gin.H{
		"sbom":    sbom,
		"message": "SBOM generated successfully",
	})
}

// GetSBOM retrieves a SBOM.
func (h *SBOMHandler) GetSBOM(c *gin.Context) {
	imageRef := c.Param("imageRef")

	sbom, err := h.sbomService.GetSBOM(imageRef)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sbom": sbom})
}

// ExportSBOM exports a SBOM.
func (h *SBOMHandler) ExportSBOM(c *gin.Context) {
	imageRef := c.Param("imageRef")
	format := c.DefaultQuery("format", "spdx-json")

	data, err := h.sbomService.ExportSBOM(imageRef, format)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	filename := "sbom-" + imageRef + "." + format
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/json")
	c.Data(http.StatusOK, "application/json", data)
}

// ScanVulnerabilities scans an image for vulnerabilities.
func (h *SBOMHandler) ScanVulnerabilities(c *gin.Context) {
	var req service.ScanVulnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	result, err := h.sbomService.ScanVulnerabilities(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log vulnerability scan
	if h.auditService != nil {
		user := getCurrentUser(c)
		var userID int64
		var username string
		if user != nil {
			userID = user.ID
			username = user.Username
		}

		h.auditService.LogAuditEvent(&service.AuditLog{
			Level:     "info",
			Event:     "vulnerability_scan",
			UserID:    userID,
			Username:  username,
			IPAddress: c.ClientIP(),
			Action:    "scan",
			Status:    "success",
			Details: map[string]interface{}{
				"image_ref": req.ImageRef,
				"total":     result.Summary.Total,
				"critical":  result.Summary.Critical,
			},
		})
	}

	c.JSON(http.StatusOK, result)
}

// DeleteSBOM deletes a SBOM.
func (h *SBOMHandler) DeleteSBOM(c *gin.Context) {
	imageRef := c.Param("imageRef")

	if err := h.sbomService.DeleteSBOM(imageRef); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "SBOM deleted successfully"})
}
