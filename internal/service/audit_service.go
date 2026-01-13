// Package service provides business logic services for CYP-Registry.
package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AuditService provides audit logging services.
type AuditService struct {
	config    *AuditConfig
	chainHash string
	mu        sync.Mutex
	logger    *zap.Logger
	logFile   *os.File
}

// AuditConfig holds audit configuration.
type AuditConfig struct {
	LogAllRequests   bool
	LogFailedAuth    bool
	LogLockEvents    bool
	BlockchainHash   bool
	ImmutableStorage bool
	Retention        time.Duration
	AlertOnTamper    bool
	LogFilePath      string
}

// AccessAttempt represents an access attempt for audit logging.
type AccessAttempt struct {
	ID             int64     `json:"id"`
	IPAddress      string    `json:"ip_address"`
	UserAgent      string    `json:"user_agent"`
	UserID         int64     `json:"user_id,omitempty"`
	Action         string    `json:"action"`
	Resource       string    `json:"resource"`
	Status         string    `json:"status"`
	ErrorMsg       string    `json:"error_msg,omitempty"`
	BlockchainHash string    `json:"blockchain_hash,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID             int64                  `json:"id"`
	Timestamp      time.Time              `json:"timestamp"`
	Level          string                 `json:"level"`
	Event          string                 `json:"event"`
	UserID         int64                  `json:"user_id,omitempty"`
	Username       string                 `json:"username,omitempty"`
	IPAddress      string                 `json:"ip_address"`
	Resource       string                 `json:"resource"`
	Action         string                 `json:"action"`
	Status         string                 `json:"status"`
	Details        map[string]interface{} `json:"details,omitempty"`
	BlockchainHash string                 `json:"blockchain_hash,omitempty"`
}

// NewAuditService creates a new AuditService instance.
func NewAuditService(config *AuditConfig, logger *zap.Logger) (*AuditService, error) {
	if config == nil {
		config = &AuditConfig{
			LogAllRequests: true,
			LogFailedAuth:  true,
			LogLockEvents:  true,
			BlockchainHash: true,
			Retention:      365 * 24 * time.Hour, // 1 year
		}
	}

	s := &AuditService{
		config: config,
		logger: logger,
	}

	// Open log file if path is specified
	if config.LogFilePath != "" {
		file, err := os.OpenFile(config.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		s.logFile = file
	}

	return s, nil
}

// LogAccessAttempt logs an access attempt.
func (s *AuditService) LogAccessAttempt(attempt *AccessAttempt) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate blockchain hash
	if s.config.BlockchainHash {
		attempt.BlockchainHash = s.calculateChainHash(attempt)
		s.chainHash = attempt.BlockchainHash
	}

	// Log to file
	if s.logFile != nil {
		data, _ := json.Marshal(attempt)
		s.logFile.WriteString(string(data) + "\n")
	}

	// Log to logger
	if s.logger != nil {
		s.logger.Info("Access attempt",
			zap.String("ip", attempt.IPAddress),
			zap.String("action", attempt.Action),
			zap.String("resource", attempt.Resource),
			zap.String("status", attempt.Status),
			zap.String("hash", attempt.BlockchainHash),
		)
	}

	return nil
}

// LogAuditEvent logs a general audit event.
func (s *AuditService) LogAuditEvent(log *AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Timestamp = time.Now()

	// Calculate blockchain hash
	if s.config.BlockchainHash {
		log.BlockchainHash = s.calculateAuditHash(log)
		s.chainHash = log.BlockchainHash
	}

	// Log to file
	if s.logFile != nil {
		data, _ := json.Marshal(log)
		s.logFile.WriteString(string(data) + "\n")
	}

	// Log to logger
	if s.logger != nil {
		s.logger.Info("Audit event",
			zap.String("event", log.Event),
			zap.String("action", log.Action),
			zap.String("status", log.Status),
			zap.String("hash", log.BlockchainHash),
		)
	}

	return nil
}

// LogLockEvent logs a system lock event.
func (s *AuditService) LogLockEvent(ip, reason, lockType string) error {
	if !s.config.LogLockEvents {
		return nil
	}

	return s.LogAuditEvent(&AuditLog{
		Level:     "critical",
		Event:     "system_locked",
		IPAddress: ip,
		Action:    "lock",
		Status:    "triggered",
		Details: map[string]interface{}{
			"reason":    reason,
			"lock_type": lockType,
		},
	})
}

// LogUnlockEvent logs a system unlock event.
func (s *AuditService) LogUnlockEvent(ip, username string) error {
	if !s.config.LogLockEvents {
		return nil
	}

	return s.LogAuditEvent(&AuditLog{
		Level:     "info",
		Event:     "system_unlocked",
		IPAddress: ip,
		Username:  username,
		Action:    "unlock",
		Status:    "success",
	})
}

// LogAuthFailure logs an authentication failure.
func (s *AuditService) LogAuthFailure(ip, username, reason string) error {
	if !s.config.LogFailedAuth {
		return nil
	}

	return s.LogAuditEvent(&AuditLog{
		Level:     "warn",
		Event:     "auth_failure",
		IPAddress: ip,
		Username:  username,
		Action:    "login",
		Status:    "failure",
		Details: map[string]interface{}{
			"reason": reason,
		},
	})
}

// calculateChainHash calculates the blockchain hash for an access attempt.
func (s *AuditService) calculateChainHash(attempt *AccessAttempt) string {
	data := fmt.Sprintf("%s|%s|%s|%d|%s",
		s.chainHash,
		attempt.IPAddress,
		attempt.Action,
		attempt.CreatedAt.Unix(),
		attempt.Resource,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// calculateAuditHash calculates the blockchain hash for an audit log.
func (s *AuditService) calculateAuditHash(log *AuditLog) string {
	data := fmt.Sprintf("%s|%s|%s|%d|%s",
		s.chainHash,
		log.IPAddress,
		log.Event,
		log.Timestamp.Unix(),
		log.Action,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// VerifyChain verifies the integrity of the audit log chain.
func (s *AuditService) VerifyChain(logs []*AuditLog) bool {
	var prevHash string

	for _, log := range logs {
		expectedHash := s.calculateAuditHash(log)
		if prevHash != "" && log.BlockchainHash != expectedHash {
			if s.config.AlertOnTamper && s.logger != nil {
				s.logger.Error("Audit log chain tampered",
					zap.Int64("log_id", log.ID),
					zap.String("expected", expectedHash),
					zap.String("actual", log.BlockchainHash),
				)
			}
			return false
		}
		prevHash = log.BlockchainHash
	}

	return true
}

// Close closes the audit service.
func (s *AuditService) Close() error {
	if s.logFile != nil {
		return s.logFile.Close()
	}
	return nil
}

// IncrementFailedAttempt is a helper method for middleware compatibility.
func (s *AuditService) IncrementFailedAttempt(ip, code string) {
	// This is handled by IntrusionService, but we log it here too
	s.LogAccessAttempt(&AccessAttempt{
		IPAddress: ip,
		Action:    "failed_attempt",
		Status:    "failure",
		ErrorMsg:  code,
		CreatedAt: time.Now(),
	})
}

// ShouldLock is a helper method for middleware compatibility.
func (s *AuditService) ShouldLock(ip string) bool {
	// This is handled by IntrusionService
	return false
}
