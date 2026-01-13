// Package signature 提供 TUF (The Update Framework) 管理功能
package signature

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// TUF 角色类型
const (
	RoleRoot      = "root"
	RoleTargets   = "targets"
	RoleSnapshot  = "snapshot"
	RoleTimestamp = "timestamp"
)

// TUFConfig TUF配置
type TUFConfig struct {
	RepoPath           string        `yaml:"repo_path" json:"repo_path"`
	KeysPath           string        `yaml:"keys_path" json:"keys_path"`
	RootThreshold      int           `yaml:"root_threshold" json:"root_threshold"`
	TargetsThreshold   int           `yaml:"targets_threshold" json:"targets_threshold"`
	RootExpiry         time.Duration `yaml:"root_expiry" json:"root_expiry"`
	TargetsExpiry      time.Duration `yaml:"targets_expiry" json:"targets_expiry"`
	SnapshotExpiry     time.Duration `yaml:"snapshot_expiry" json:"snapshot_expiry"`
	TimestampExpiry    time.Duration `yaml:"timestamp_expiry" json:"timestamp_expiry"`
	ConsistentSnapshot bool          `yaml:"consistent_snapshot" json:"consistent_snapshot"`
}

// DefaultTUFConfig 返回默认TUF配置
func DefaultTUFConfig() *TUFConfig {
	return &TUFConfig{
		RepoPath:           "/app/data/tuf/repository",
		KeysPath:           "/app/data/tuf/keys",
		RootThreshold:      1,
		TargetsThreshold:   1,
		RootExpiry:         365 * 24 * time.Hour, // 1年
		TargetsExpiry:      90 * 24 * time.Hour,  // 90天
		SnapshotExpiry:     7 * 24 * time.Hour,   // 7天
		TimestampExpiry:    24 * time.Hour,       // 1天
		ConsistentSnapshot: true,
	}
}

// TUFKey TUF密钥
type TUFKey struct {
	ID         string            `json:"keyid"`
	Type       string            `json:"keytype"`
	Scheme     string            `json:"scheme"`
	Value      TUFKeyValue       `json:"keyval"`
	Roles      []string          `json:"-"`
	PrivateKey *ecdsa.PrivateKey `json:"-"`
}

// TUFKeyValue 密钥值
type TUFKeyValue struct {
	Public  string `json:"public"`
	Private string `json:"private,omitempty"`
}

// TUFSignature TUF签名
type TUFSignature struct {
	KeyID string `json:"keyid"`
	Sig   string `json:"sig"`
}

// TUFSigned 已签名的元数据
type TUFSigned struct {
	Signatures []TUFSignature  `json:"signatures"`
	Signed     json.RawMessage `json:"signed"`
}

// TUFRootMeta Root元数据
type TUFRootMeta struct {
	Type               string                    `json:"_type"`
	SpecVersion        string                    `json:"spec_version"`
	Version            int                       `json:"version"`
	Expires            time.Time                 `json:"expires"`
	Keys               map[string]*TUFKey        `json:"keys"`
	Roles              map[string]*TUFRoleConfig `json:"roles"`
	ConsistentSnapshot bool                      `json:"consistent_snapshot"`
}

// TUFRoleConfig 角色配置
type TUFRoleConfig struct {
	KeyIDs    []string `json:"keyids"`
	Threshold int      `json:"threshold"`
}

// TUFTargetsMeta Targets元数据
type TUFTargetsMeta struct {
	Type        string                `json:"_type"`
	SpecVersion string                `json:"spec_version"`
	Version     int                   `json:"version"`
	Expires     time.Time             `json:"expires"`
	Targets     map[string]*TUFTarget `json:"targets"`
	Delegations *TUFDelegations       `json:"delegations,omitempty"`
}

