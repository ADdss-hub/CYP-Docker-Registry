// Package service 提供P2P服务
package service

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"cyp-docker-registry/pkg/p2p"

	"go.uber.org/zap"
)

// P2PService P2P服务
type P2PService struct {
	node         *p2p.Node
	discovery    *p2p.Discovery
	natTraversal *p2p.NATTraversal
	holePunch    *p2p.HolePunch
	blobStore    p2p.BlobStore
	config       *p2p.Config
	logger       *zap.Logger
	started      bool
	mu           sync.RWMutex
}

// P2PStatus P2P状态
type P2PStatus struct {
	Enabled        bool           `json:"enabled"`
	Running        bool           `json:"running"`
	PeerID         string         `json:"peer_id"`
	Addresses      []string       `json:"addresses"`
	PeerCount      int            `json:"peer_count"`
	ConnectedPeers int            `json:"connected_peers"`
	BytesSent      int64          `json:"bytes_sent"`
	BytesReceived  int64          `json:"bytes_received"`
	BlobsShared    int64          `json:"blobs_shared"`
	BlobsReceived  int64          `json:"blobs_received"`
	Uptime         string         `json:"uptime"`
	NATStatus      *p2p.NATStatus `json:"nat_status"`
	ShareMode      string         `json:"share_mode"`
}

// P2PPeerInfo P2P节点信息
type P2PPeerInfo struct {
	ID            string    `json:"id"`
	Addresses     []string  `json:"addresses"`
	ConnectedAt   time.Time `json:"connected_at"`
	LastSeen      time.Time `json:"last_seen"`
	BytesSent     int64     `json:"bytes_sent"`
	BytesReceived int64     `json:"bytes_received"`
	Latency       string    `json:"latency"`
}

// NewP2PService 创建P2P服务
func NewP2PService(config *p2p.Config, blobPath string, logger *zap.Logger) (*P2PService, error) {
	if config == nil {
		config = p2p.DefaultConfig()
	}

	// 创建Blob存储
	blobStore, err := p2p.NewFileBlobStore(blobPath, logger)
	if err != nil {
		return nil, fmt.Errorf("创建Blob存储失败: %w", err)
	}

	// 创建P2P节点
	node, err := p2p.NewNode(config, blobStore, logger)
	if err != nil {
		return nil, fmt.Errorf("创建P2P节点失败: %w", err)
	}

	return &P2PService{
		node:      node,
		blobStore: blobStore,
		config:    config,
		logger:    logger,
	}, nil
}

// Start 启动P2P服务
func (s *P2PService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	if !s.config.Enabled {
		s.logger.Info("P2P服务已禁用")
		return nil
	}

	// 启动节点
	if err := s.node.Start(); err != nil {
		return fmt.Errorf("启动P2P节点失败: %w", err)
	}

	// 启动发现服务
	s.discovery = p2p.NewDiscovery(s.node, s.logger)
	if err := s.discovery.Start(); err != nil {
		s.logger.Warn("启动发现服务失败", zap.Error(err))
	}

	// 启动NAT穿透
	s.natTraversal = p2p.NewNATTraversal(s.node, s.logger)
	if err := s.natTraversal.Start(context.Background()); err != nil {
		s.logger.Warn("启动NAT穿透失败", zap.Error(err))
	}

	// 创建打洞服务
	s.holePunch = p2p.NewHolePunch(s.node, s.logger)

	s.started = true
	s.logger.Info("P2P服务已启动")
	return nil
}

// Stop 停止P2P服务
func (s *P2PService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	if s.discovery != nil {
		s.discovery.Stop()
	}

	if err := s.node.Stop(); err != nil {
		return fmt.Errorf("停止P2P节点失败: %w", err)
	}

	s.started = false
	s.logger.Info("P2P服务已停止")
	return nil
}

