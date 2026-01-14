// Package handler provides HTTP handlers for the container registry.
package handler

import (
	"net/http"
	"strconv"

	"cyp-docker-registry/internal/service"

	"github.com/gin-gonic/gin"
)

// OrgHandler handles organization requests.
type OrgHandler struct {
	orgService   *service.OrgService
	auditService *service.AuditService
}

// NewOrgHandler creates a new OrgHandler instance.
func NewOrgHandler(orgSvc *service.OrgService, auditSvc *service.AuditService) *OrgHandler {
	return &OrgHandler{
		orgService:   orgSvc,
		auditService: auditSvc,
	}
}

// RegisterRoutes registers organization routes.
func (h *OrgHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("", h.ListOrganizations)
	r.POST("", h.CreateOrganization)
	r.GET("/:id", h.GetOrganization)
	r.PUT("/:id", h.UpdateOrganization)
	r.DELETE("/:id", h.DeleteOrganization)
	r.GET("/:id/members", h.GetMembers)
	r.POST("/:id/members", h.AddMember)
	r.DELETE("/:id/members/:userId", h.RemoveMember)
}

// ListOrganizations lists all organizations.
func (h *OrgHandler) ListOrganizations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// Get current user
	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	// If not admin, only show user's organizations
	var orgs []*service.Organization
	var total int
	var err error

	if user.Role == "admin" {
		orgs, total, err = h.orgService.ListOrganizations(page, pageSize)
	} else {
		orgs, err = h.orgService.ListUserOrganizations(user.ID)
		total = len(orgs)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"organizations": orgs,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
	})
}

// CreateOrganization creates a new organization.
func (h *OrgHandler) CreateOrganization(c *gin.Context) {
	var req service.CreateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	org, err := h.orgService.CreateOrganization(&req, user.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"organization": org,
		"message":      "组织创建成功",
	})
}

// GetOrganization retrieves an organization by ID.
func (h *OrgHandler) GetOrganization(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的组织ID"})
		return
	}

	org, err := h.orgService.GetOrganization(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"organization": org})
}

// UpdateOrganization updates an organization.
func (h *OrgHandler) UpdateOrganization(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的组织ID"})
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	if err := h.orgService.UpdateOrganization(id, req.DisplayName, user.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "组织更新成功"})
}

// DeleteOrganization deletes an organization.
func (h *OrgHandler) DeleteOrganization(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的组织ID"})
		return
	}

	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	if err := h.orgService.DeleteOrganization(id, user.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "组织删除成功"})
}

// GetMembers retrieves members of an organization.
func (h *OrgHandler) GetMembers(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的组织ID"})
		return
	}

	members, err := h.orgService.GetMembers(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// AddMember adds a member to an organization.
func (h *OrgHandler) AddMember(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的组织ID"})
		return
	}

	var req struct {
		UserID int64  `json:"user_id" binding:"required"`
		Role   string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	if err := h.orgService.AddMember(id, req.UserID, user.ID, req.Role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "成员添加成功"})
}

// RemoveMember removes a member from an organization.
func (h *OrgHandler) RemoveMember(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的组织ID"})
		return
	}

	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	user := getCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	if err := h.orgService.RemoveMember(id, userID, user.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "成员移除成功"})
}

// Helper function to get current user from context
func getCurrentUser(c *gin.Context) *service.User {
	user, exists := c.Get("currentUser")
	if !exists {
		return nil
	}
	if u, ok := user.(*service.User); ok {
		return u
	}
	return nil
}