// TUFTarget 目标文件
type TUFTarget struct {
	Length int64                  `json:"length"`
	Hashes map[string]string      `json:"hashes"`
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// TUFDelegations 委托配置
type TUFDelegations struct {
	Keys  map[string]*TUFKey  `json:"keys"`
	Roles []*TUFDelegatedRole `json:"roles"`
}

// TUFDelegatedRole 委托角色
type TUFDelegatedRole struct {
	Name        string   `json:"name"`
	KeyIDs      []string `json:"keyids"`
	Threshold   int      `json:"threshold"`
	Paths       []string `json:"paths"`
	Terminating bool     `json:"terminating"`
}

// TUFSnapshotMeta Snapshot元数据
type TUFSnapshotMeta struct {
	Type        string                  `json:"_type"`
	SpecVersion string                  `json:"spec_version"`
	Version     int                     `json:"version"`
	Expires     time.Time               `json:"expires"`
	Meta        map[string]*TUFMetaFile `json:"meta"`
}

// TUFTimestampMeta Timestamp元数据
type TUFTimestampMeta struct {
	Type        string                  `json:"_type"`
	SpecVersion string                  `json:"spec_version"`
	Version     int                     `json:"version"`
	Expires     time.Time               `json:"expires"`
	Meta        map[string]*TUFMetaFile `json:"meta"`
}

// TUFMetaFile 元数据文件信息
type TUFMetaFile struct {
	Version int               `json:"version,omitempty"`
	Length  int64             `json:"length"`
	Hashes  map[string]string `json:"hashes"`
}

// TUFManager TUF管理器
type TUFManager struct {
	config    *TUFConfig
	logger    *zap.Logger
	keys      map[string]*TUFKey
	root      *TUFRootMeta
	targets   *TUFTargetsMeta
	snapshot  *TUFSnapshotMeta
	timestamp *TUFTimestampMeta
	mu        sync.RWMutex
}

// NewTUFManager 创建TUF管理器
func NewTUFManager(config *TUFConfig, logger *zap.Logger) (*TUFManager, error) {
	if config == nil {
		config = DefaultTUFConfig()
	}

	// 确保目录存在
	if err := os.MkdirAll(config.RepoPath, 0755); err != nil {
		return nil, fmt.Errorf("创建仓库目录失败: %w", err)
	}
	if err := os.MkdirAll(config.KeysPath, 0700); err != nil {
		return nil, fmt.Errorf("创建密钥目录失败: %w", err)
	}

	mgr := &TUFManager{
		config: config,
		logger: logger,
		keys:   make(map[string]*TUFKey),
	}

	// 尝试加载现有仓库
	if err := mgr.loadRepository(); err != nil {
		logger.Debug("加载TUF仓库失败，将创建新仓库", zap.Error(err))
	}

	return mgr, nil
}

// Initialize 初始化TUF仓库
func (m *TUFManager) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("初始化TUF仓库...")

	// 生成各角色密钥
	roles := []string{RoleRoot, RoleTargets, RoleSnapshot, RoleTimestamp}
	for _, role := range roles {
		key, err := m.generateKey(role)
		if err != nil {
			return fmt.Errorf("生成%s密钥失败: %w", role, err)
		}
		m.keys[key.ID] = key
		m.logger.Info("生成密钥", zap.String("role", role), zap.String("keyid", key.ID[:16]))
	}

	// 创建Root元数据
	if err := m.createRootMeta(); err != nil {
		return fmt.Errorf("创建Root元数据失败: %w", err)
	}

	// 创建Targets元数据
	if err := m.createTargetsMeta(); err != nil {
		return fmt.Errorf("创建Targets元数据失败: %w", err)
	}

	// 创建Snapshot元数据
	if err := m.createSnapshotMeta(); err != nil {
		return fmt.Errorf("创建Snapshot元数据失败: %w", err)
	}

	// 创建Timestamp元数据
	if err := m.createTimestampMeta(); err != nil {
		return fmt.Errorf("创建Timestamp元数据失败: %w", err)
	}

	// 保存所有元数据
	if err := m.saveRepository(); err != nil {
		return fmt.Errorf("保存仓库失败: %w", err)
	}

	m.logger.Info("TUF仓库初始化完成")
	return nil
}

// generateKey 生成ECDSA密钥
func (m *TUFManager) generateKey(role string) (*TUFKey, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	// 编码公钥
	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	// 计算密钥ID
	hash := sha256.Sum256(pubBytes)
	keyID := hex.EncodeToString(hash[:])

	key := &TUFKey{
		ID:         keyID,
		Type:       "ecdsa",
		Scheme:     "ecdsa-sha2-nistp256",
		Value:      TUFKeyValue{Public: string(pubPEM)},
		Roles:      []string{role},
		PrivateKey: privateKey,
	}

	// 保存私钥
	if err := m.savePrivateKey(key, role); err != nil {
		return nil, err
	}

	return key, nil
}

