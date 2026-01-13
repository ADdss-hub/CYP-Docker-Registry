// Package handler provides HTTP handlers for the container registry.
package handler

import (
	"net/http"
	"time"

	"cyp-registry/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication requests.
type AuthHandler struct {
	authService      *service.AuthService
	lockService      *service.LockService
	intrusionService *service.IntrusionService
	auditService     *service.AuditService
}

// NewAuthHandler creates a new AuthHandler instance.
func NewAuthHandler(
	authSvc *service.AuthService,
	lockSvc *service.LockService,
	intrusionSvc *service.IntrusionService,
	auditSvc *service.AuditService,
) *AuthHandler {
	return &AuthHandler{
		authService:      authSvc,
		lockService:      lockSvc,
		intrusionService: intrusionSvc,
		auditService:     auditSvc,
	}
}

// RegisterRoutes registers auth routes.
func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/login", h.Login)
	r.POST("/logout", h.Logout)
	r.POST("/verify-token", h.VerifyToken)
	r.GET("/heartbeat", h.Heartbeat)
	r.GET("/me", h.GetCurrentUser)
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Captcha  string `json:"captcha,omitempty"`
}

// Login handles user login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
			"code":  "invalid_request",
		})
		return
	}

	clientIP := c.ClientIP()

	// Check if system is locked
	if h.lockService != nil && h.lockService.IsSystemLocked() {
		c.JSON(http.StatusForbidden, gin.H{
			"error":       "System is locked",
			"details":     "system_locked",
			"lock_reason": h.lockService.GetLockReason(),
		})
		return
	}

	// Check progressive delay
	if h.intrusionService != nil {
		delay := h.intrusionService.GetProgressiveDelay(clientIP)
		if delay > 0 {
			time.Sleep(delay)
		}
	}

	// Attempt login
	loginReq := &service.LoginRequest{
		Username: req.Username,
		Password: req.Password,
		ClientIP: clientIP,
	}

	resp, err := h.authService.Login(loginReq)
	if err != nil {
		// Log failed attempt
		if h.intrusionService != nil {
			h.intrusionService.IncrementFailedAttempt(clientIP, "login_failure")
		}

		if h.auditService != nil {
			h.auditService.LogAuthFailure(clientIP, req.Username, err.Error())
		}

		// Get remaining attempts
		remaining := 3
		if h.intrusionService != nil {
			remaining = h.intrusionService.GetRemainingAttempts(clientIP, "login_failure")
		}

		c.JSON(http.StatusUnauthorized, gin.H{
			"error":              "Invalid credentials",
			"code":               "login_failure",
			"remaining_attempts": remaining,
		})
		return
	}

	// Reset failed attempts on successful login
	if h.intrusionService != nil {
		h.intrusionService.ResetAttempts(clientIP)
	}

	// Log successful login
	if h.auditService != nil {
		h.auditService.LogAuditEvent(&service.AuditLog{
			Level:     "info",
			Event:     "login_success",
			UserID:    resp.User.ID,
			Username:  resp.User.Username,
			IPAddress: clientIP,
			Action:    "login",
			Status:    "success",
		})
	}

	c.JSON(http.StatusOK, resp)
}

// Logout handles user logout.
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get current user from context
	user, exists := c.Get("currentUser")
	if exists {
		u := user.(*service.User)
		h.authService.TerminateSession(u.ID)

		if h.auditService != nil {
			h.auditService.LogAuditEvent(&service.AuditLog{
				Level:     "info",
				Event:     "logout",
				UserID:    u.ID,
				Username:  u.Username,
				IPAddress: c.ClientIP(),
				Action:    "logout",
				Status:    "success",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// VerifyToken verifies a JWT token.
func (h *AuthHandler) VerifyToken(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
			"code":  "invalid_request",
		})
		return
	}

	user, err := h.authService.ValidateJWT(req.Token)
	if err != nil {
		if h.intrusionService != nil {
			h.intrusionService.IncrementFailedAttempt(c.ClientIP(), "invalid_jwt")
		}

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token",
			"code":  "invalid_token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"user":  user,
	})
}

// Heartbeat handles session heartbeat.
func (h *AuthHandler) Heartbeat(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}

// GetCurrentUser returns the current authenticated user.
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	user, exists := c.Get("currentUser")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Not authenticated",
			"code":  "not_authenticated",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}
