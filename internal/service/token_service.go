// Package service provides business logic services for CYP-Registry.
package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"cyp-registry/internal/dao"

	"go.uber.org/zap"
)

// TokenService provides personal access token management services.
type TokenService struct {
	logger *zap.Logger
}

// Token represents a personal access token.
type Token struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	Name       string    `json:"name"`
	Scopes     []string  `json:"scopes"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
	LastUsedAt time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateTokenRequest represents a request to create a token.
type CreateTokenRequest struct {
	Name      string   `json:"name" binding:"required"`
	Scopes    []string `json:"scopes" binding:"required"`
	ExpiresIn string   `json:"expires_in,omitempty"` // e.g., "30d", "1y"
}

// CreateTokenResponse represents the response when creating a token.
type CreateTokenResponse struct {
	Token      *Token `json:"token"`
	PlainToken string `json:"plain_token"` // Only returned once
}

// NewTokenService creates a new TokenService instance.
func NewTokenService(logger *zap.Logger) *TokenService {
	return &TokenService{
		logger: logger,
	}
}

// CreateToken creates a new personal access token.
func (s *TokenService) CreateToken(req *CreateTokenRequest, userID int64) (*CreateTokenResponse, error) {
	// Generate token
	plainToken := generatePlainToken()
	tokenHash := hashToken(plainToken)

	// Parse expiration
	var expiresAt time.Time
	if req.ExpiresIn != "" {
		duration, err := parseDuration(req.ExpiresIn)
		if err != nil {
			return nil, err
		}
		expiresAt = time.Now().Add(duration)
	}

	daoToken := &dao.PersonalAccessToken{
		UserID:    userID,
		Name:      req.Name,
		TokenHash: tokenHash,
		Scopes:    req.Scopes,
	}

	if !expiresAt.IsZero() {
		daoToken.ExpiresAt.Time = expiresAt
		daoToken.ExpiresAt.Valid = true
	}

	if err := dao.CreateToken(daoToken); err != nil {
		return nil, err
	}

	token := &Token{
		ID:        daoToken.ID,
		UserID:    daoToken.UserID,
		Name:      daoToken.Name,
		Scopes:    daoToken.Scopes,
		ExpiresAt: expiresAt,
		CreatedAt: daoToken.CreatedAt,
	}

	return &CreateTokenResponse{
		Token:      token,
		PlainToken: "pat_" + plainToken,
	}, nil
}

// ValidateToken validates a personal access token.
func (s *TokenService) ValidateToken(plainToken string) (*Token, error) {
	// Remove prefix if present
	if len(plainToken) > 4 && plainToken[:4] == "pat_" {
		plainToken = plainToken[4:]
	}

	tokenHash := hashToken(plainToken)
	daoToken, err := dao.GetTokenByHash(tokenHash)
	if err != nil {
		return nil, err
	}
	if daoToken == nil {
		return nil, errors.New("invalid token")
	}

	// Check expiration
	if daoToken.ExpiresAt.Valid && time.Now().After(daoToken.ExpiresAt.Time) {
		return nil, errors.New("token expired")
	}

	// Update last used
	dao.UpdateTokenLastUsed(daoToken.ID)

	token := &Token{
		ID:        daoToken.ID,
		UserID:    daoToken.UserID,
		Name:      daoToken.Name,
		Scopes:    daoToken.Scopes,
		CreatedAt: daoToken.CreatedAt,
	}

	if daoToken.ExpiresAt.Valid {
		token.ExpiresAt = daoToken.ExpiresAt.Time
	}
	if daoToken.LastUsedAt.Valid {
		token.LastUsedAt = daoToken.LastUsedAt.Time
	}

	return token, nil
}

// ListTokens lists all tokens for a user.
func (s *TokenService) ListTokens(userID int64) ([]*Token, error) {
	daoTokens, err := dao.ListUserTokens(userID)
	if err != nil {
		return nil, err
	}

	tokens := make([]*Token, len(daoTokens))
	for i, daoToken := range daoTokens {
		token := &Token{
			ID:        daoToken.ID,
			UserID:    daoToken.UserID,
			Name:      daoToken.Name,
			Scopes:    daoToken.Scopes,
			CreatedAt: daoToken.CreatedAt,
		}
		if daoToken.ExpiresAt.Valid {
			token.ExpiresAt = daoToken.ExpiresAt.Time
		}
		if daoToken.LastUsedAt.Valid {
			token.LastUsedAt = daoToken.LastUsedAt.Time
		}
		tokens[i] = token
	}

	return tokens, nil
}

// DeleteToken deletes a token.
func (s *TokenService) DeleteToken(id int64, userID int64) error {
	// TODO: Verify ownership
	return dao.DeleteToken(id)
}

// HasScope checks if a token has a specific scope.
func (s *TokenService) HasScope(token *Token, scope string) bool {
	for _, s := range token.Scopes {
		if s == scope || s == "*" {
			return true
		}
	}
	return false
}

func generatePlainToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func parseDuration(s string) (time.Duration, error) {
	// Try standard duration first
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Try custom formats like "30d", "1y"
	if len(s) < 2 {
		return 0, errors.New("invalid duration format")
	}

	unit := s[len(s)-1]
	value := s[:len(s)-1]

	var multiplier time.Duration
	switch unit {
	case 'd':
		multiplier = 24 * time.Hour
	case 'w':
		multiplier = 7 * 24 * time.Hour
	case 'y':
		multiplier = 365 * 24 * time.Hour
	default:
		return 0, errors.New("invalid duration unit")
	}

	var num int
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0, errors.New("invalid duration value")
		}
		num = num*10 + int(c-'0')
	}

	return time.Duration(num) * multiplier, nil
}
