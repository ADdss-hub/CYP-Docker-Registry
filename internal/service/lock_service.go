// Package service provides business logic services for the container registry.
package service

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// LockService provides system lock management.
type LockService struct {
	mu           sync.RWMutex
	isLocked     bool
	lockReason   string
	lockType     string
	lockedAt     time.Time
	lockedByIP   string
	lockedByUser string
	unlockAt     time.Time
	requireManual bool
	logger       *zap.Logger
}

// LockConfig holds lock configuration.
type LockConfig struct {
	Enabled            bool
	LockOnBypassAttempt bool
	LockDuration       time.Duration
	RequireManual      bool
	HardwareLock       HardwareLockConfig
	NetworkLock        NetworkLockConfig
	ServiceLock        ServiceLockConfig
}

// HardwareLockConfig holds hardware lock configuration.
type HardwareLockConfig struct {
	LockCPUPercent    int
	LockMemoryPercent int
}

// NetworkLockConfig holds network lock configuration.
type NetworkLockConfig struct {
	BlockIncoming bool
	BlockOutgoing bool
	BlockDuration time.Duration
}

// ServiceLockConfig holds service lock configuration.
type ServiceLockConfig struct {
	PauseAllWorkflows   bool
	AllowReadOnlyMode   bool
	ShutdownGracePeriod time.Duration
}

// LockStatus represents the current lock status.
type LockStatus struct {
	IsLocked      bool      `json:"is_locked"`
	LockReason    string    `json:"lock_reason"`
	LockType      string    `json:"lock_type"`
	LockedAt      time.Time `json:"locked_at"`
	LockedByIP    string    `json:"locked_by_ip"`
	LockedByUser  string    `json:"locked_by_user,omitempty"`
	UnlockAt      time.Time `json:"unlock_at,omitempty"`
	RequireManual bool      `json:"require_manual"`
}

// NewLockService creates a new LockService instance.
func NewLockService(logger *zap.Logger) *LockService {
	return &LockService{
		logger:        logger,
		requireManual: true,
	}
}

// IsSystemLocked returns whether the system is locked.
func (s *LockService) IsSystemLocked() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isLocked {
		return false
	}

	// Check if auto-unlock time has passed
	if !s.requireManual && !s.unlockAt.IsZero() && time.Now().After(s.unlockAt) {
		// Auto-unlock (need to upgrade to write lock)
		s.mu.RUnlock()
		s.mu.Lock()
		s.isLocked = false
		s.mu.Unlock()
		s.mu.RLock()
		return false
	}

	return true
}

// GetLockReason returns the lock reason.
func (s *LockService) GetLockReason() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lockReason
}

// GetLockStatus returns the full lock status.
func (s *LockService) GetLockStatus() *LockStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &LockStatus{
		IsLocked:      s.isLocked,
		LockReason:    s.lockReason,
		LockType:      s.lockType,
		LockedAt:      s.lockedAt,
		LockedByIP:    s.lockedByIP,
		LockedByUser:  s.lockedByUser,
		UnlockAt:      s.unlockAt,
		RequireManual: s.requireManual,
	}
}

// LockSystem locks the system.
func (s *LockService) LockSystem(reason, ip string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.isLocked = true
	s.lockReason = reason
	s.lockType = "rule_triggered"
	s.lockedAt = time.Now()
	s.lockedByIP = ip

	if s.logger != nil {
		s.logger.Error("System locked",
			zap.String("reason", reason),
			zap.String("ip", ip),
			zap.Time("locked_at", s.lockedAt),
		)
	}

	return nil
}

// LockSystemByBypass locks the system due to bypass attempt.
func (s *LockService) LockSystemByBypass(ip, user string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.isLocked = true
	s.lockReason = "Bypass attempt detected"
	s.lockType = "bypass_attempt"
	s.lockedAt = time.Now()
	s.lockedByIP = ip
	s.lockedByUser = user
	s.requireManual = true

	if s.logger != nil {
		s.logger.Error("System locked due to bypass attempt",
			zap.String("ip", ip),
			zap.String("user", user),
			zap.Time("locked_at", s.lockedAt),
		)
	}

	return nil
}

// UnlockSystem unlocks the system.
func (s *LockService) UnlockSystem(adminPassword string) error {
	// TODO: Validate admin password
	s.mu.Lock()
	defer s.mu.Unlock()

	s.isLocked = false
	s.lockReason = ""
	s.lockType = ""
	s.lockedByIP = ""
	s.lockedByUser = ""

	if s.logger != nil {
		s.logger.Info("System unlocked")
	}

	return nil
}

// SetAutoUnlock sets the auto-unlock time.
func (s *LockService) SetAutoUnlock(duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.unlockAt = time.Now().Add(duration)
	s.requireManual = false
}

// SetRequireManual sets whether manual unlock is required.
func (s *LockService) SetRequireManual(required bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requireManual = required
}