// savePrivateKey 保存私钥
func (m *TUFManager) savePrivateKey(key *TUFKey, role string) error {
	privBytes, err := x509.MarshalECPrivateKey(key.PrivateKey)
	if err != nil {
		return err
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privBytes,
	})

	path := filepath.Join(m.config.KeysPath, fmt.Sprintf("%s.key", role))
	return os.WriteFile(path, privPEM, 0600)
}

// loadPrivateKey 加载私钥
func (m *TUFManager) loadPrivateKey(role string) (*ecdsa.PrivateKey, error) {
	path := filepath.Join(m.config.KeysPath, fmt.Sprintf("%s.key", role))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("无效的PEM数据")
	}

	return x509.ParseECPrivateKey(block.Bytes)
}

// createRootMeta 创建Root元数据
func (m *TUFManager) createRootMeta() error {
	keys := make(map[string]*TUFKey)
	roles := make(map[string]*TUFRoleConfig)

	// 添加所有密钥
	for _, key := range m.keys {
		pubKey := &TUFKey{
			ID:     key.ID,
			Type:   key.Type,
			Scheme: key.Scheme,
			Value:  TUFKeyValue{Public: key.Value.Public},
		}
		keys[key.ID] = pubKey
	}

	// 配置角色
	for _, role := range []string{RoleRoot, RoleTargets, RoleSnapshot, RoleTimestamp} {
		var keyIDs []string
		for _, key := range m.keys {
			for _, r := range key.Roles {
				if r == role {
					keyIDs = append(keyIDs, key.ID)
				}
			}
		}

		threshold := 1
		if role == RoleRoot {
			threshold = m.config.RootThreshold
		} else if role == RoleTargets {
			threshold = m.config.TargetsThreshold
		}

		roles[role] = &TUFRoleConfig{
			KeyIDs:    keyIDs,
			Threshold: threshold,
		}
	}

	m.root = &TUFRootMeta{
		Type:               "root",
		SpecVersion:        "1.0.0",
		Version:            1,
		Expires:            time.Now().Add(m.config.RootExpiry),
		Keys:               keys,
		Roles:              roles,
		ConsistentSnapshot: m.config.ConsistentSnapshot,
	}

	return nil
}

// createTargetsMeta 创建Targets元数据
func (m *TUFManager) createTargetsMeta() error {
	m.targets = &TUFTargetsMeta{
		Type:        "targets",
		SpecVersion: "1.0.0",
		Version:     1,
		Expires:     time.Now().Add(m.config.TargetsExpiry),
		Targets:     make(map[string]*TUFTarget),
	}
	return nil
}

// createSnapshotMeta 创建Snapshot元数据
func (m *TUFManager) createSnapshotMeta() error {
	m.snapshot = &TUFSnapshotMeta{
		Type:        "snapshot",
		SpecVersion: "1.0.0",
		Version:     1,
		Expires:     time.Now().Add(m.config.SnapshotExpiry),
		Meta:        make(map[string]*TUFMetaFile),
	}
	return nil
}

// createTimestampMeta 创建Timestamp元数据
func (m *TUFManager) createTimestampMeta() error {
	m.timestamp = &TUFTimestampMeta{
		Type:        "timestamp",
		SpecVersion: "1.0.0",
		Version:     1,
		Expires:     time.Now().Add(m.config.TimestampExpiry),
		Meta:        make(map[string]*TUFMetaFile),
	}
	return nil
}

// AddTarget 添加目标文件
func (m *TUFManager) AddTarget(name string, data []byte, custom map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.targets == nil {
		return fmt.Errorf("TUF仓库未初始化")
	}

	// 计算哈希
	sha256Hash := sha256.Sum256(data)

	target := &TUFTarget{
		Length: int64(len(data)),
		Hashes: map[string]string{
			"sha256": hex.EncodeToString(sha256Hash[:]),
		},
		Custom: custom,
	}

	m.targets.Targets[name] = target
	m.targets.Version++
	m.targets.Expires = time.Now().Add(m.config.TargetsExpiry)

	// 保存目标文件
	targetPath := filepath.Join(m.config.RepoPath, "targets", name)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return err
	}

	// 更新Snapshot和Timestamp
	if err := m.updateSnapshotAndTimestamp(); err != nil {
		return err
	}

	return m.saveRepository()
}

