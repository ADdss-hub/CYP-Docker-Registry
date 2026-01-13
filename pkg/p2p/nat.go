// Package p2p 提供NAT穿透功能
package p2p

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
)

// NATType NAT类型
type NATType string

const (
	// NATTypeNone 无NAT（公网IP）
	NATTypeNone NATType = "none"
	// NATTypeFullCone 完全锥形NAT
	NATTypeFullCone NATType = "full_cone"
	// NATTypeRestrictedCone 受限锥形NAT
	NATTypeRestrictedCone NATType = "restricted_cone"
	// NATTypePortRestricted 端口受限锥形NAT
	NATTypePortRestricted NATType = "port_restricted"
	// NATTypeSymmetric 对称NAT
	NATTypeSymmetric NATType = "symmetric"
	// NATTypeUnknown 未知
	NATTypeUnknown NATType = "unknown"
)

// NATTraversal NAT穿透服务
type NATTraversal struct {
	node       *Node
	host       host.Host
	logger     *zap.Logger
	natType    NATType
	publicAddr string
	relays     []peer.AddrInfo
	mu         sync.RWMutex
}

// NATStatus NAT状态
type NATStatus struct {
	Type          NATType  `json:"type"`
	PublicIP      string   `json:"public_ip"`
	PublicPort    int      `json:"public_port"`
	MappedAddress string   `json:"mapped_address"`
	Reachable     bool     `json:"reachable"`
	UsingRelay    bool     `json:"using_relay"`
	RelayAddrs    []string `json:"relay_addrs"`
}

// NewNATTraversal 创建NAT穿透服务
func NewNATTraversal(node *Node, logger *zap.Logger) *NATTraversal {
	return &NATTraversal{
		node:    node,
		host:    node.host,
		logger:  logger,
		natType: NATTypeUnknown,
		relays:  make([]peer.AddrInfo, 0),
	}
}

// Start 启动NAT穿透服务
func (nt *NATTraversal) Start(ctx context.Context) error {
	// 检测NAT类型
	go nt.detectNATType(ctx)

	// 查找并连接中继节点
	go nt.findRelays(ctx)

	// 定期刷新
	go nt.refreshLoop(ctx)

	nt.logger.Info("NAT穿透服务已启动")
	return nil
}

// detectNATType 检测NAT类型
func (nt *NATTraversal) detectNATType(_ context.Context) {
	nt.mu.Lock()
	defer nt.mu.Unlock()

	// 获取本地地址
	addrs := nt.host.Addrs()
	hasPublic := false
	hasPrivate := false

	for _, addr := range addrs {
		ip := extractIP(addr)
		if ip == nil {
			continue
		}

		if isPublicIP(ip) {
			hasPublic = true
			nt.publicAddr = addr.String()
		} else {
			hasPrivate = true
		}
	}

	// 简单判断NAT类型
	if hasPublic && !hasPrivate {
		nt.natType = NATTypeNone
	} else if hasPublic && hasPrivate {
		nt.natType = NATTypeFullCone // 可能是UPnP映射
	} else {
		nt.natType = NATTypeUnknown // 需要进一步检测
	}

	nt.logger.Info("NAT类型检测完成",
		zap.String("type", string(nt.natType)),
		zap.String("public_addr", nt.publicAddr),
	)
}

// findRelays 查找中继节点
func (nt *NATTraversal) findRelays(_ context.Context) {
	// 从DHT查找中继节点
	if nt.node.dht == nil {
		return
	}

	// 简化实现：使用已连接的节点作为潜在中继
	peers := nt.host.Network().Peers()
	for _, peerID := range peers {
		// 检查节点是否支持中继
		protos, err := nt.host.Peerstore().GetProtocols(peerID)
		if err != nil {
			continue
		}

		for _, proto := range protos {
			if proto == "/libp2p/circuit/relay/0.2.0/hop" {
				addrs := nt.host.Peerstore().Addrs(peerID)
				nt.mu.Lock()
				nt.relays = append(nt.relays, peer.AddrInfo{
					ID:    peerID,
					Addrs: addrs,
				})
				nt.mu.Unlock()
				nt.logger.Debug("发现中继节点", zap.String("peer", peerID.String()))
				break
			}
		}
	}
}

// refreshLoop 定期刷新
func (nt *NATTraversal) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nt.detectNATType(ctx)
			nt.findRelays(ctx)
		}
	}
}

// GetStatus 获取NAT状态
func (nt *NATTraversal) GetStatus() *NATStatus {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	status := &NATStatus{
		Type:       nt.natType,
		Reachable:  nt.natType == NATTypeNone || nt.natType == NATTypeFullCone,
		UsingRelay: len(nt.relays) > 0 && nt.natType == NATTypeSymmetric,
		RelayAddrs: make([]string, 0),
	}

	// 解析公网地址
	if nt.publicAddr != "" {
		status.MappedAddress = nt.publicAddr
		ip := extractIPString(nt.publicAddr)
		if ip != "" {
			status.PublicIP = ip
		}
	}

	// 中继地址
	for _, relay := range nt.relays {
		for _, addr := range relay.Addrs {
			status.RelayAddrs = append(status.RelayAddrs, addr.String())
		}
	}

	return status
}

// GetRelayAddrs 获取中继地址
func (nt *NATTraversal) GetRelayAddrs() []peer.AddrInfo {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	result := make([]peer.AddrInfo, len(nt.relays))
	copy(result, nt.relays)
	return result
}

