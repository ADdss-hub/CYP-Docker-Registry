// Package model defines all data models for CYP-Docker-Registry.
package model

import (
	"time"
)

// User represents a system user.
type User struct {
	ID           int64     `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Email        string    `json:"email" db:"email"`
	Role         string    `json:"role" db:"role"` // admin, user, readonly
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	LastLoginAt  time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
}

// Session represents a user session.
type Session struct {
	ID        string    `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	IP        string    `json:"ip" db:"ip"`
	UserAgent string    `json:"user_agent" db:"user_agent"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
}

// PersonalAccessToken represents a personal access token.
type PersonalAccessToken struct {
	ID         int64     `json:"id" db:"id"`
	UserID     int64     `json:"user_id" db:"user_id"`
	Name       string    `json:"name" db:"name"`
	TokenHash  string    `json:"-" db:"token_hash"`
	Scopes     []string  `json:"scopes" db:"scopes"`
	ExpiresAt  time.Time `json:"expires_at" db:"expires_at"`
	LastUsedAt time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// AccessAttempt represents an access attempt for audit logging.
type AccessAttempt struct {
	ID             int64     `json:"id" db:"id"`
	IPAddress      string    `json:"ip_address" db:"ip_address"`
	UserAgent      string    `json:"user_agent" db:"user_agent"`
	UserID         int64     `json:"user_id,omitempty" db:"user_id"`
	Action         string    `json:"action" db:"action"`
	Resource       string    `json:"resource" db:"resource"`
	Status         string    `json:"status" db:"status"` // success, failure
	ErrorMsg       string    `json:"error_msg,omitempty" db:"error_msg"`
	BlockchainHash string    `json:"blockchain_hash,omitempty" db:"blockchain_hash"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// LockStatus represents the system lock status.
type LockStatus struct {
	IsLocked      bool      `json:"is_locked" db:"is_locked"`
	LockReason    string    `json:"lock_reason" db:"lock_reason"`
	LockType      string    `json:"lock_type" db:"lock_type"` // bypass_attempt, rule_triggered
	LockedAt      time.Time `json:"locked_at" db:"locked_at"`
	LockedByIP    string    `json:"locked_by_ip" db:"locked_by_ip"`
	LockedByUser  string    `json:"locked_by_user,omitempty" db:"locked_by_user"`
	UnlockAt      time.Time `json:"unlock_at,omitempty" db:"unlock_at"`
	RequireManual bool      `json:"require_manual" db:"require_manual"`
}

// IntrusionRule represents an intrusion detection rule.
type IntrusionRule struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Action      string `json:"action" yaml:"action"` // lock, warn, ban
	Threshold   int    `json:"threshold" yaml:"threshold"`
}

// Organization represents a team organization.
type Organization struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	DisplayName string    `json:"display_name" db:"display_name"`
	OwnerID     int64     `json:"owner_id" db:"owner_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ShareLink represents a share link for images.
type ShareLink struct {
	ID         int64     `json:"id" db:"id"`
	Code       string    `json:"code" db:"code"`
	ImageRef   string    `json:"image_ref" db:"image_ref"`
	CreatedBy  int64     `json:"created_by" db:"created_by"`
	Password   string    `json:"-" db:"password"`
	MaxUsage   int       `json:"max_usage" db:"max_usage"`
	UsageCount int       `json:"usage_count" db:"usage_count"`
	ExpiresAt  time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Captcha  string `json:"captcha,omitempty"`
	ClientIP string `json:"client_ip,omitempty"`
}

// LoginResponse represents a login response.
type LoginResponse struct {
	User               *User    `json:"user"`
	Token              string   `json:"token"`
	Session            *Session `json:"session"`
	MustChangePassword bool     `json:"must_change_password"`
	LockWarning        bool     `json:"lock_warning"`
}

// TokenCreateRequest represents a token creation request.
type TokenCreateRequest struct {
	Name      string   `json:"name" binding:"required"`
	Scopes    []string `json:"scopes" binding:"required"`
	ExpiresIn string   `json:"expires_in"` // e.g., "30d", "1y"
}