// RemoveTarget 移除目标文件
func (m *TUFManager) RemoveTarget(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.targets == nil {
		return fmt.Errorf("TUF仓库未初始化")
	}

	if _, exists := m.targets.Targets[name]; !exists {
		return fmt.Errorf("目标不存在: %s", name)
	}

	delete(m.targets.Targets, name)
	m.targets.Version++

	// 删除目标文件
	targetPath := filepath.Join(m.config.RepoPath, "targets", name)
	os.Remove(targetPath)

	// 更新Snapshot和Timestamp
	if err := m.updateSnapshotAndTimestamp(); err != nil {
		return err
	}

	return m.saveRepository()
}

// GetTarget 获取目标信息
func (m *TUFManager) GetTarget(name string) (*TUFTarget, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.targets == nil {
		return nil, fmt.Errorf("TUF仓库未初始化")
	}

	target, exists := m.targets.Targets[name]
	if !exists {
		return nil, fmt.Errorf("目标不存在: %s", name)
	}

	return target, nil
}

// ListTargets 列出所有目标
func (m *TUFManager) ListTargets() map[string]*TUFTarget {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.targets == nil {
		return nil
	}

	result := make(map[string]*TUFTarget)
	for k, v := range m.targets.Targets {
		result[k] = v
	}
	return result
}

// VerifyTarget 验证目标文件
func (m *TUFManager) VerifyTarget(name string, data []byte) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	target, exists := m.targets.Targets[name]
	if !exists {
		return false, fmt.Errorf("目标不存在: %s", name)
	}

	// 验证长度
	if int64(len(data)) != target.Length {
		return false, fmt.Errorf("长度不匹配: 期望 %d, 实际 %d", target.Length, len(data))
	}

	// 验证哈希
	sha256Hash := sha256.Sum256(data)
	expectedHash := target.Hashes["sha256"]
	actualHash := hex.EncodeToString(sha256Hash[:])

	if expectedHash != actualHash {
		return false, fmt.Errorf("哈希不匹配")
	}

	return true, nil
}

// updateSnapshotAndTimestamp 更新Snapshot和Timestamp
func (m *TUFManager) updateSnapshotAndTimestamp() error {
	// 更新Snapshot
	targetsData, _ := json.Marshal(m.targets)
	targetsHash := sha256.Sum256(targetsData)

	m.snapshot.Meta["targets.json"] = &TUFMetaFile{
		Version: m.targets.Version,
		Length:  int64(len(targetsData)),
		Hashes:  map[string]string{"sha256": hex.EncodeToString(targetsHash[:])},
	}
	m.snapshot.Version++
	m.snapshot.Expires = time.Now().Add(m.config.SnapshotExpiry)

	// 更新Timestamp
	snapshotData, _ := json.Marshal(m.snapshot)
	snapshotHash := sha256.Sum256(snapshotData)

	m.timestamp.Meta["snapshot.json"] = &TUFMetaFile{
		Version: m.snapshot.Version,
		Length:  int64(len(snapshotData)),
		Hashes:  map[string]string{"sha256": hex.EncodeToString(snapshotHash[:])},
	}
	m.timestamp.Version++
	m.timestamp.Expires = time.Now().Add(m.config.TimestampExpiry)

	return nil
}

// signMeta 签名元数据
func (m *TUFManager) signMeta(role string, meta interface{}) (*TUFSigned, error) {
	// 序列化元数据
	signedData, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}

	// 查找角色密钥
	var signatures []TUFSignature
	for _, key := range m.keys {
		for _, r := range key.Roles {
			if r == role && key.PrivateKey != nil {
				// 计算签名
				hash := sha256.Sum256(signedData)
				r, s, err := ecdsa.Sign(rand.Reader, key.PrivateKey, hash[:])
				if err != nil {
					return nil, err
				}

				// 编码签名
				sig := append(r.Bytes(), s.Bytes()...)
				signatures = append(signatures, TUFSignature{
					KeyID: key.ID,
					Sig:   hex.EncodeToString(sig),
				})
			}
		}
	}

	return &TUFSigned{
		Signatures: signatures,
		Signed:     signedData,
	}, nil
}

