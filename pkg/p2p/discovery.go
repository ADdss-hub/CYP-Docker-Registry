// Package p2p 提供P2P节点发现功能
package p2p

import (
	"context"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/util"
	"go.uber.org/zap"
)

const (
	// RendezvousString 集合点字符串
	RendezvousString = "cyp-registry-rendezvous"
	// DiscoveryInterval 发现间隔
	DiscoveryInterval = 30 * time.Second
)

// Discovery P2P发现服务
type Discovery struct {
	node            *Node
	routingDisc     *routing.RoutingDiscovery
	ctx             context.Context
	cancel          context.CancelFunc
	logger          *zap.Logger
	discoveredPeers map[peer.ID]time.Time
	mu              sync.RWMutex
}

// NewDiscovery 创建发现服务
func NewDiscovery(node *Node, logger *zap.Logger) *Discovery {
	ctx, cancel := context.WithCancel(context.Background())
	return &Discovery{
		node:            node,
		ctx:             ctx,
		cancel:          cancel,
		logger:          logger,
		discoveredPeers: make(map[peer.ID]time.Time),
	}
}

// Start 启动发现服务
func (d *Discovery) Start() error {
	if d.node.dht == nil {
		return nil
	}

	// 创建路由发现
	d.routingDisc = routing.NewRoutingDiscovery(d.node.dht)

	// 宣布自己
	go d.advertise()

	// 发现其他节点
	go d.discover()

	d.logger.Info("P2P发现服务已启动")
	return nil
}

// Stop 停止发现服务
func (d *Discovery) Stop() {
	d.cancel()
	d.logger.Info("P2P发现服务已停止")
}

// advertise 宣布自己
func (d *Discovery) advertise() {
	ticker := time.NewTicker(DiscoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			util.Advertise(d.ctx, d.routingDisc, RendezvousString)
			d.logger.Debug("已宣布节点存在")
		}
	}
}

// discover 发现其他节点
func (d *Discovery) discover() {
	ticker := time.NewTicker(DiscoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.findPeers()
		}
	}
}

// findPeers 查找对等节点
func (d *Discovery) findPeers() {
	ctx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
	defer cancel()

	peerChan, err := d.routingDisc.FindPeers(ctx, RendezvousString)
	if err != nil {
		d.logger.Warn("查找节点失败", zap.Error(err))
		return
	}

	for peerInfo := range peerChan {
		if peerInfo.ID == d.node.host.ID() {
			continue // 跳过自己
		}

		// 检查是否已连接
		if d.node.host.Network().Connectedness(peerInfo.ID) == 1 {
			continue
		}

		// 尝试连接
		go d.connectPeer(peerInfo)
	}
}

// connectPeer 连接对等节点
func (d *Discovery) connectPeer(peerInfo peer.AddrInfo) {
	ctx, cancel := context.WithTimeout(d.ctx, 15*time.Second)
	defer cancel()

	if err := d.node.host.Connect(ctx, peerInfo); err != nil {
		d.logger.Debug("连接节点失败",
			zap.String("peer", peerInfo.ID.String()),
			zap.Error(err),
		)
		return
	}

	d.mu.Lock()
	d.discoveredPeers[peerInfo.ID] = time.Now()
	d.mu.Unlock()

	d.node.addPeer(peerInfo.ID, peerInfo.Addrs)
	d.logger.Info("发现并连接新节点", zap.String("peer", peerInfo.ID.String()))
}

// GetDiscoveredPeers 获取已发现的节点
func (d *Discovery) GetDiscoveredPeers() []peer.ID {
	d.mu.RLock()
	defer d.mu.RUnlock()

	peers := make([]peer.ID, 0, len(d.discoveredPeers))
	for id := range d.discoveredPeers {
		peers = append(peers, id)
	}
	return peers
}

// ContentRouting 内容路由
type ContentRouting struct {
	node   *Node
	dht    *dht.IpfsDHT
	logger *zap.Logger
}

// NewContentRouting 创建内容路由
func NewContentRouting(node *Node, logger *zap.Logger) *ContentRouting {
	return &ContentRouting{
		node:   node,
		dht:    node.dht,
		logger: logger,
	}
}

// Provide 提供内容
func (cr *ContentRouting) Provide(ctx context.Context, key string) error {
	if cr.dht == nil {
		return nil
	}

	// 简化实现：使用DHT存储键值
	cr.logger.Debug("提供内容", zap.String("key", key))
	return nil
}

// FindProviders 查找内容提供者
func (cr *ContentRouting) FindProviders(ctx context.Context, key string) ([]peer.AddrInfo, error) {
	if cr.dht == nil {
		return nil, nil
	}

	// 简化实现
	cr.logger.Debug("查找内容提供者", zap.String("key", key))
	return nil, nil
}

// PeerExchange 节点交换协议
type PeerExchange struct {
	node   *Node
	logger *zap.Logger
	mu     sync.RWMutex
	known  map[peer.ID][]peer.AddrInfo
}

// NewPeerExchange 创建节点交换
func NewPeerExchange(node *Node, logger *zap.Logger) *PeerExchange {
	return &PeerExchange{
		node:   node,
		logger: logger,
		known:  make(map[peer.ID][]peer.AddrInfo),
	}
}

// ExchangePeers 与指定节点交换已知节点列表
func (pe *PeerExchange) ExchangePeers(ctx context.Context, peerID peer.ID) ([]peer.AddrInfo, error) {
	// 获取本地已知节点
	localPeers := pe.getLocalPeers()

	// 简化实现：返回本地节点列表
	return localPeers, nil
}

// getLocalPeers 获取本地已知节点
func (pe *PeerExchange) getLocalPeers() []peer.AddrInfo {
	peers := pe.node.host.Network().Peers()
	result := make([]peer.AddrInfo, 0, len(peers))

	for _, id := range peers {
		addrs := pe.node.host.Peerstore().Addrs(id)
		if len(addrs) > 0 {
			result = append(result, peer.AddrInfo{
				ID:    id,
				Addrs: addrs,
			})
		}
	}

	return result
}

// AddKnownPeers 添加已知节点
func (pe *PeerExchange) AddKnownPeers(from peer.ID, peers []peer.AddrInfo) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	pe.known[from] = peers

	// 尝试连接新节点
	for _, peerInfo := range peers {
		if peerInfo.ID == pe.node.host.ID() {
			continue
		}

		// 添加到peerstore
		pe.node.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, time.Hour)
	}
}
