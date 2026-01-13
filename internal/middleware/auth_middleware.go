// Package middleware provides security middleware for CYP-Registry.
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var logger *zap.Logger

// InitLogger initializes the middleware logger.
func InitLogger(l *zap.Logger) {
	logger = l
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	Enabled          bool
	MaxLoginAttempts int
	MaxTokenAttempts int
	MaxAPIAttempts   int
	LockDuration     time.Duration
	EnforceIPBinding bool
}

// AuthMiddleware provides authentication middleware functionality.
type AuthMiddleware struct {
	config       *AuthConfig
	lockService  LockServiceInterface
	authService  AuthServiceInterface
	auditService AuditServiceInterface
}

// LockServiceInterface defines lock service methods.
type LockServiceInterface interface {
	IsSystemLocked() bool
	GetLockReason() string
	LockSystem(reason, ip string) error
}

// AuthServiceInterface defines auth service methods.
type AuthServiceInterface interface {
	ValidateJWT(token string) (*UserInfo, error)
	ValidateToken(token string) (*UserInfo, *TokenInfo, error)
	GetSession(userID int64) *SessionInfo
	TerminateSession(userID int64) error
	UpdateTokenLastUsed(tokenID int64) error
}

// AuditServiceInterface defines audit service methods.
type AuditServiceInterface interface {
	LogAccessAttempt(attempt *AccessAttemptInfo) error
	IncrementFailedAttempt(ip, code string)
	ShouldLock(ip string) bool
}

// UserInfo represents user information.
type UserInfo struct {
	ID       int64
	Username string
	Role     string
	IsActive bool
}

// TokenInfo represents token information.
type TokenInfo struct {
	ID     int64
	Name   string
	Scopes []string
}

// SessionInfo represents session information.
type SessionInfo struct {
	ID        string
	UserID    int64
	IP        string
	ExpiresAt time.Time
}

// AccessAttemptInfo represents access attempt information.
type AccessAttemptInfo struct {
	IPAddress string
	UserAgent string
	UserID    int64
	Action    string
	Resource  string
	Status    string
	ErrorMsg  string
	CreatedAt time.Time
}

// Whitelist paths that don't require authentication.
var authWhitelist = []string{
	"/api/v1/auth/login",
	"/api/v1/auth/logout",
	"/api/v1/auth/verify-token",
	"/api/v1/auth/heartbeat",
	"/api/v1/system/health",
	"/api/version",
	"/health",
	"/metrics",
}

// NewAuthMiddleware creates a new AuthMiddleware instance.
func NewAuthMiddleware(config *AuthConfig, lockSvc LockServiceInterface, authSvc AuthServiceInterface, auditSvc AuditServiceInterface) *AuthMiddleware {
	return &AuthMiddleware{
		config:       config,
		lockService:  lockSvc,
		authService:  authSvc,
		auditService: auditSvc,
	}
}

// ForceAuth returns a middleware that enforces authentication.
func (m *AuthMiddleware) ForceAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if system is locked
		if m.lockService != nil && m.lockService.IsSystemLocked() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":       "System is locked",
				"details":     "system_locked",
				"lock_reason": m.lockService.GetLockReason(),
			})
			return
		}

		// Check whitelist
		path := c.Request.URL.Path
		for _, wp := range authWhitelist {
			if path == wp {
				c.Next()
				return
			}
		}

		// Check share link access
		if strings.HasPrefix(path, "/s/") {
			m.handleShareAccess(c)
			return
		}

		// Check authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			m.handleUnauthorized(c, "Missing Authorization header", "no_auth_header")
			return
		}

		// Validate JWT or Token
		var user *UserInfo
		var token *TokenInfo

		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			var err error
			user, err = m.authService.ValidateJWT(tokenStr)
			if err != nil {
				m.logUnauthorizedAttempt(c, "Invalid JWT: "+err.Error())
				m.handleUnauthorized(c, "Invalid JWT token", "invalid_jwt")
				return
			}
		} else if strings.HasPrefix(authHeader, "Token ") {
			tokenStr := strings.TrimPrefix(authHeader, "Token ")
			var err error
			user, token, err = m.authService.ValidateToken(tokenStr)
			if err != nil {
				m.logUnauthorizedAttempt(c, "Invalid token: "+err.Error())
				m.handleUnauthorized(c, "Invalid token", "invalid_token")
				return
			}
		} else {
			m.logUnauthorizedAttempt(c, "Invalid Authorization format")
			m.handleUnauthorized(c, "Invalid Authorization format", "invalid_format")
			return
		}

		// Validate user status
		if !user.IsActive {
			m.logUnauthorizedAttempt(c, "User is inactive")
			m.handleUnauthorized(c, "User is inactive", "inactive_user")
			return
		}

		// IP binding check
		if m.config.EnforceIPBinding {
			if session := m.authService.GetSession(user.ID); session != nil && session.IP != c.ClientIP() {
				m.logUnauthorizedAttempt(c, "IP mismatch")
				m.handleUnauthorized(c, "IP address changed during session", "ip_mismatch")
				m.authService.TerminateSession(user.ID)
				return
			}
		}

		// Update token last used time
		if token != nil {
			m.authService.UpdateTokenLastUsed(token.ID)
		}

		// Set context
		c.Set("currentUser", user)
		c.Set("currentToken", token)
		c.Next()
	}
}

// handleShareAccess handles share link access.
func (m *AuthMiddleware) handleShareAccess(c *gin.Context) {
	// Share links have their own authentication flow
	c.Next()
}

// handleUnauthorized handles unauthorized access.
func (m *AuthMiddleware) handleUnauthorized(c *gin.Context, message, code string) {
	if m.auditService != nil {
		m.auditService.LogAccessAttempt(&AccessAttemptInfo{
			IPAddress: c.ClientIP(),
			UserAgent: c.GetHeader("User-Agent"),
			Action:    "unauthorized_access",
			Resource:  c.Request.URL.Path,
			Status:    "failure",
			ErrorMsg:  message,
			CreatedAt: time.Now(),
		})

		m.auditService.IncrementFailedAttempt(c.ClientIP(), code)

		if m.auditService.ShouldLock(c.ClientIP()) && m.lockService != nil {
			m.lockService.LockSystem("too_many_failed_attempts", c.ClientIP())
		}
	}

	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": message,
		"code":  code,
	})
}

// logUnauthorizedAttempt logs unauthorized access attempts.
func (m *AuthMiddleware) logUnauthorizedAttempt(c *gin.Context, reason string) {
	if logger != nil {
		logger.Warn("Unauthorized access attempt",
			zap.String("ip", c.ClientIP()),
			zap.String("path", c.Request.URL.Path),
			zap.String("user_agent", c.GetHeader("User-Agent")),
			zap.String("reason", reason),
		)
	}
}