// saveRepository 保存仓库
func (m *TUFManager) saveRepository() error {
	// 保存Root
	if m.root != nil {
		signed, err := m.signMeta(RoleRoot, m.root)
		if err != nil {
			return fmt.Errorf("签名Root失败: %w", err)
		}
		if err := m.saveMetaFile("root.json", signed); err != nil {
			return err
		}
	}

	// 保存Targets
	if m.targets != nil {
		signed, err := m.signMeta(RoleTargets, m.targets)
		if err != nil {
			return fmt.Errorf("签名Targets失败: %w", err)
		}
		if err := m.saveMetaFile("targets.json", signed); err != nil {
			return err
		}
	}

	// 保存Snapshot
	if m.snapshot != nil {
		signed, err := m.signMeta(RoleSnapshot, m.snapshot)
		if err != nil {
			return fmt.Errorf("签名Snapshot失败: %w", err)
		}
		if err := m.saveMetaFile("snapshot.json", signed); err != nil {
			return err
		}
	}

	// 保存Timestamp
	if m.timestamp != nil {
		signed, err := m.signMeta(RoleTimestamp, m.timestamp)
		if err != nil {
			return fmt.Errorf("签名Timestamp失败: %w", err)
		}
		if err := m.saveMetaFile("timestamp.json", signed); err != nil {
			return err
		}
	}

	return nil
}

// saveMetaFile 保存元数据文件
func (m *TUFManager) saveMetaFile(name string, data interface{}) error {
	path := filepath.Join(m.config.RepoPath, name)
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, jsonData, 0644)
}

// loadRepository 加载仓库
func (m *TUFManager) loadRepository() error {
	// 加载密钥
	roles := []string{RoleRoot, RoleTargets, RoleSnapshot, RoleTimestamp}
	for _, role := range roles {
		privKey, err := m.loadPrivateKey(role)
		if err != nil {
			continue
		}

		// 计算密钥ID
		pubBytes, _ := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
		hash := sha256.Sum256(pubBytes)
		keyID := hex.EncodeToString(hash[:])

		pubPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubBytes,
		})

		m.keys[keyID] = &TUFKey{
			ID:         keyID,
			Type:       "ecdsa",
			Scheme:     "ecdsa-sha2-nistp256",
			Value:      TUFKeyValue{Public: string(pubPEM)},
			Roles:      []string{role},
			PrivateKey: privKey,
		}
	}

	// 加载Root
	if data, err := os.ReadFile(filepath.Join(m.config.RepoPath, "root.json")); err == nil {
		var signed TUFSigned
		if err := json.Unmarshal(data, &signed); err == nil {
			var root TUFRootMeta
			if err := json.Unmarshal(signed.Signed, &root); err == nil {
				m.root = &root
			}
		}
	}

	// 加载Targets
	if data, err := os.ReadFile(filepath.Join(m.config.RepoPath, "targets.json")); err == nil {
		var signed TUFSigned
		if err := json.Unmarshal(data, &signed); err == nil {
			var targets TUFTargetsMeta
			if err := json.Unmarshal(signed.Signed, &targets); err == nil {
				m.targets = &targets
			}
		}
	}

	// 加载Snapshot
	if data, err := os.ReadFile(filepath.Join(m.config.RepoPath, "snapshot.json")); err == nil {
		var signed TUFSigned
		if err := json.Unmarshal(data, &signed); err == nil {
			var snapshot TUFSnapshotMeta
			if err := json.Unmarshal(signed.Signed, &snapshot); err == nil {
				m.snapshot = &snapshot
			}
		}
	}

	// 加载Timestamp
	if data, err := os.ReadFile(filepath.Join(m.config.RepoPath, "timestamp.json")); err == nil {
		var signed TUFSigned
		if err := json.Unmarshal(data, &signed); err == nil {
			var timestamp TUFTimestampMeta
			if err := json.Unmarshal(signed.Signed, &timestamp); err == nil {
				m.timestamp = &timestamp
			}
		}
	}

	if m.root == nil {
		return fmt.Errorf("未找到Root元数据")
	}

	return nil
}

// RefreshTimestamp 刷新Timestamp
func (m *TUFManager) RefreshTimestamp() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.timestamp == nil {
		return fmt.Errorf("TUF仓库未初始化")
	}

	m.timestamp.Version++
	m.timestamp.Expires = time.Now().Add(m.config.TimestampExpiry)

	return m.saveRepository()
}

