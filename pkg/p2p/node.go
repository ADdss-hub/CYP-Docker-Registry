// Package p2p 提供基于 libp2p 的去中心化镜像分发功能
package p2p

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
)

const (
	// ProtocolID P2P协议标识
	ProtocolID = "/cyp-docker-registry/1.0.0"
	// BlobProtocolID Blob传输协议
	BlobProtocolID = "/cyp-docker-registry/blob/1.0.0"
	// MetaProtocolID 元数据协议
	MetaProtocolID = "/cyp-docker-registry/meta/1.0.0"
	// DiscoveryServiceTag mDNS发现标签
	DiscoveryServiceTag = "cyp-docker-registry-discovery"
)

// Config P2P节点配置
type Config struct {
	Enabled          bool     `yaml:"enabled" json:"enabled"`
	ListenPort       int      `yaml:"listen_port" json:"listen_port"`
	BootstrapPeers   []string `yaml:"bootstrap_peers" json:"bootstrap_peers"`
	MaxConnections   int      `yaml:"max_connections" json:"max_connections"`
	EnableRelay      bool     `yaml:"enable_relay" json:"enable_relay"`
	EnableNATPortMap bool     `yaml:"enable_nat_port_map" json:"enable_nat_port_map"`
	DataDir          string   `yaml:"data_dir" json:"data_dir"`
	ShareMode        string   `yaml:"share_mode" json:"share_mode"` // all/selective/none
	BandwidthLimit   string   `yaml:"bandwidth_limit" json:"bandwidth_limit"`
	EnableMDNS       bool     `yaml:"enable_mdns" json:"enable_mdns"`
	PrivateKeyPath   string   `yaml:"private_key_path" json:"private_key_path"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Enabled:          false,
		ListenPort:       4001,
		BootstrapPeers:   []string{},
		MaxConnections:   50,
		EnableRelay:      true,
		EnableNATPortMap: true,
		DataDir:          "/app/data/p2p",
		ShareMode:        "selective",
		BandwidthLimit:   "100Mbps",
		EnableMDNS:       true,
	}
}

// Node P2P节点
type Node struct {
	config     *Config
	host       host.Host
	dht        *dht.IpfsDHT
	ctx        context.Context
	cancel     context.CancelFunc
	logger     *zap.Logger
	peers      map[peer.ID]*PeerInfo
	peersMu    sync.RWMutex
	blobStore  BlobStore
	handlers   map[protocol.ID]network.StreamHandler
	handlersMu sync.RWMutex
	stats      *NodeStats
	statsMu    sync.RWMutex
}

// PeerInfo 对等节点信息
type PeerInfo struct {
	ID            peer.ID
	Addrs         []multiaddr.Multiaddr
	ConnectedAt   time.Time
	LastSeen      time.Time
	BytesSent     int64
	BytesReceived int64
	Latency       time.Duration
	Version       string
}

// NodeStats 节点统计信息
type NodeStats struct {
	PeerCount       int           `json:"peer_count"`
	ConnectedPeers  int           `json:"connected_peers"`
	TotalBytesSent  int64         `json:"total_bytes_sent"`
	TotalBytesRecv  int64         `json:"total_bytes_recv"`
	BlobsShared     int64         `json:"blobs_shared"`
	BlobsReceived   int64         `json:"blobs_received"`
	Uptime          time.Duration `json:"uptime"`
	StartTime       time.Time     `json:"start_time"`
	NATStatus       string        `json:"nat_status"`
	PublicAddresses []string      `json:"public_addresses"`
}

// BlobStore Blob存储接口
type BlobStore interface {
	Has(digest string) (bool, error)
	Get(digest string) (io.ReadCloser, int64, error)
	Put(digest string, reader io.Reader, size int64) error
	Delete(digest string) error
	List() ([]string, error)
}

// NewNode 创建新的P2P节点
func NewNode(config *Config, blobStore BlobStore, logger *zap.Logger) (*Node, error) {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	node := &Node{
		config:    config,
		ctx:       ctx,
		cancel:    cancel,
		logger:    logger,
		peers:     make(map[peer.ID]*PeerInfo),
		blobStore: blobStore,
		handlers:  make(map[protocol.ID]network.StreamHandler),
		stats: &NodeStats{
			StartTime: time.Now(),
		},
	}

	return node, nil
}

// Start 启动P2P节点
func (n *Node) Start() error {
	if !n.config.Enabled {
		n.logger.Info("P2P功能已禁用")
		return nil
	}

	n.logger.Info("正在启动P2P节点...")

	// 生成或加载密钥
	priv, err := n.loadOrGenerateKey()
	if err != nil {
		return fmt.Errorf("加载密钥失败: %w", err)
	}

	// 构建libp2p选项
	opts := []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", n.config.ListenPort),
			fmt.Sprintf("/ip6/::/tcp/%d", n.config.ListenPort),
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", n.config.ListenPort),
		),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.ConnectionManager(nil), // 使用默认连接管理器
	}

	// NAT穿透
	if n.config.EnableNATPortMap {
		opts = append(opts, libp2p.NATPortMap())
	}

	// 中继支持
	if n.config.EnableRelay {
		opts = append(opts, libp2p.EnableRelay())
	}

	// 创建host
	h, err := libp2p.New(opts...)
	if err != nil {
		return fmt.Errorf("创建libp2p host失败: %w", err)
	}
	n.host = h

	// 创建DHT
	kadDHT, err := dht.New(n.ctx, h, dht.Mode(dht.ModeAutoServer))
	if err != nil {
		return fmt.Errorf("创建DHT失败: %w", err)
	}
	n.dht = kadDHT

	// 启动DHT
	if err := n.dht.Bootstrap(n.ctx); err != nil {
		return fmt.Errorf("DHT bootstrap失败: %w", err)
	}

	// 注册协议处理器
	n.registerHandlers()

	// 连接引导节点
	if err := n.connectBootstrapPeers(); err != nil {
		n.logger.Warn("连接引导节点失败", zap.Error(err))
	}

	// 启动mDNS发现
	if n.config.EnableMDNS {
		if err := n.setupMDNS(); err != nil {
			n.logger.Warn("mDNS设置失败", zap.Error(err))
		}
	}

	// 启动后台任务
	go n.backgroundTasks()

	n.logger.Info("P2P节点已启动",
		zap.String("peer_id", h.ID().String()),
		zap.Any("addresses", h.Addrs()),
	)

	return nil
}

// Stop 停止P2P节点
func (n *Node) Stop() error {
	n.logger.Info("正在停止P2P节点...")
	n.cancel()

	if n.dht != nil {
		if err := n.dht.Close(); err != nil {
			n.logger.Warn("关闭DHT失败", zap.Error(err))
		}
	}

	if n.host != nil {
		if err := n.host.Close(); err != nil {
			return fmt.Errorf("关闭host失败: %w", err)
		}
	}

	n.logger.Info("P2P节点已停止")
	return nil
}

// loadOrGenerateKey 加载或生成密钥
func (n *Node) loadOrGenerateKey() (crypto.PrivKey, error) {
	// 生成新密钥
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, -1, rand.Reader)
	if err != nil {
		return nil, err
	}
	return priv, nil
}

// registerHandlers 注册协议处理器
func (n *Node) registerHandlers() {
	// Blob传输处理器
	n.host.SetStreamHandler(BlobProtocolID, n.handleBlobStream)
	// 元数据处理器
	n.host.SetStreamHandler(MetaProtocolID, n.handleMetaStream)
	// 通用协议处理器
	n.host.SetStreamHandler(ProtocolID, n.handleGeneralStream)
}

// connectBootstrapPeers 连接引导节点
func (n *Node) connectBootstrapPeers() error {
	for _, addrStr := range n.config.BootstrapPeers {
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			n.logger.Warn("解析引导节点地址失败", zap.String("addr", addrStr), zap.Error(err))
			continue
		}

		peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			n.logger.Warn("解析peer信息失败", zap.Error(err))
			continue
		}

		go func(pi peer.AddrInfo) {
			ctx, cancel := context.WithTimeout(n.ctx, 30*time.Second)
			defer cancel()

			if err := n.host.Connect(ctx, pi); err != nil {
				n.logger.Warn("连接引导节点失败", zap.String("peer", pi.ID.String()), zap.Error(err))
			} else {
				n.logger.Info("已连接引导节点", zap.String("peer", pi.ID.String()))
				n.addPeer(pi.ID, pi.Addrs)
			}
		}(*peerInfo)
	}
	return nil
}

// setupMDNS 设置mDNS本地发现
func (n *Node) setupMDNS() error {
	notifee := &mdnsNotifee{node: n}
	service := mdns.NewMdnsService(n.host, DiscoveryServiceTag, notifee)
	return service.Start()
}

// mdnsNotifee mDNS发现通知
type mdnsNotifee struct {
	node *Node
}

func (m *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	m.node.logger.Debug("mDNS发现新节点", zap.String("peer", pi.ID.String()))

	ctx, cancel := context.WithTimeout(m.node.ctx, 10*time.Second)
	defer cancel()

	if err := m.node.host.Connect(ctx, pi); err != nil {
		m.node.logger.Debug("连接mDNS节点失败", zap.Error(err))
	} else {
		m.node.addPeer(pi.ID, pi.Addrs)
	}
}

// addPeer 添加对等节点
func (n *Node) addPeer(id peer.ID, addrs []multiaddr.Multiaddr) {
	n.peersMu.Lock()
	defer n.peersMu.Unlock()

	if _, exists := n.peers[id]; !exists {
		n.peers[id] = &PeerInfo{
			ID:          id,
			Addrs:       addrs,
			ConnectedAt: time.Now(),
			LastSeen:    time.Now(),
		}
		n.stats.PeerCount++
	} else {
		n.peers[id].LastSeen = time.Now()
	}
}

// removePeer 移除对等节点
func (n *Node) removePeer(id peer.ID) {
	n.peersMu.Lock()
	defer n.peersMu.Unlock()

	if _, exists := n.peers[id]; exists {
		delete(n.peers, id)
		n.stats.PeerCount--
	}
}

// backgroundTasks 后台任务
func (n *Node) backgroundTasks() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.updateStats()
			n.cleanupStaleConnections()
		}
	}
}

// updateStats 更新统计信息
func (n *Node) updateStats() {
	n.statsMu.Lock()
	defer n.statsMu.Unlock()

	n.stats.Uptime = time.Since(n.stats.StartTime)
	n.stats.ConnectedPeers = len(n.host.Network().Peers())

	// 获取公网地址
	addrs := n.host.Addrs()
	n.stats.PublicAddresses = make([]string, 0, len(addrs))
	for _, addr := range addrs {
		n.stats.PublicAddresses = append(n.stats.PublicAddresses, addr.String())
	}

	// 检测NAT状态
	n.stats.NATStatus = n.detectNATStatus()
}

// detectNATStatus 检测NAT状态
func (n *Node) detectNATStatus() string {
	// 简化的NAT检测
	addrs := n.host.Addrs()
	hasPublic := false
	for _, addr := range addrs {
		addrStr := addr.String()
		if !isPrivateAddr(addrStr) {
			hasPublic = true
			break
		}
	}

	if hasPublic {
		return "public"
	}
	return "behind_nat"
}

// isPrivateAddr 检查是否为私有地址
func isPrivateAddr(addr string) bool {
	// 简化检测
	return len(addr) > 0 && (addr[0:4] == "/ip4" &&
		(addr[5:8] == "10." || addr[5:12] == "192.168" || addr[5:10] == "172."))
}

// cleanupStaleConnections 清理过期连接
func (n *Node) cleanupStaleConnections() {
	n.peersMu.Lock()
	defer n.peersMu.Unlock()

	staleThreshold := time.Now().Add(-5 * time.Minute)
	for id, info := range n.peers {
		if info.LastSeen.Before(staleThreshold) {
			// 检查是否仍然连接
			if n.host.Network().Connectedness(id) != network.Connected {
				delete(n.peers, id)
				n.stats.PeerCount--
			}
		}
	}
}

// GetStats 获取节点统计信息
func (n *Node) GetStats() *NodeStats {
	n.statsMu.RLock()
	defer n.statsMu.RUnlock()

	stats := *n.stats
	return &stats
}

// GetPeers 获取对等节点列表
func (n *Node) GetPeers() []*PeerInfo {
	n.peersMu.RLock()
	defer n.peersMu.RUnlock()

	peers := make([]*PeerInfo, 0, len(n.peers))
	for _, p := range n.peers {
		info := *p
		peers = append(peers, &info)
	}
	return peers
}

// PeerID 获取本节点ID
func (n *Node) PeerID() string {
	if n.host == nil {
		return ""
	}
	return n.host.ID().String()
}

// Addresses 获取本节点地址
func (n *Node) Addresses() []string {
	if n.host == nil {
		return nil
	}

	addrs := n.host.Addrs()
	result := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		result = append(result, addr.String())
	}
	return result
}

// IsEnabled 检查P2P是否启用
func (n *Node) IsEnabled() bool {
	return n.config.Enabled && n.host != nil
}
