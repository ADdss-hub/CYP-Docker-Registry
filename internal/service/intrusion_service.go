// Package service provides business logic services for CYP-Registry.
package service

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// IntrusionService provides intrusion detection services.
type IntrusionService struct {
	config       *IntrusionConfig
	attemptStore sync.Map // map[ip]*AttemptInfo
	lockService  *LockService
	logger       *zap.Logger
}

// IntrusionConfig holds intrusion detection configuration.
type IntrusionConfig struct {
	Enabled            bool
	MaxLoginAttempts   int
	MaxTokenAttempts   int
	MaxAPIAttempts     int
	LockDuration       time.Duration
	ProgressiveDelay   bool
	Rules              []IntrusionRule
	RealTimeMonitoring bool
	LogAllAccess       bool
	NotifyOnLock       bool
	NotifyChannels     []string
}

// IntrusionRule represents an intrusion detection rule.
type IntrusionRule struct {
	Name             string   `json:"name" yaml:"name"`
	Description      string   `json:"description" yaml:"description"`
	Action           string   `json:"action" yaml:"action"` // lock, warn, ban
	Threshold        int      `json:"threshold" yaml:"threshold"`
	AllowedHours     string   `json:"allowed_hours,omitempty" yaml:"allowed_hours"`
	AllowedCountries []string `json:"allowed_countries,omitempty" yaml:"allowed_countries"`
}

// AttemptInfo holds information about access attempts.
type AttemptInfo struct {
	Count       int
	LastAttempt time.Time
	Codes       map[string]int
	Delays      []time.Duration
}

// NewIntrusionService creates a new IntrusionService instance.
func NewIntrusionService(config *IntrusionConfig, lockService *LockService, logger *zap.Logger) *IntrusionService {
	if config == nil {
		config = &IntrusionConfig{
			Enabled:          true,
			MaxLoginAttempts: 3,
			MaxTokenAttempts: 5,
			MaxAPIAttempts:   10,
			LockDuration:     time.Hour,
			ProgressiveDelay: true,
		}
	}

	return &IntrusionService{
		config:      config,
		lockService: lockService,
		logger:      logger,
	}
}

// IncrementFailedAttempt increments the failed attempt count for an IP.
func (s *IntrusionService) IncrementFailedAttempt(ip, code string) {
	info, _ := s.attemptStore.LoadOrStore(ip, &AttemptInfo{
		Codes: make(map[string]int),
	})

	attempt := info.(*AttemptInfo)
	attempt.Count++
	attempt.LastAttempt = time.Now()
	attempt.Codes[code]++

	// Check if should lock
	if s.shouldLock(ip, code) {
		if s.lockService != nil {
			s.lockService.LockSystem("too_many_failed_attempts", ip)
		}
		s.logIntrusion(ip, code, "system_locked")
	}
}

// ShouldLock determines if the system should be locked based on attempts.
func (s *IntrusionService) ShouldLock(ip string) bool {
	info, ok := s.attemptStore.Load(ip)
	if !ok {
		return false
	}

	attempt := info.(*AttemptInfo)
	return attempt.Count >= s.config.MaxAPIAttempts
}

// shouldLock checks if the system should be locked based on specific code.
func (s *IntrusionService) shouldLock(ip, code string) bool {
	info, ok := s.attemptStore.Load(ip)
	if !ok {
		return false
	}

	attempt := info.(*AttemptInfo)

	// Check specific rules
	switch code {
	case "direct_url_access", "forged_jwt":
		return attempt.Codes[code] >= 1

	case "invalid_jwt", "invalid_token":
		return attempt.Count >= s.config.MaxTokenAttempts

	case "unauthorized_access":
		return attempt.Count >= s.config.MaxAPIAttempts

	case "login_failure":
		return attempt.Count >= s.config.MaxLoginAttempts

	default:
		return attempt.Count >= 10
	}
}

// GetProgressiveDelay returns the progressive delay for an IP.
func (s *IntrusionService) GetProgressiveDelay(ip string) time.Duration {
	if !s.config.ProgressiveDelay {
		return 0
	}

	info, ok := s.attemptStore.Load(ip)
	if !ok {
		return 0
	}

	attempt := info.(*AttemptInfo)
	// Progressive delay: 1s, 2s, 4s, 8s, 16s, max 30s
	delay := time.Second * time.Duration(1<<uint(attempt.Count-1))
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}

	return delay
}

// ResetAttempts resets the attempt count for an IP.
func (s *IntrusionService) ResetAttempts(ip string) {
	s.attemptStore.Delete(ip)
}

// GetAttemptInfo returns attempt information for an IP.
func (s *IntrusionService) GetAttemptInfo(ip string) *AttemptInfo {
	info, ok := s.attemptStore.Load(ip)
	if !ok {
		return nil
	}
	return info.(*AttemptInfo)
}

// GetRemainingAttempts returns the remaining attempts for an IP.
func (s *IntrusionService) GetRemainingAttempts(ip, code string) int {
	info, ok := s.attemptStore.Load(ip)
	if !ok {
		return s.config.MaxLoginAttempts
	}

	attempt := info.(*AttemptInfo)
	var max int

	switch code {
	case "login_failure":
		max = s.config.MaxLoginAttempts
	case "invalid_jwt", "invalid_token":
		max = s.config.MaxTokenAttempts
	default:
		max = s.config.MaxAPIAttempts
	}

	remaining := max - attempt.Count
	if remaining < 0 {
		remaining = 0
	}

	return remaining
}

// CheckRule checks if a specific intrusion rule is triggered.
func (s *IntrusionService) CheckRule(ruleName, ip string) bool {
	for _, rule := range s.config.Rules {
		if rule.Name == ruleName {
			info, ok := s.attemptStore.Load(ip)
			if !ok {
				return false
			}

			attempt := info.(*AttemptInfo)
			if attempt.Codes[ruleName] >= rule.Threshold {
				if rule.Action == "lock" && s.lockService != nil {
					s.lockService.LockSystem(rule.Description, ip)
				}
				return true
			}
		}
	}
	return false
}

// logIntrusion logs an intrusion event.
func (s *IntrusionService) logIntrusion(ip, code, action string) {
	if s.logger != nil {
		s.logger.Error("Intrusion detected",
			zap.String("ip", ip),
			zap.String("code", code),
			zap.String("action", action),
			zap.Time("timestamp", time.Now()),
		)
	}
}

// CleanupOldAttempts removes old attempt records.
func (s *IntrusionService) CleanupOldAttempts(maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)

	s.attemptStore.Range(func(key, value interface{}) bool {
		attempt := value.(*AttemptInfo)
		if attempt.LastAttempt.Before(cutoff) {
			s.attemptStore.Delete(key)
		}
		return true
	})
}