// RotateKey 轮换密钥
func (m *TUFManager) RotateKey(role string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("轮换密钥", zap.String("role", role))

	// 生成新密钥
	newKey, err := m.generateKey(role)
	if err != nil {
		return fmt.Errorf("生成新密钥失败: %w", err)
	}

	// 移除旧密钥
	for id, key := range m.keys {
		for _, r := range key.Roles {
			if r == role {
				delete(m.keys, id)
				break
			}
		}
	}

	// 添加新密钥
	m.keys[newKey.ID] = newKey

	// 更新Root元数据
	if m.root != nil {
		// 更新密钥
		m.root.Keys[newKey.ID] = &TUFKey{
			ID:     newKey.ID,
			Type:   newKey.Type,
			Scheme: newKey.Scheme,
			Value:  TUFKeyValue{Public: newKey.Value.Public},
		}

		// 更新角色配置
		if roleConfig, exists := m.root.Roles[role]; exists {
			roleConfig.KeyIDs = []string{newKey.ID}
		}

		m.root.Version++
	}

	return m.saveRepository()
}

// GetStatus 获取TUF状态
func (m *TUFManager) GetStatus() *TUFStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := &TUFStatus{
		Initialized: m.root != nil,
		KeyCount:    len(m.keys),
	}

	if m.root != nil {
		status.RootVersion = m.root.Version
		status.RootExpires = m.root.Expires
		status.RootExpired = time.Now().After(m.root.Expires)
	}

	if m.targets != nil {
		status.TargetsVersion = m.targets.Version
		status.TargetsExpires = m.targets.Expires
		status.TargetCount = len(m.targets.Targets)
	}

	if m.snapshot != nil {
		status.SnapshotVersion = m.snapshot.Version
		status.SnapshotExpires = m.snapshot.Expires
	}

	if m.timestamp != nil {
		status.TimestampVersion = m.timestamp.Version
		status.TimestampExpires = m.timestamp.Expires
		status.TimestampExpired = time.Now().After(m.timestamp.Expires)
	}

	// 密钥信息
	status.Keys = make([]TUFKeyInfo, 0, len(m.keys))
	for _, key := range m.keys {
		status.Keys = append(status.Keys, TUFKeyInfo{
			ID:    key.ID[:16] + "...",
			Type:  key.Type,
			Roles: key.Roles,
		})
	}

	return status
}

// TUFStatus TUF状态
type TUFStatus struct {
	Initialized      bool         `json:"initialized"`
	KeyCount         int          `json:"key_count"`
	RootVersion      int          `json:"root_version"`
	RootExpires      time.Time    `json:"root_expires"`
	RootExpired      bool         `json:"root_expired"`
	TargetsVersion   int          `json:"targets_version"`
	TargetsExpires   time.Time    `json:"targets_expires"`
	TargetCount      int          `json:"target_count"`
	SnapshotVersion  int          `json:"snapshot_version"`
	SnapshotExpires  time.Time    `json:"snapshot_expires"`
	TimestampVersion int          `json:"timestamp_version"`
	TimestampExpires time.Time    `json:"timestamp_expires"`
	TimestampExpired bool         `json:"timestamp_expired"`
	Keys             []TUFKeyInfo `json:"keys"`
}

// TUFKeyInfo 密钥信息
type TUFKeyInfo struct {
	ID    string   `json:"id"`
	Type  string   `json:"type"`
	Roles []string `json:"roles"`
}

// AddDelegation 添加委托
func (m *TUFManager) AddDelegation(name string, paths []string, threshold int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.targets == nil {
		return fmt.Errorf("TUF仓库未初始化")
	}

	// 生成委托密钥
	key, err := m.generateKey(name)
	if err != nil {
		return err
	}

	// 初始化委托
	if m.targets.Delegations == nil {
		m.targets.Delegations = &TUFDelegations{
			Keys:  make(map[string]*TUFKey),
			Roles: make([]*TUFDelegatedRole, 0),
		}
	}

	// 添加密钥
	m.targets.Delegations.Keys[key.ID] = &TUFKey{
		ID:     key.ID,
		Type:   key.Type,
		Scheme: key.Scheme,
		Value:  TUFKeyValue{Public: key.Value.Public},
	}

	// 添加角色
	m.targets.Delegations.Roles = append(m.targets.Delegations.Roles, &TUFDelegatedRole{
		Name:        name,
		KeyIDs:      []string{key.ID},
		Threshold:   threshold,
		Paths:       paths,
		Terminating: false,
	})

	m.targets.Version++

	return m.saveRepository()
}

