// Package handler provides HTTP handlers for the container registry.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"container-registry/internal/dao"

	"github.com/gin-gonic/gin"
)

// AuditHandler handles audit log requests.
type AuditHandler struct{}

// NewAuditHandler creates a new AuditHandler instance.
func NewAuditHandler() *AuditHandler {
	return &AuditHandler{}
}

// RegisterRoutes registers audit routes.
func (h *AuditHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/logs", h.GetAuditLogs)
	r.GET("/logs/export", h.ExportAuditLogs)
}

// GetAuditLogs retrieves audit logs with pagination and filters.
func (h *AuditHandler) GetAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	eventType := c.Query("event_type")

	var startDate, endDate time.Time
	if s := c.Query("start_date"); s != "" {
		startDate, _ = time.Parse(time.RFC3339, s)
	}
	if e := c.Query("end_date"); e != "" {
		endDate, _ = time.Parse(time.RFC3339, e)
	}

	logs, total, err := dao.GetAuditLogs(page, pageSize, eventType, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to response format
	responseLogs := make([]map[string]interface{}, len(logs))
	for i, log := range logs {
		responseLogs[i] = map[string]interface{}{
			"id":              log.ID,
			"timestamp":       log.Timestamp,
			"level":           log.Level,
			"event":           log.Event,
			"user_id":         log.UserID.Int64,
			"username":        log.Username.String,
			"ip_address":      log.IPAddress,
			"resource":        log.Resource,
			"action":          log.Action,
			"status":          log.Status,
			"details":         log.Details,
			"blockchain_hash": log.BlockchainHash,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":      responseLogs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ExportAuditLogs exports audit logs as JSON.
func (h *AuditHandler) ExportAuditLogs(c *gin.Context) {
	var startDate, endDate time.Time
	if s := c.Query("start_date"); s != "" {
		startDate, _ = time.Parse(time.RFC3339, s)
	}
	if e := c.Query("end_date"); e != "" {
		endDate, _ = time.Parse(time.RFC3339, e)
	}

	// Get all logs within date range
	logs, _, err := dao.GetAuditLogs(1, 10000, "", startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to export format
	exportLogs := make([]map[string]interface{}, len(logs))
	for i, log := range logs {
		exportLogs[i] = map[string]interface{}{
			"id":              log.ID,
			"timestamp":       log.Timestamp.Format(time.RFC3339),
			"level":           log.Level,
			"event":           log.Event,
			"user_id":         log.UserID.Int64,
			"username":        log.Username.String,
			"ip_address":      log.IPAddress,
			"resource":        log.Resource,
			"action":          log.Action,
			"status":          log.Status,
			"details":         log.Details,
			"blockchain_hash": log.BlockchainHash,
		}
	}

	data, _ := json.MarshalIndent(exportLogs, "", "  ")

	c.Header("Content-Disposition", "attachment; filename=audit-logs.json")
	c.Header("Content-Type", "application/json")
	c.Data(http.StatusOK, "application/json", data)
}
