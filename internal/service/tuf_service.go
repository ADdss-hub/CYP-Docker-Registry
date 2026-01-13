// Package service 提供TUF服务
package service

import (
	"context"
	"sync"
	"time"

	"cyp-registry/pkg/signature"

	"go.uber.org/zap"
)

// TUFService TUF服务
type TUFService struct {
	manager   *signature.TUFManager
	config    *signature.TUFConfig
	logger    *zap.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	refreshMu sync.Mutex
}

// NewTUFService 创建TUF服务
func NewTUFService(config *signature.TUFConfig, logger *zap.Logger) (*TUFService, error) {
	if config == nil {
		config = signature.DefaultTUFConfig()
	}

	manager, err := signature.NewTUFManager(config, logger)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &TUFService{
		manager: manager,
		config:  config,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// Start 启动TUF服务
func (s *TUFService) Start() error {
	// 检查是否需要初始化
	if !s.manager.IsInitialized() {
		s.logger.Info("TUF仓库未初始化，正在初始化...")
		if err := s.manager.Initialize(); err != nil {
			return err
		}
	}

	// 启动自动刷新
	go s.autoRefreshLoop()

	s.logger.Info("TUF服务已启动")
	return nil
}

// Stop 停止TUF服务
func (s *TUFService) Stop() {
	s.cancel()
	s.logger.Info("TUF服务已停止")
}

// autoRefreshLoop 自动刷新循环
func (s *TUFService) autoRefreshLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.refreshMu.Lock()
			if err := s.manager.AutoRefresh(); err != nil {
				s.logger.Warn("自动刷新TUF失败", zap.Error(err))
			}
			s.refreshMu.Unlock()
		}
	}
}

// Initialize 初始化TUF仓库
func (s *TUFService) Initialize() error {
	return s.manager.Initialize()
}

// GetStatus 获取TUF状态
func (s *TUFService) GetStatus() *signature.TUFStatus {
	return s.manager.GetStatus()
}

// AddTarget 添加目标
func (s *TUFService) AddTarget(name string, data []byte, custom map[string]interface{}) error {
	return s.manager.AddTarget(name, data, custom)
}

// RemoveTarget 移除目标
func (s *TUFService) RemoveTarget(name string) error {
	return s.manager.RemoveTarget(name)
}

// GetTarget 获取目标
func (s *TUFService) GetTarget(name string) (*signature.TUFTarget, error) {
	return s.manager.GetTarget(name)
}

// ListTargets 列出所有目标
func (s *TUFService) ListTargets() map[string]*signature.TUFTarget {
	return s.manager.ListTargets()
}

// VerifyTarget 验证目标
func (s *TUFService) VerifyTarget(name string, data []byte) (bool, error) {
	return s.manager.VerifyTarget(name, data)
}

// RotateKey 轮换密钥
func (s *TUFService) RotateKey(role string) error {
	return s.manager.RotateKey(role)
}

// RefreshTimestamp 刷新Timestamp
func (s *TUFService) RefreshTimestamp() error {
	return s.manager.RefreshTimestamp()
}

// AddDelegation 添加委托
func (s *TUFService) AddDelegation(name string, paths []string, threshold int) error {
	return s.manager.AddDelegation(name, paths, threshold)
}

// RemoveDelegation 移除委托
func (s *TUFService) RemoveDelegation(name string) error {
	return s.manager.RemoveDelegation(name)
}

// ListDelegations 列出委托
func (s *TUFService) ListDelegations() []*signature.TUFDelegatedRole {
	return s.manager.ListDelegations()
}

// GetRootMetadata 获取Root元数据
func (s *TUFService) GetRootMetadata() ([]byte, error) {
	return s.manager.GetRootMetadata()
}

// GetTimestampMetadata 获取Timestamp元数据
func (s *TUFService) GetTimestampMetadata() ([]byte, error) {
	return s.manager.GetTimestampMetadata()
}

// GetSnapshotMetadata 获取Snapshot元数据
func (s *TUFService) GetSnapshotMetadata() ([]byte, error) {
	return s.manager.GetSnapshotMetadata()
}

// GetTargetsMetadata 获取Targets元数据
func (s *TUFService) GetTargetsMetadata() ([]byte, error) {
	return s.manager.GetTargetsMetadata()
}

// CheckExpiry 检查过期状态
func (s *TUFService) CheckExpiry() []string {
	return s.manager.CheckExpiry()
}

// ExportPublicKeys 导出公钥
func (s *TUFService) ExportPublicKeys() map[string]string {
	return s.manager.ExportPublicKeys()
}

// IsInitialized 检查是否已初始化
func (s *TUFService) IsInitialized() bool {
	return s.manager.IsInitialized()
}

// TUFTargetInfo 目标信息（用于API响应）
type TUFTargetInfo struct {
	Name   string                 `json:"name"`
	Length int64                  `json:"length"`
	Hashes map[string]string      `json:"hashes"`
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// GetTargetList 获取目标列表（用于API）
func (s *TUFService) GetTargetList() []TUFTargetInfo {
	targets := s.manager.ListTargets()
	result := make([]TUFTargetInfo, 0, len(targets))

	for name, target := range targets {
		result = append(result, TUFTargetInfo{
			Name:   name,
			Length: target.Length,
			Hashes: target.Hashes,
			Custom: target.Custom,
		})
	}

	return result
}

// TUFDelegationInfo 委托信息（用于API响应）
type TUFDelegationInfo struct {
	Name        string   `json:"name"`
	Paths       []string `json:"paths"`
	Threshold   int      `json:"threshold"`
	Terminating bool     `json:"terminating"`
}

// GetDelegationList 获取委托列表（用于API）
func (s *TUFService) GetDelegationList() []TUFDelegationInfo {
	delegations := s.manager.ListDelegations()
	result := make([]TUFDelegationInfo, 0, len(delegations))

	for _, d := range delegations {
		result = append(result, TUFDelegationInfo{
			Name:        d.Name,
			Paths:       d.Paths,
			Threshold:   d.Threshold,
			Terminating: d.Terminating,
		})
	}

	return result
}