// RemoveDelegation 移除委托
func (m *TUFManager) RemoveDelegation(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.targets == nil || m.targets.Delegations == nil {
		return fmt.Errorf("没有委托配置")
	}

	// 查找并移除角色
	found := false
	newRoles := make([]*TUFDelegatedRole, 0)
	for _, role := range m.targets.Delegations.Roles {
		if role.Name == name {
			found = true
			// 移除相关密钥
			for _, keyID := range role.KeyIDs {
				delete(m.targets.Delegations.Keys, keyID)
			}
		} else {
			newRoles = append(newRoles, role)
		}
	}

	if !found {
		return fmt.Errorf("委托不存在: %s", name)
	}

	m.targets.Delegations.Roles = newRoles
	m.targets.Version++

	return m.saveRepository()
}

// ListDelegations 列出所有委托
func (m *TUFManager) ListDelegations() []*TUFDelegatedRole {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.targets == nil || m.targets.Delegations == nil {
		return nil
	}

	result := make([]*TUFDelegatedRole, len(m.targets.Delegations.Roles))
	copy(result, m.targets.Delegations.Roles)
	return result
}

// ExportPublicKeys 导出公钥
func (m *TUFManager) ExportPublicKeys() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string)
	for id, key := range m.keys {
		result[id] = key.Value.Public
	}
	return result
}

// GetRootMetadata 获取Root元数据（用于客户端验证）
func (m *TUFManager) GetRootMetadata() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path := filepath.Join(m.config.RepoPath, "root.json")
	return os.ReadFile(path)
}

// GetTimestampMetadata 获取Timestamp元数据
func (m *TUFManager) GetTimestampMetadata() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path := filepath.Join(m.config.RepoPath, "timestamp.json")
	return os.ReadFile(path)
}

// GetSnapshotMetadata 获取Snapshot元数据
func (m *TUFManager) GetSnapshotMetadata() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path := filepath.Join(m.config.RepoPath, "snapshot.json")
	return os.ReadFile(path)
}

// GetTargetsMetadata 获取Targets元数据
func (m *TUFManager) GetTargetsMetadata() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path := filepath.Join(m.config.RepoPath, "targets.json")
	return os.ReadFile(path)
}

// CheckExpiry 检查过期状态
func (m *TUFManager) CheckExpiry() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var warnings []string
	now := time.Now()

	if m.root != nil && now.After(m.root.Expires) {
		warnings = append(warnings, "Root元数据已过期")
	}

	if m.targets != nil && now.After(m.targets.Expires) {
		warnings = append(warnings, "Targets元数据已过期")
	}

	if m.snapshot != nil && now.After(m.snapshot.Expires) {
		warnings = append(warnings, "Snapshot元数据已过期")
	}

	if m.timestamp != nil && now.After(m.timestamp.Expires) {
		warnings = append(warnings, "Timestamp元数据已过期")
	}

	return warnings
}

// AutoRefresh 自动刷新过期的元数据
func (m *TUFManager) AutoRefresh() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	needSave := false

	// 刷新Timestamp（每天）
	if m.timestamp != nil && now.After(m.timestamp.Expires.Add(-1*time.Hour)) {
		m.timestamp.Version++
		m.timestamp.Expires = now.Add(m.config.TimestampExpiry)
		needSave = true
		m.logger.Info("自动刷新Timestamp")
	}

	// 刷新Snapshot（每周）
	if m.snapshot != nil && now.After(m.snapshot.Expires.Add(-24*time.Hour)) {
		m.snapshot.Version++
		m.snapshot.Expires = now.Add(m.config.SnapshotExpiry)
		needSave = true
		m.logger.Info("自动刷新Snapshot")
	}

	if needSave {
		return m.saveRepository()
	}

	return nil
}

// IsInitialized 检查是否已初始化
func (m *TUFManager) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.root != nil
}