// GetStatus 获取P2P状态
func (s *P2PService) GetStatus() *P2PStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := &P2PStatus{
		Enabled:   s.config.Enabled,
		Running:   s.started && s.node.IsEnabled(),
		ShareMode: s.config.ShareMode,
	}

	if !status.Running {
		return status
	}

	// 获取节点统计
	stats := s.node.GetStats()
	status.PeerID = s.node.PeerID()
	status.Addresses = s.node.Addresses()
	status.PeerCount = stats.PeerCount
	status.ConnectedPeers = stats.ConnectedPeers
	status.BytesSent = stats.TotalBytesSent
	status.BytesReceived = stats.TotalBytesRecv
	status.BlobsShared = stats.BlobsShared
	status.BlobsReceived = stats.BlobsReceived
	status.Uptime = stats.Uptime.String()

	// 获取NAT状态
	if s.natTraversal != nil {
		status.NATStatus = s.natTraversal.GetStatus()
	}

	return status
}

// GetPeers 获取对等节点列表
func (s *P2PService) GetPeers() []*P2PPeerInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started || !s.node.IsEnabled() {
		return nil
	}

	peers := s.node.GetPeers()
	result := make([]*P2PPeerInfo, 0, len(peers))

	for _, p := range peers {
		addrs := make([]string, 0, len(p.Addrs))
		for _, addr := range p.Addrs {
			addrs = append(addrs, addr.String())
		}

		result = append(result, &P2PPeerInfo{
			ID:            p.ID.String(),
			Addresses:     addrs,
			ConnectedAt:   p.ConnectedAt,
			LastSeen:      p.LastSeen,
			BytesSent:     p.BytesSent,
			BytesReceived: p.BytesReceived,
			Latency:       p.Latency.String(),
		})
	}

	return result
}

// RequestBlob 从P2P网络请求Blob
func (s *P2PService) RequestBlob(ctx context.Context, digest string) (io.ReadCloser, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started || !s.node.IsEnabled() {
		return nil, 0, fmt.Errorf("P2P服务未运行")
	}

	return s.node.RequestBlob(ctx, digest)
}

// HasBlob 检查P2P网络中是否有Blob
func (s *P2PService) HasBlob(ctx context.Context, digest string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started || !s.node.IsEnabled() {
		return false
	}

	has, _ := s.node.HasBlob(ctx, digest)
	return has
}

// AnnounceBlob 向P2P网络宣布拥有某个Blob
func (s *P2PService) AnnounceBlob(ctx context.Context, digest string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started || !s.node.IsEnabled() {
		return nil
	}

	return s.node.AnnounceBlob(ctx, digest)
}

// StoreBlob 存储Blob到本地
func (s *P2PService) StoreBlob(digest string, reader io.Reader, size int64) error {
	return s.blobStore.Put(digest, reader, size)
}

// GetLocalBlob 获取本地Blob
func (s *P2PService) GetLocalBlob(digest string) (io.ReadCloser, int64, error) {
	return s.blobStore.Get(digest)
}

// HasLocalBlob 检查本地是否有Blob
func (s *P2PService) HasLocalBlob(digest string) bool {
	has, _ := s.blobStore.Has(digest)
	return has
}

// DeleteBlob 删除Blob
func (s *P2PService) DeleteBlob(digest string) error {
	return s.blobStore.Delete(digest)
}

// ListBlobs 列出所有本地Blob
func (s *P2PService) ListBlobs() ([]string, error) {
	return s.blobStore.List()
}

// ConnectPeer 连接指定节点
func (s *P2PService) ConnectPeer(ctx context.Context, addr string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started || !s.node.IsEnabled() {
		return fmt.Errorf("P2P服务未运行")
	}

	// TODO: 实现连接指定节点
	return nil
}

// DisconnectPeer 断开指定节点
func (s *P2PService) DisconnectPeer(peerID string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started || !s.node.IsEnabled() {
		return fmt.Errorf("P2P服务未运行")
	}

	// TODO: 实现断开指定节点
	return nil
}

// UpdateConfig 更新配置
func (s *P2PService) UpdateConfig(config *p2p.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wasRunning := s.started

	// 如果正在运行，先停止
	if wasRunning {
		s.mu.Unlock()
		s.Stop()
		s.mu.Lock()
	}

	s.config = config

	// 如果之前在运行，重新启动
	if wasRunning && config.Enabled {
		s.mu.Unlock()
		return s.Start()
	}

	return nil
}

// IsEnabled 检查P2P是否启用
func (s *P2PService) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.Enabled
}

// IsRunning 检查P2P是否运行中
func (s *P2PService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started && s.node.IsEnabled()
}
