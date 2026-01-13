// Package service provides business logic services for CYP-Registry.
package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"cyp-registry/internal/dao"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService provides authentication services.
type AuthService struct {
	jwtSecret     []byte
	sessions      sync.Map // map[int64]*Session
	tokenExpiry   time.Duration
	sessionExpiry time.Duration
}

// User represents a user in the system.
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Session represents a user session.
type Session struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PersonalAccessToken represents a personal access token.
type PersonalAccessToken struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	Name       string    `json:"name"`
	TokenHash  string    `json:"-"`
	Scopes     []string  `json:"scopes"`
	ExpiresAt  time.Time `json:"expires_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// JWTClaims represents JWT claims.
type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	ClientIP string `json:"client_ip"`
}

// LoginResponse represents a login response.
type LoginResponse struct {
	User               *User    `json:"user"`
	Token              string   `json:"token"`
	Session            *Session `json:"session"`
	MustChangePassword bool     `json:"must_change_password"`
	LockWarning        bool     `json:"lock_warning"`
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(jwtSecret string) *AuthService {
	return &AuthService{
		jwtSecret:     []byte(jwtSecret),
		tokenExpiry:   24 * time.Hour,
		sessionExpiry: 24 * time.Hour,
	}
}

// Login authenticates a user and returns a JWT token.
func (s *AuthService) Login(req *LoginRequest) (*LoginResponse, error) {
	// Look up user from database
	daoUser, err := dao.GetUserByUsername(req.Username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	if daoUser == nil {
		return nil, errors.New("invalid credentials")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(daoUser.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Check if user is active
	if !daoUser.IsActive {
		return nil, errors.New("user is inactive")
	}

	user := &User{
		ID:       daoUser.ID,
		Username: daoUser.Username,
		Email:    daoUser.Email.String,
		Role:     daoUser.Role,
		IsActive: daoUser.IsActive,
	}

	// Generate JWT token
	token, err := s.generateJWT(user)
	if err != nil {
		return nil, err
	}

	// Create session
	session := s.createSession(user.ID, req.ClientIP, "")

	// Update last login time
	dao.UpdateUserLastLogin(user.ID)

	// Check if password needs to be changed (default password)
	mustChangePassword := req.Password == "admin123"

	return &LoginResponse{
		User:               user,
		Token:              token,
		Session:            session,
		MustChangePassword: mustChangePassword,
		LockWarning:        false,
	}, nil
}

// ValidateJWT validates a JWT token and returns user info.
func (s *AuthService) ValidateJWT(tokenStr string) (*User, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Check expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token expired")
	}

	return &User{
		ID:       claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
		IsActive: true,
	}, nil
}

// ValidateToken validates a personal access token.
func (s *AuthService) ValidateToken(tokenStr string) (*User, *PersonalAccessToken, error) {
	// TODO: Implement token validation from database
	return nil, nil, errors.New("token validation not implemented")
}

// GetSession returns a user's session.
func (s *AuthService) GetSession(userID int64) *Session {
	if session, ok := s.sessions.Load(userID); ok {
		return session.(*Session)
	}
	return nil
}

// TerminateSession terminates a user's session.
func (s *AuthService) TerminateSession(userID int64) error {
	s.sessions.Delete(userID)
	return nil
}

// UpdateTokenLastUsed updates the last used time of a token.
func (s *AuthService) UpdateTokenLastUsed(tokenID int64) error {
	// TODO: Implement database update
	return nil
}

// generateJWT generates a JWT token for a user.
func (s *AuthService) generateJWT(user *User) (string, error) {
	claims := &JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "CYP-Registry",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// createSession creates a new session for a user.
func (s *AuthService) createSession(userID int64, ip, userAgent string) *Session {
	sessionID := generateSessionID()
	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		IP:        ip,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(s.sessionExpiry),
	}

	s.sessions.Store(userID, session)
	return session
}

// HashPassword hashes a password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword checks if a password matches a hash.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// generateSessionID generates a random session ID.
func generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// HashToken hashes a token for storage.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
