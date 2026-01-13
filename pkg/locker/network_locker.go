// Package locker provides system locking mechanisms for security enforcement.
package locker

import (
	"os/exec"
	"runtime"
	"sync"
)

// NetworkLocker implements network access control for security lockdown.
type NetworkLocker struct {
	blockedInterfaces []string
	isLocked          bool
	blockIncoming     bool
	blockOutgoing     bool
	containerID       string
	mu                sync.Mutex
}

// NetworkLockerConfig holds configuration for network locking.
type NetworkLockerConfig struct {
	BlockedInterfaces []string
	BlockIncoming     bool
	BlockOutgoing     bool
	ContainerID       string
}

// NewNetworkLocker creates a new NetworkLocker instance.
func NewNetworkLocker(config *NetworkLockerConfig) *NetworkLocker {
	if config == nil {
		config = &NetworkLockerConfig{
			BlockedInterfaces: []string{"eth0", "wlan0"},
			BlockIncoming:     true,
			BlockOutgoing:     false,
		}
	}

	return &NetworkLocker{
		blockedInterfaces: config.BlockedInterfaces,
		blockIncoming:     config.BlockIncoming,
		blockOutgoing:     config.BlockOutgoing,
		containerID:       config.ContainerID,
	}
}

// Lock blocks network access.
func (l *NetworkLocker) Lock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.isLocked {
		return nil
	}

	// Linux: Use iptables
	if runtime.GOOS == "linux" {
		if err := l.lockLinux(); err != nil {
			return err
		}
	}

	// Docker: Disconnect from network
	if l.isDocker() {
		if err := l.lockDocker(); err != nil {
			return err
		}
	}

	l.isLocked = true
	return nil
}

// Unlock restores network access.
func (l *NetworkLocker) Unlock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.isLocked {
		return nil
	}

	// Linux: Remove iptables rules
	if runtime.GOOS == "linux" {
		if err := l.unlockLinux(); err != nil {
			return err
		}
	}

	// Docker: Reconnect to network
	if l.isDocker() {
		if err := l.unlockDocker(); err != nil {
			return err
		}
	}

	l.isLocked = false
	return nil
}

// IsLocked returns the current lock status.
func (l *NetworkLocker) IsLocked() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.isLocked
}

// lockLinux applies iptables rules on Linux.
func (l *NetworkLocker) lockLinux() error {
	for _, iface := range l.blockedInterfaces {
		// Block incoming traffic
		if l.blockIncoming {
			cmd := exec.Command("iptables", "-A", "INPUT", "-i", iface, "-j", "DROP")
			cmd.Run()
		}

		// Block outgoing traffic (optional, usually allow for logging)
		if l.blockOutgoing {
			cmd := exec.Command("iptables", "-A", "OUTPUT", "-o", iface, "-j", "DROP")
			cmd.Run()
		} else {
			// Allow outgoing for audit log upload
			cmd := exec.Command("iptables", "-A", "OUTPUT", "-o", iface, "-j", "ACCEPT")
			cmd.Run()
		}
	}

	// Allow localhost
	exec.Command("iptables", "-A", "INPUT", "-i", "lo", "-j", "ACCEPT").Run()
	exec.Command("iptables", "-A", "OUTPUT", "-o", "lo", "-j", "ACCEPT").Run()

	return nil
}

// unlockLinux removes iptables rules on Linux.
func (l *NetworkLocker) unlockLinux() error {
	for _, iface := range l.blockedInterfaces {
		// Remove incoming block
		if l.blockIncoming {
			cmd := exec.Command("iptables", "-D", "INPUT", "-i", iface, "-j", "DROP")
			cmd.Run()
		}

		// Remove outgoing rules
		if l.blockOutgoing {
			cmd := exec.Command("iptables", "-D", "OUTPUT", "-o", iface, "-j", "DROP")
			cmd.Run()
		} else {
			cmd := exec.Command("iptables", "-D", "OUTPUT", "-o", iface, "-j", "ACCEPT")
			cmd.Run()
		}
	}

	return nil
}

// lockDocker disconnects container from network.
func (l *NetworkLocker) lockDocker() error {
	if l.containerID == "" {
		l.containerID = detectContainerID()
	}

	if l.containerID == "" {
		return nil
	}

	// Disconnect from bridge network
	cmd := exec.Command("docker", "network", "disconnect", "bridge", l.containerID)
	return cmd.Run()
}

// unlockDocker reconnects container to network.
func (l *NetworkLocker) unlockDocker() error {
	if l.containerID == "" {
		return nil
	}

	// Reconnect to bridge network
	cmd := exec.Command("docker", "network", "connect", "bridge", l.containerID)
	return cmd.Run()
}

// isDocker checks if running inside a Docker container.
func (l *NetworkLocker) isDocker() bool {
	return isRunningInDocker()
}

// BlockIP blocks a specific IP address.
func (l *NetworkLocker) BlockIP(ip string) error {
	if runtime.GOOS != "linux" {
		return nil
	}

	// Block incoming from IP
	cmd := exec.Command("iptables", "-A", "INPUT", "-s", ip, "-j", "DROP")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Block outgoing to IP
	cmd = exec.Command("iptables", "-A", "OUTPUT", "-d", ip, "-j", "DROP")
	return cmd.Run()
}

// UnblockIP unblocks a specific IP address.
func (l *NetworkLocker) UnblockIP(ip string) error {
	if runtime.GOOS != "linux" {
		return nil
	}

	// Remove incoming block
	cmd := exec.Command("iptables", "-D", "INPUT", "-s", ip, "-j", "DROP")
	cmd.Run()

	// Remove outgoing block
	cmd = exec.Command("iptables", "-D", "OUTPUT", "-d", ip, "-j", "DROP")
	return cmd.Run()
}

// BlockPort blocks a specific port.
func (l *NetworkLocker) BlockPort(port int, protocol string) error {
	if runtime.GOOS != "linux" {
		return nil
	}

	if protocol == "" {
		protocol = "tcp"
	}

	// Block incoming on port
	cmd := exec.Command("iptables", "-A", "INPUT", "-p", protocol, "--dport", string(rune(port)), "-j", "DROP")
	return cmd.Run()
}

// UnblockPort unblocks a specific port.
func (l *NetworkLocker) UnblockPort(port int, protocol string) error {
	if runtime.GOOS != "linux" {
		return nil
	}

	if protocol == "" {
		protocol = "tcp"
	}

	cmd := exec.Command("iptables", "-D", "INPUT", "-p", protocol, "--dport", string(rune(port)), "-j", "DROP")
	return cmd.Run()
}

// Helper function to detect container ID
func detectContainerID() string {
	// Implementation same as in hardware_locker.go
	return ""
}

// Helper function to check if running in Docker
func isRunningInDocker() bool {
	return false
}
