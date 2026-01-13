// Package middleware provides security middleware for CYP-Registry.
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// SecurityMiddleware provides security headers and CSRF protection.
type SecurityMiddleware struct {
	csrfEnabled bool
	csrfTokens  map[string]time.Time
}

// NewSecurityMiddleware creates a new SecurityMiddleware instance.
func NewSecurityMiddleware(csrfEnabled bool) *SecurityMiddleware {
	return &SecurityMiddleware{
		csrfEnabled: csrfEnabled,
		csrfTokens:  make(map[string]time.Time),
	}
}

// SecurityHeaders returns a middleware that adds security headers.
func (m *SecurityMiddleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		// Enable XSS filter
		c.Header("X-XSS-Protection", "1; mode=block")
		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		// Strict Transport Security (for HTTPS)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// Permissions Policy
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// CSRF returns a middleware that provides CSRF protection.
func (m *SecurityMiddleware) CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.csrfEnabled {
			c.Next()
			return
		}

		// Skip CSRF for safe methods
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Skip CSRF for API endpoints with Bearer token
		if c.GetHeader("Authorization") != "" {
			c.Next()
			return
		}

		// Validate CSRF token
		token := c.GetHeader("X-CSRF-Token")
		if token == "" {
			token = c.PostForm("_csrf")
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "CSRF token missing",
				"code":  "csrf_missing",
			})
			return
		}

		if !m.validateCSRFToken(token) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Invalid CSRF token",
				"code":  "csrf_invalid",
			})
			return
		}

		c.Next()
	}
}

// GenerateCSRFToken generates a new CSRF token.
func (m *SecurityMiddleware) GenerateCSRFToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	token := hex.EncodeToString(bytes)

	m.csrfTokens[token] = time.Now().Add(24 * time.Hour)
	m.cleanupExpiredTokens()

	return token
}

// validateCSRFToken validates a CSRF token.
func (m *SecurityMiddleware) validateCSRFToken(token string) bool {
	expiry, exists := m.csrfTokens[token]
	if !exists {
		return false
	}

	if time.Now().After(expiry) {
		delete(m.csrfTokens, token)
		return false
	}

	return true
}

// cleanupExpiredTokens removes expired CSRF tokens.
func (m *SecurityMiddleware) cleanupExpiredTokens() {
	now := time.Now()
	for token, expiry := range m.csrfTokens {
		if now.After(expiry) {
			delete(m.csrfTokens, token)
		}
	}
}

// RateLimiter provides rate limiting functionality.
type RateLimiter struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// RateLimit returns a middleware that limits request rate.
func (r *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		// Clean old requests
		r.cleanupOldRequests(ip, now)

		// Check rate limit
		if len(r.requests[ip]) >= r.limit {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too many requests",
				"code":        "rate_limit_exceeded",
				"retry_after": r.window.Seconds(),
			})
			return
		}

		// Record request
		r.requests[ip] = append(r.requests[ip], now)

		c.Next()
	}
}

// cleanupOldRequests removes requests outside the time window.
func (r *RateLimiter) cleanupOldRequests(ip string, now time.Time) {
	cutoff := now.Add(-r.window)
	var valid []time.Time

	for _, t := range r.requests[ip] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	r.requests[ip] = valid
}
