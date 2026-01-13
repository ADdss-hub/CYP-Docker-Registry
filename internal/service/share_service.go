// Package service provides business logic services for the container registry.
package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"container-registry/internal/dao"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// ShareService provides share link management services.
type ShareService struct {
	logger *zap.Logger
}

// ShareLink represents a share link.
type ShareLink struct {
	ID              int64     `json:"id"`
	Code            string    `json:"code"`
	ImageRef        string    `json:"image_ref"`
	CreatedBy       int64     `json:"created_by"`
	CreatedByName   string    `json:"created_by_name,omitempty"`
	RequirePassword bool      `json:"require_password"`
	MaxUsage        int       `json:"max_usage"`
	UsageCount      int       `json:"usage_count"`
	ExpiresAt       time.Time `json:"expires_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreateShareRequest represents a request to create a share link.
type CreateShareRequest struct {
	ImageRef  string `json:"image_ref" binding:"required"`
	Password  string `json:"password,omitempty"`
	MaxUsage  int    `json:"max_usage,omitempty"`
	ExpiresIn string `json:"expires_in,omitempty"` // e.g., "24h", "7d"
}

// NewShareService creates a new ShareService instance.
func NewShareService(logger *zap.Logger) *ShareService {
	return &ShareService{
		logger: logger,
	}
}

// CreateShareLink creates a new share link.
func (s *ShareService) CreateShareLink(req *CreateShareRequest, userID int64) (*ShareLink, string, error) {
	// Generate unique code
	code := generateShareCode()

	// Hash password if provided
	var passwordHash string
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, "", err
		}
		passwordHash = string(hash)
	}

	// Parse expiration
	var expiresAt time.Time
	if req.ExpiresIn != "" {
		duration, err := time.ParseDuration(req.ExpiresIn)
		if err != nil {
			// Try parsing as days
			if req.ExpiresIn[len(req.ExpiresIn)-1] == 'd' {
				days := req.ExpiresIn[:len(req.ExpiresIn)-1]
				var d int
				if _, err := time.ParseDuration(days + "h"); err == nil {
					d = 24
				}
				duration = time.Duration(d) * 24 * time.Hour
			} else {
				return nil, "", errors.New("invalid expires_in format")
			}
		}
		expiresAt = time.Now().Add(duration)
	} else {
		// Default: 24 hours
		expiresAt = time.Now().Add(24 * time.Hour)
	}

	daoLink := &dao.ShareLink{
		Code:     code,
		ImageRef: req.ImageRef,
		CreatedBy: userID,
		MaxUsage: req.MaxUsage,
	}

	if passwordHash != "" {
		daoLink.PasswordHash.String = passwordHash
		daoLink.PasswordHash.Valid = true
	}

	if !expiresAt.IsZero() {
		daoLink.ExpiresAt.Time = expiresAt
		daoLink.ExpiresAt.Valid = true
	}

	if err := dao.CreateShareLink(daoLink); err != nil {
		return nil, "", err
	}

	link := &ShareLink{
		ID:              daoLink.ID,
		Code:            daoLink.Code,
		ImageRef:        daoLink.ImageRef,
		CreatedBy:       daoLink.CreatedBy,
		RequirePassword: passwordHash != "",
		MaxUsage:        daoLink.MaxUsage,
		UsageCount:      0,
		ExpiresAt:       expiresAt,
		CreatedAt:       daoLink.CreatedAt,
	}

	return link, code, nil
}

// GetShareLink retrieves a share link by code.
func (s *ShareService) GetShareLink(code string) (*ShareLink, error) {
	daoLink, err := dao.GetShareLink(code)
	if err != nil {
		return nil, err
	}
	if daoLink == nil {
		return nil, errors.New("share link not found")
	}

	// Check expiration
	if daoLink.ExpiresAt.Valid && time.Now().After(daoLink.ExpiresAt.Time) {
		return nil, errors.New("share link expired")
	}

	// Check usage limit
	if daoLink.MaxUsage > 0 && daoLink.UsageCount >= daoLink.MaxUsage {
		return nil, errors.New("share link usage limit exceeded")
	}

	return s.convertLink(daoLink), nil
}

// VerifySharePassword verifies the password for a share link.
func (s *ShareService) VerifySharePassword(code, password string) error {
	daoLink, err := dao.GetShareLink(code)
	if err != nil {
		return err
	}
	if daoLink == nil {
		return errors.New("share link not found")
	}

	if !daoLink.PasswordHash.Valid || daoLink.PasswordHash.String == "" {
		return nil // No password required
	}

	if err := bcrypt.CompareHashAndPassword([]byte(daoLink.PasswordHash.String), []byte(password)); err != nil {
		return errors.New("invalid password")
	}

	return nil
}

// IncrementUsage increments the usage count of a share link.
func (s *ShareService) IncrementUsage(code string) error {
	return dao.IncrementShareLinkUsage(code)
}

// ListShareLinks lists share links created by a user.
func (s *ShareService) ListShareLinks(userID int64, page, pageSize int) ([]*ShareLink, int, error) {
	daoLinks, total, err := dao.ListShareLinks(userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	links := make([]*ShareLink, len(daoLinks))
	for i, daoLink := range daoLinks {
		links[i] = s.convertLink(daoLink)
	}

	return links, total, nil
}

// DeleteShareLink deletes a share link.
func (s *ShareService) DeleteShareLink(id int64, userID int64) error {
	// TODO: Verify ownership
	return dao.DeleteShareLink(id)
}

// RevokeShareLink revokes a share link (same as delete).
func (s *ShareService) RevokeShareLink(code string, userID int64) error {
	daoLink, err := dao.GetShareLink(code)
	if err != nil {
		return err
	}
	if daoLink == nil {
		return errors.New("share link not found")
	}

	if daoLink.CreatedBy != userID {
		return errors.New("permission denied")
	}

	return dao.DeleteShareLink(daoLink.ID)
}

func (s *ShareService) convertLink(daoLink *dao.ShareLink) *ShareLink {
	link := &ShareLink{
		ID:              daoLink.ID,
		Code:            daoLink.Code,
		ImageRef:        daoLink.ImageRef,
		CreatedBy:       daoLink.CreatedBy,
		RequirePassword: daoLink.PasswordHash.Valid && daoLink.PasswordHash.String != "",
		MaxUsage:        daoLink.MaxUsage,
		UsageCount:      daoLink.UsageCount,
		CreatedAt:       daoLink.CreatedAt,
	}

	if daoLink.ExpiresAt.Valid {
		link.ExpiresAt = daoLink.ExpiresAt.Time
	}

	return link
}

func generateShareCode() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
