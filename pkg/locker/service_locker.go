// Package locker provides system locking mechanisms for security enforcement.
package locker

import (
	"sync"
)

// WorkflowController interface for workflow service control.
type WorkflowController interface {
	PauseAll() error
	ResumeAll() error
	GetRunningCount() int
}

// ConfigController interface for configuration control.
type ConfigController interface {
	SetReadOnlyMode(enabled bool)
	IsReadOnlyMode() bool
}

// ServiceLocker implements service-level locking for security lockdown.
type ServiceLocker struct {
	workflowController WorkflowController
	configController   ConfigController
	isPaused           bool
	isReadOnly         bool
	gracePeriod        int // seconds
	mu                 sync.Mutex
}

// ServiceLockerConfig holds configuration for service locking.
type ServiceLockerConfig struct {
	GracePeriod int // seconds before full lock
}

// NewServiceLocker creates a new ServiceLocker instance.
func NewServiceLocker(wc WorkflowController, cc ConfigController, config *ServiceLockerConfig) *ServiceLocker {
	if config == nil {
		config = &ServiceLockerConfig{
			GracePeriod: 30,
		}
	}

	return &ServiceLocker{
		workflowController: wc,
		configController:   cc,
		gracePeriod:        config.GracePeriod,
	}
}

// Lock pauses all services and enables read-only mode.
func (l *ServiceLocker) Lock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.isPaused {
		return nil
	}

	// Pause all workflows
	if l.workflowController != nil {
		if err := l.workflowController.PauseAll(); err != nil {
			return err
		}
	}

	// Enable read-only mode
	if l.configController != nil {
		l.configController.SetReadOnlyMode(true)
	}

	l.isPaused = true
	l.isReadOnly = true

	return nil
}

// Unlock resumes all services and disables read-only mode.
func (l *ServiceLocker) Unlock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.isPaused {
		return nil
	}

	// Resume all workflows
	if l.workflowController != nil {
		if err := l.workflowController.ResumeAll(); err != nil {
			return err
		}
	}

	// Disable read-only mode
	if l.configController != nil {
		l.configController.SetReadOnlyMode(false)
	}

	l.isPaused = false
	l.isReadOnly = false

	return nil
}

// IsLocked returns the current lock status.
func (l *ServiceLocker) IsLocked() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.isPaused
}

// IsReadOnly returns the current read-only status.
func (l *ServiceLocker) IsReadOnly() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.isReadOnly
}

// SetReadOnlyMode enables or disables read-only mode without full lock.
func (l *ServiceLocker) SetReadOnlyMode(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.configController != nil {
		l.configController.SetReadOnlyMode(enabled)
	}
	l.isReadOnly = enabled
}

// GetStatus returns the current service locker status.
func (l *ServiceLocker) GetStatus() *ServiceLockerStatus {
	l.mu.Lock()
	defer l.mu.Unlock()

	status := &ServiceLockerStatus{
		IsPaused:   l.isPaused,
		IsReadOnly: l.isReadOnly,
	}

	if l.workflowController != nil {
		status.RunningWorkflows = l.workflowController.GetRunningCount()
	}

	return status
}

// ServiceLockerStatus represents the current status of the service locker.
type ServiceLockerStatus struct {
	IsPaused         bool `json:"is_paused"`
	IsReadOnly       bool `json:"is_read_only"`
	RunningWorkflows int  `json:"running_workflows"`
}

// LockManager coordinates all lockers for comprehensive system lockdown.
type LockManager struct {
	hardwareLocker *HardwareLocker
	networkLocker  *NetworkLocker
	serviceLocker  *ServiceLocker
	isLocked       bool
	lockReason     string
	lockIP         string
	mu             sync.Mutex
}

// LockManagerConfig holds configuration for the lock manager.
type LockManagerConfig struct {
	HardwareConfig *HardwareLockerConfig
	NetworkConfig  *NetworkLockerConfig
	ServiceConfig  *ServiceLockerConfig
}

// NewLockManager creates a new LockManager instance.
func NewLockManager(config *LockManagerConfig, wc WorkflowController, cc ConfigController) *LockManager {
	if config == nil {
		config = &LockManagerConfig{}
	}

	return &LockManager{
		hardwareLocker: NewHardwareLocker(config.HardwareConfig),
		networkLocker:  NewNetworkLocker(config.NetworkConfig),
		serviceLocker:  NewServiceLocker(wc, cc, config.ServiceConfig),
	}
}

// LockAll locks all subsystems.
func (m *LockManager) LockAll(reason, ip string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isLocked {
		return nil
	}

	// Lock services first (graceful)
	if err := m.serviceLocker.Lock(); err != nil {
		return err
	}

	// Lock network
	if err := m.networkLocker.Lock(); err != nil {
		m.serviceLocker.Unlock()
		return err
	}

	// Lock hardware last
	if err := m.hardwareLocker.Lock(); err != nil {
		m.networkLocker.Unlock()
		m.serviceLocker.Unlock()
		return err
	}

	m.isLocked = true
	m.lockReason = reason
	m.lockIP = ip

	return nil
}

// UnlockAll unlocks all subsystems.
func (m *LockManager) UnlockAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isLocked {
		return nil
	}

	// Unlock in reverse order
	m.hardwareLocker.Unlock()
	m.networkLocker.Unlock()
	m.serviceLocker.Unlock()

	m.isLocked = false
	m.lockReason = ""
	m.lockIP = ""

	return nil
}

// IsLocked returns the current lock status.
func (m *LockManager) IsLocked() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isLocked
}

// GetLockInfo returns information about the current lock.
func (m *LockManager) GetLockInfo() *LockInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	return &LockInfo{
		IsLocked:   m.isLocked,
		Reason:     m.lockReason,
		TriggerIP:  m.lockIP,
		Hardware:   m.hardwareLocker.IsLocked(),
		Network:    m.networkLocker.IsLocked(),
		Service:    m.serviceLocker.IsLocked(),
		ReadOnly:   m.serviceLocker.IsReadOnly(),
	}
}

// LockInfo represents information about the current lock state.
type LockInfo struct {
	IsLocked  bool   `json:"is_locked"`
	Reason    string `json:"reason"`
	TriggerIP string `json:"trigger_ip"`
	Hardware  bool   `json:"hardware_locked"`
	Network   bool   `json:"network_locked"`
	Service   bool   `json:"service_locked"`
	ReadOnly  bool   `json:"read_only"`
}
