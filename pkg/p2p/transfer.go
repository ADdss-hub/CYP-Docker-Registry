// Package p2p 提供P2P传输功能
package p2p

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

// MessageType 消息类型
type MessageType uint8

const (
	// MsgTypeRequest 请求消息
	MsgTypeRequest MessageType = iota + 1
	// MsgTypeResponse 响应消息
	MsgTypeResponse
	// MsgTypeBlobData Blob数据
	MsgTypeBlobData
	// MsgTypeBlobRequest Blob请求
	MsgTypeBlobRequest
	// MsgTypeHave 拥有通知
	MsgTypeHave
	// MsgTypeWant 需要通知
	MsgTypeWant
	// MsgTypePing Ping消息
	MsgTypePing
	// MsgTypePong Pong消息
	MsgTypePong
)

// Message P2P消息
type Message struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id"`
	Digest    string      `json:"digest,omitempty"`
	Size      int64       `json:"size,omitempty"`
	Data      []byte      `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// BlobRequest Blob请求
type BlobRequest struct {
	Digest string `json:"digest"`
	Offset int64  `json:"offset,omitempty"`
	Length int64  `json:"length,omitempty"`
}

// BlobResponse Blob响应
type BlobResponse struct {
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
	Found  bool   `json:"found"`
	Error  string `json:"error,omitempty"`
}

// handleBlobStream 处理Blob传输流
func (n *Node) handleBlobStream(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	n.logger.Debug("收到Blob请求", zap.String("from", remotePeer.String()))

	reader := bufio.NewReader(stream)
	writer := bufio.NewWriter(stream)

	// 读取请求
	msg, err := n.readMessage(reader)
	if err != nil {
		n.logger.Warn("读取Blob请求失败", zap.Error(err))
		return
	}

	if msg.Type != MsgTypeBlobRequest {
		n.logger.Warn("无效的消息类型", zap.Uint8("type", uint8(msg.Type)))
		return
	}

	// 检查是否有该Blob
	has, err := n.blobStore.Has(msg.Digest)
	if err != nil || !has {
		// 发送未找到响应
		resp := &Message{
			Type:      MsgTypeResponse,
			ID:        msg.ID,
			Digest:    msg.Digest,
			Error:     "blob not found",
			Timestamp: time.Now().Unix(),
		}
		n.writeMessage(writer, resp)
		writer.Flush()
		return
	}

	// 获取Blob
	blobReader, size, err := n.blobStore.Get(msg.Digest)
	if err != nil {
		resp := &Message{
			Type:      MsgTypeResponse,
			ID:        msg.ID,
			Digest:    msg.Digest,
			Error:     err.Error(),
			Timestamp: time.Now().Unix(),
		}
		n.writeMessage(writer, resp)
		writer.Flush()
		return
	}
	defer blobReader.Close()

	// 发送成功响应
	resp := &Message{
		Type:      MsgTypeResponse,
		ID:        msg.ID,
		Digest:    msg.Digest,
		Size:      size,
		Timestamp: time.Now().Unix(),
	}
	if err := n.writeMessage(writer, resp); err != nil {
		n.logger.Warn("发送响应失败", zap.Error(err))
		return
	}
	writer.Flush()

	// 发送Blob数据
	written, err := io.Copy(writer, blobReader)
	if err != nil {
		n.logger.Warn("发送Blob数据失败", zap.Error(err))
		return
	}
	writer.Flush()

	// 更新统计
	n.statsMu.Lock()
	n.stats.TotalBytesSent += written
	n.stats.BlobsShared++
	n.statsMu.Unlock()

	// 更新peer统计
	n.peersMu.Lock()
	if peerInfo, ok := n.peers[remotePeer]; ok {
		peerInfo.BytesSent += written
		peerInfo.LastSeen = time.Now()
	}
	n.peersMu.Unlock()

	n.logger.Debug("Blob传输完成",
		zap.String("digest", msg.Digest),
		zap.Int64("size", written),
		zap.String("to", remotePeer.String()),
	)
}

// handleMetaStream 处理元数据流
func (n *Node) handleMetaStream(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	n.logger.Debug("收到元数据请求", zap.String("from", remotePeer.String()))

	reader := bufio.NewReader(stream)
	writer := bufio.NewWriter(stream)

	msg, err := n.readMessage(reader)
	if err != nil {
		n.logger.Warn("读取元数据请求失败", zap.Error(err))
		return
	}

	switch msg.Type {
	case MsgTypeHave:
		// 查询是否有某个Blob
		has, _ := n.blobStore.Has(msg.Digest)
		resp := &Message{
			Type:      MsgTypeResponse,
			ID:        msg.ID,
			Digest:    msg.Digest,
			Timestamp: time.Now().Unix(),
		}
		if has {
			resp.Data = []byte("true")
		} else {
			resp.Data = []byte("false")
		}
		n.writeMessage(writer, resp)

	case MsgTypePing:
		// Ping响应
		resp := &Message{
			Type:      MsgTypePong,
			ID:        msg.ID,
			Timestamp: time.Now().Unix(),
		}
		n.writeMessage(writer, resp)

	default:
		n.logger.Warn("未知的元数据消息类型", zap.Uint8("type", uint8(msg.Type)))
	}

	writer.Flush()
}

// handleGeneralStream 处理通用流
func (n *Node) handleGeneralStream(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	n.logger.Debug("收到通用请求", zap.String("from", remotePeer.String()))

	// 更新peer最后活跃时间
	n.peersMu.Lock()
	if peerInfo, ok := n.peers[remotePeer]; ok {
		peerInfo.LastSeen = time.Now()
	}
	n.peersMu.Unlock()
}

// RequestBlob 从P2P网络请求Blob
func (n *Node) RequestBlob(ctx context.Context, digest string) (io.ReadCloser, int64, error) {
	if !n.IsEnabled() {
		return nil, 0, fmt.Errorf("P2P未启用")
	}

	// 获取连接的peers
	peers := n.host.Network().Peers()
	if len(peers) == 0 {
		return nil, 0, fmt.Errorf("没有可用的P2P节点")
	}

	// 尝试从每个peer获取
	for _, peerID := range peers {
		reader, size, err := n.requestBlobFromPeer(ctx, peerID, digest)
		if err == nil {
			return reader, size, nil
		}
		n.logger.Debug("从peer获取Blob失败",
			zap.String("peer", peerID.String()),
			zap.String("digest", digest),
			zap.Error(err),
		)
	}

	return nil, 0, fmt.Errorf("无法从P2P网络获取Blob: %s", digest)
}

// requestBlobFromPeer 从指定peer请求Blob
func (n *Node) requestBlobFromPeer(ctx context.Context, peerID peer.ID, digest string) (io.ReadCloser, int64, error) {
	// 打开流
	stream, err := n.host.NewStream(ctx, peerID, BlobProtocolID)
	if err != nil {
		return nil, 0, fmt.Errorf("打开流失败: %w", err)
	}

	reader := bufio.NewReader(stream)
	writer := bufio.NewWriter(stream)

	// 发送请求
	req := &Message{
		Type:      MsgTypeBlobRequest,
		ID:        generateMessageID(),
		Digest:    digest,
		Timestamp: time.Now().Unix(),
	}
	if err := n.writeMessage(writer, req); err != nil {
		stream.Close()
		return nil, 0, fmt.Errorf("发送请求失败: %w", err)
	}
	writer.Flush()

	// 读取响应
	resp, err := n.readMessage(reader)
	if err != nil {
		stream.Close()
		return nil, 0, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.Error != "" {
		stream.Close()
		return nil, 0, fmt.Errorf("peer返回错误: %s", resp.Error)
	}

	// 返回流读取器
	return &streamReader{
		stream: stream,
		reader: reader,
		size:   resp.Size,
		node:   n,
		peer:   peerID,
	}, resp.Size, nil
}

// streamReader 流读取器
type streamReader struct {
	stream network.Stream
	reader *bufio.Reader
	size   int64
	read   int64
	node   *Node
	peer   peer.ID
}

func (r *streamReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += int64(n)

	// 更新统计
	if n > 0 {
		r.node.statsMu.Lock()
		r.node.stats.TotalBytesRecv += int64(n)
		r.node.statsMu.Unlock()

		r.node.peersMu.Lock()
		if peerInfo, ok := r.node.peers[r.peer]; ok {
			peerInfo.BytesReceived += int64(n)
		}
		r.node.peersMu.Unlock()
	}

	return n, err
}

func (r *streamReader) Close() error {
	if r.read > 0 {
		r.node.statsMu.Lock()
		r.node.stats.BlobsReceived++
		r.node.statsMu.Unlock()
	}
	return r.stream.Close()
}

// HasBlob 检查P2P网络中是否有Blob
func (n *Node) HasBlob(ctx context.Context, digest string) (bool, peer.ID) {
	if !n.IsEnabled() {
		return false, ""
	}

	peers := n.host.Network().Peers()
	for _, peerID := range peers {
		has, err := n.queryBlobFromPeer(ctx, peerID, digest)
		if err == nil && has {
			return true, peerID
		}
	}

	return false, ""
}

// queryBlobFromPeer 查询peer是否有Blob
func (n *Node) queryBlobFromPeer(ctx context.Context, peerID peer.ID, digest string) (bool, error) {
	stream, err := n.host.NewStream(ctx, peerID, MetaProtocolID)
	if err != nil {
		return false, err
	}
	defer stream.Close()

	reader := bufio.NewReader(stream)
	writer := bufio.NewWriter(stream)

	req := &Message{
		Type:      MsgTypeHave,
		ID:        generateMessageID(),
		Digest:    digest,
		Timestamp: time.Now().Unix(),
	}
	if err := n.writeMessage(writer, req); err != nil {
		return false, err
	}
	writer.Flush()

	resp, err := n.readMessage(reader)
	if err != nil {
		return false, err
	}

	return string(resp.Data) == "true", nil
}

// AnnounceBlob 向P2P网络宣布拥有某个Blob
func (n *Node) AnnounceBlob(ctx context.Context, digest string) error {
	if !n.IsEnabled() {
		return nil
	}

	// 使用DHT提供内容
	// 这里简化实现，实际应该使用CID
	n.logger.Debug("宣布Blob", zap.String("digest", digest))
	return nil
}

// readMessage 读取消息
func (n *Node) readMessage(reader *bufio.Reader) (*Message, error) {
	// 读取长度前缀
	var length uint32
	if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	if length > 10*1024*1024 { // 10MB限制
		return nil, fmt.Errorf("消息过大: %d", length)
	}

	// 读取消息体
	data := make([]byte, length)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// writeMessage 写入消息
func (n *Node) writeMessage(writer *bufio.Writer, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// 写入长度前缀
	if err := binary.Write(writer, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}

	// 写入消息体
	_, err = writer.Write(data)
	return err
}

// generateMessageID 生成消息ID
func generateMessageID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
