// Package service provides business logic services for CYP-Docker-Registry.
package service

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SecurityService 提供安全保护服务
// 问题8：对系统中的密码进行安全保护，如果强制查询立即删除所有数据库信息
type SecurityService struct {
	logger              *zap.Logger
	mu                  sync.RWMutex
	forceQueryAttempts  int
	lastForceQueryTime  time.Time
	maxForceQueryBefore int
	dataPath            string
	lockService         *LockService
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	MaxForceQueryAttempts int    // 最大强制查询尝试次数
	DataPath              string // 数据目录路径
}

// NewSecurityService 创建安全服务实例
func NewSecurityService(config *SecurityConfig, lockService *LockService, logger *zap.Logger) *SecurityService {
	if config == nil {
		config = &SecurityConfig{
			MaxForceQueryAttempts: 3,
			DataPath:              "./data",
		}
	}

	return &SecurityService{
		logger:              logger,
		maxForceQueryBefore: config.MaxForceQueryAttempts,
		dataPath:            config.DataPath,
		lockService:         lockService,
	}
}

// DetectForceQuery 检测强制查询密码的行为
// 如果检测到强制查询，立即删除所有数据库信息并锁定系统
func (s *SecurityService) DetectForceQuery(queryType string, ip string, userAgent string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 记录可疑查询
	s.forceQueryAttempts++
	s.lastForceQueryTime = time.Now()

	s.logger.Warn("检测到可疑的密码查询行为",
		zap.String("query_type", queryType),
		zap.String("ip", ip),
		zap.String("user_agent", userAgent),
		zap.Int("attempts", s.forceQueryAttempts),
	)

	// 如果超过最大尝试次数，执行安全措施
	if s.forceQueryAttempts >= s.maxForceQueryBefore {
		s.logger.Error("强制查询次数超限，执行安全保护措施",
			zap.Int("attempts", s.forceQueryAttempts),
			zap.String("ip", ip),
		)

		// 执行数据清除
		s.executeSecurityProtection(ip)
		return true
	}

	return false
}

// executeSecurityProtection 执行安全保护措施
// 删除所有敏感数据并锁定系统
func (s *SecurityService) executeSecurityProtection(triggerIP string) {
	s.logger.Error("开始执行安全保护措施 - 删除所有数据库信息")

	// 1. 首先锁定系统
	if s.lockService != nil {
		s.lockService.LockSystemByBypass(triggerIP, "security_protection")
		s.lockService.SetRequireManual(true) // 必须手动解锁
	}

	// 2. 删除数据库文件
	dbPaths := []string{
		filepath.Join(s.dataPath, "registry.db"),
		filepath.Join(s.dataPath, "users.db"),
		filepath.Join(s.dataPath, "tokens.db"),
		filepath.Join(s.dataPath, "audit.db"),
		filepath.Join(s.dataPath, "meta"),
	}

	for _, dbPath := range dbPaths {
		if err := s.secureDelete(dbPath); err != nil {
			s.logger.Error("删除数据文件失败",
				zap.String("path", dbPath),
				zap.Error(err),
			)
		} else {
			s.logger.Info("已删除数据文件",
				zap.String("path", dbPath),
			)
		}
	}

	// 3. 创建安全标记文件
	s.createSecurityMarker(triggerIP)

	s.logger.Error("安全保护措施执行完成 - 系统已锁定，数据已清除")
}

// secureDelete 安全删除文件或目录
func (s *SecurityService) secureDelete(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if info.IsDir() {
		return os.RemoveAll(path)
	}

	// 对于文件，先覆写再删除
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err == nil {
		// 用零字节覆写文件
		zeros := make([]byte, 4096)
		fileSize := info.Size()
		for written := int64(0); written < fileSize; {
			n, _ := file.Write(zeros)
			written += int64(n)
		}
		file.Sync()
		file.Close()
	}

	return os.Remove(path)
}

// createSecurityMarker 创建安全标记文件
func (s *SecurityService) createSecurityMarker(triggerIP string) {
	markerPath := filepath.Join(s.dataPath, ".security_triggered")
	content := []byte("Security protection triggered at " + time.Now().Format(time.RFC3339) + " from IP: " + triggerIP + "\nSystem requires reinstallation.")
	os.WriteFile(markerPath, content, 0600)
}

// IsSecurityTriggered 检查安全保护是否已触发
func (s *SecurityService) IsSecurityTriggered() bool {
	markerPath := filepath.Join(s.dataPath, ".security_triggered")
	_, err := os.Stat(markerPath)
	return err == nil
}

// ResetForceQueryCounter 重置强制查询计数器（仅用于测试）
func (s *SecurityService) ResetForceQueryCounter() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.forceQueryAttempts = 0
}

// GetForceQueryAttempts 获取当前强制查询尝试次数
func (s *SecurityService) GetForceQueryAttempts() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.forceQueryAttempts
}

// ValidatePasswordQuery 验证密码查询请求是否合法
// 返回 true 表示查询合法，false 表示可疑
func (s *SecurityService) ValidatePasswordQuery(queryType string, userID int64, ip string) bool {
	// 检查是否是已知的合法查询类型
	validQueryTypes := map[string]bool{
		"login":           true,
		"change_password": true,
		"reset_password":  true,
	}

	if !validQueryTypes[queryType] {
		// 未知的查询类型，记录并检测
		s.DetectForceQuery(queryType, ip, "")
		return false
	}

	return true
}