// ConnectThroughRelay 通过中继连接
func (nt *NATTraversal) ConnectThroughRelay(ctx context.Context, targetPeer peer.ID) error {
	nt.mu.RLock()
	relays := nt.relays
	nt.mu.RUnlock()

	if len(relays) == 0 {
		return fmt.Errorf("没有可用的中继节点")
	}

	// 尝试通过每个中继连接
	for _, relay := range relays {
		// 构建中继地址
		relayAddr, err := multiaddr.NewMultiaddr(
			fmt.Sprintf("/p2p/%s/p2p-circuit/p2p/%s", relay.ID.String(), targetPeer.String()),
		)
		if err != nil {
			continue
		}

		// 尝试连接
		targetInfo := peer.AddrInfo{
			ID:    targetPeer,
			Addrs: []multiaddr.Multiaddr{relayAddr},
		}

		if err := nt.host.Connect(ctx, targetInfo); err == nil {
			nt.logger.Info("通过中继连接成功",
				zap.String("relay", relay.ID.String()),
				zap.String("target", targetPeer.String()),
			)
			return nil
		}
	}

	return fmt.Errorf("无法通过中继连接到 %s", targetPeer.String())
}

// ReserveRelay 预留中继资源
func (nt *NATTraversal) ReserveRelay(ctx context.Context, relayPeer peer.ID) error {
	_, err := client.Reserve(ctx, nt.host, peer.AddrInfo{ID: relayPeer})
	if err != nil {
		return fmt.Errorf("预留中继失败: %w", err)
	}

	nt.logger.Info("已预留中继资源", zap.String("relay", relayPeer.String()))
	return nil
}

// HolePunch 打洞
type HolePunch struct {
	node   *Node
	logger *zap.Logger
}

// NewHolePunch 创建打洞服务
func NewHolePunch(node *Node, logger *zap.Logger) *HolePunch {
	return &HolePunch{
		node:   node,
		logger: logger,
	}
}

// Punch 尝试打洞连接
func (hp *HolePunch) Punch(ctx context.Context, targetPeer peer.ID) error {
	// libp2p 自动处理打洞
	// 这里只是触发连接尝试

	addrs := hp.node.host.Peerstore().Addrs(targetPeer)
	if len(addrs) == 0 {
		return fmt.Errorf("没有目标节点的地址信息")
	}

	targetInfo := peer.AddrInfo{
		ID:    targetPeer,
		Addrs: addrs,
	}

	if err := hp.node.host.Connect(ctx, targetInfo); err != nil {
		return fmt.Errorf("打洞连接失败: %w", err)
	}

	hp.logger.Info("打洞连接成功", zap.String("target", targetPeer.String()))
	return nil
}

// 辅助函数

// extractIP 从multiaddr提取IP
func extractIP(addr multiaddr.Multiaddr) net.IP {
	// 尝试提取IPv4
	if ip4, err := addr.ValueForProtocol(multiaddr.P_IP4); err == nil {
		return net.ParseIP(ip4)
	}
	// 尝试提取IPv6
	if ip6, err := addr.ValueForProtocol(multiaddr.P_IP6); err == nil {
		return net.ParseIP(ip6)
	}
	return nil
}

// extractIPString 从地址字符串提取IP
func extractIPString(addr string) string {
	ma, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return ""
	}

	if ip4, err := ma.ValueForProtocol(multiaddr.P_IP4); err == nil {
		return ip4
	}
	if ip6, err := ma.ValueForProtocol(multiaddr.P_IP6); err == nil {
		return ip6
	}
	return ""
}

// isPublicIP 检查是否为公网IP
func isPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// 检查是否为私有地址
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"fc00::/7",
		"fe80::/10",
		"::1/128",
	}

	for _, block := range privateBlocks {
		_, cidr, err := net.ParseCIDR(block)
		if err != nil {
			continue
		}
		if cidr.Contains(ip) {
			return false
		}
	}

	return true
}

// UPnPMapper UPnP端口映射
type UPnPMapper struct {
	logger      *zap.Logger
	mappedPorts map[int]int
	mu          sync.RWMutex
}

// NewUPnPMapper 创建UPnP映射器
func NewUPnPMapper(logger *zap.Logger) *UPnPMapper {
	return &UPnPMapper{
		logger:      logger,
		mappedPorts: make(map[int]int),
	}
}

// MapPort 映射端口
func (u *UPnPMapper) MapPort(internalPort, externalPort int, protocol string, description string) error {
	// libp2p 的 NATPortMap 选项会自动处理UPnP
	// 这里提供手动映射接口

	u.mu.Lock()
	u.mappedPorts[internalPort] = externalPort
	u.mu.Unlock()

	u.logger.Info("端口映射请求",
		zap.Int("internal", internalPort),
		zap.Int("external", externalPort),
		zap.String("protocol", protocol),
	)

	return nil
}

// UnmapPort 取消端口映射
func (u *UPnPMapper) UnmapPort(internalPort int) error {
	u.mu.Lock()
	delete(u.mappedPorts, internalPort)
	u.mu.Unlock()

	u.logger.Info("取消端口映射", zap.Int("port", internalPort))
	return nil
}

// GetMappedPorts 获取已映射的端口
func (u *UPnPMapper) GetMappedPorts() map[int]int {
	u.mu.RLock()
	defer u.mu.RUnlock()

	result := make(map[int]int)
	for k, v := range u.mappedPorts {
		result[k] = v
	}
	return result
}
