// Package p2p 提供P2P Blob存储实现
package p2p

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
)

// FileBlobStore 基于文件系统的Blob存储
type FileBlobStore struct {
	basePath string
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewFileBlobStore 创建文件Blob存储
func NewFileBlobStore(basePath string, logger *zap.Logger) (*FileBlobStore, error) {
	// 确保目录存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}

	return &FileBlobStore{
		basePath: basePath,
		logger:   logger,
	}, nil
}

// Has 检查是否存在Blob
func (s *FileBlobStore) Has(digest string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.blobPath(digest)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Get 获取Blob
func (s *FileBlobStore) Get(digest string) (io.ReadCloser, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.blobPath(digest)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, fmt.Errorf("blob不存�? %s", digest)
		}
		return nil, 0, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}

	return file, info.Size(), nil
}

// Put 存储Blob
func (s *FileBlobStore) Put(digest string, reader io.Reader, size int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.blobPath(digest)

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入临时文件
	tmpPath := path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}

	written, err := io.Copy(file, reader)
	file.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("写入数据失败: %w", err)
	}

	if size > 0 && written != size {
		os.Remove(tmpPath)
		return fmt.Errorf("数据大小不匹�? 期望 %d, 实际 %d", size, written)
	}

	// 重命名为最终文�?
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("重命名文件失�? %w", err)
	}

	s.logger.Debug("存储Blob成功", zap.String("digest", digest), zap.Int64("size", written))
	return nil
}

// Delete 删除Blob
func (s *FileBlobStore) Delete(digest string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.blobPath(digest)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除Blob失败: %w", err)
	}

	s.logger.Debug("删除Blob成功", zap.String("digest", digest))
	return nil
}

// List 列出所有Blob
func (s *FileBlobStore) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var digests []string

	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// 从路径提取digest
		rel, err := filepath.Rel(s.basePath, path)
		if err != nil {
			return nil
		}

		// 跳过临时文件
		if filepath.Ext(rel) == ".tmp" {
			return nil
		}

		digests = append(digests, rel)
		return nil
	})

	return digests, err
}

// Size 获取存储大小
func (s *FileBlobStore) Size() (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalSize int64

	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize, err
}

// Count 获取Blob数量
func (s *FileBlobStore) Count() (int, error) {
	digests, err := s.List()
	if err != nil {
		return 0, err
	}
	return len(digests), nil
}

// blobPath 获取Blob文件路径
func (s *FileBlobStore) blobPath(digest string) string {
	// 使用digest的前两个字符作为子目录，避免单目录文件过�?
	if len(digest) > 2 {
		return filepath.Join(s.basePath, digest[:2], digest)
	}
	return filepath.Join(s.basePath, digest)
}

// MemoryBlobStore 内存Blob存储（用于测试）
type MemoryBlobStore struct {
	blobs map[string][]byte
	mu    sync.RWMutex
}

// NewMemoryBlobStore 创建内存Blob存储
func NewMemoryBlobStore() *MemoryBlobStore {
	return &MemoryBlobStore{
		blobs: make(map[string][]byte),
	}
}

// Has 检查是否存在Blob
func (s *MemoryBlobStore) Has(digest string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.blobs[digest]
	return exists, nil
}

// Get 获取Blob
func (s *MemoryBlobStore) Get(digest string) (io.ReadCloser, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.blobs[digest]
	if !exists {
		return nil, 0, fmt.Errorf("blob不存�? %s", digest)
	}

	return io.NopCloser(NewBytesReader(data)), int64(len(data)), nil
}

// Put 存储Blob
func (s *MemoryBlobStore) Put(digest string, reader io.Reader, size int64) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.blobs[digest] = data
	s.mu.Unlock()

	return nil
}

// Delete 删除Blob
func (s *MemoryBlobStore) Delete(digest string) error {
	s.mu.Lock()
	delete(s.blobs, digest)
	s.mu.Unlock()

	return nil
}

// List 列出所有Blob
func (s *MemoryBlobStore) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	digests := make([]string, 0, len(s.blobs))
	for digest := range s.blobs {
		digests = append(digests, digest)
	}
	return digests, nil
}

// BytesReader 字节读取�?
type BytesReader struct {
	data   []byte
	offset int
}

// NewBytesReader 创建字节读取�?
func NewBytesReader(data []byte) *BytesReader {
	return &BytesReader{data: data}
}

// Read 读取数据
func (r *BytesReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}

	n := copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}

// CachedBlobStore 带缓存的Blob存储
type CachedBlobStore struct {
	primary   BlobStore
	cache     *MemoryBlobStore
	maxCache  int64
	cacheSize int64
	logger    *zap.Logger
	mu        sync.RWMutex
}

// NewCachedBlobStore 创建带缓存的Blob存储
func NewCachedBlobStore(primary BlobStore, maxCache int64, logger *zap.Logger) *CachedBlobStore {
	return &CachedBlobStore{
		primary:  primary,
		cache:    NewMemoryBlobStore(),
		maxCache: maxCache,
		logger:   logger,
	}
}

// Has 检查是否存在Blob
func (s *CachedBlobStore) Has(digest string) (bool, error) {
	// 先检查缓�?
	if has, _ := s.cache.Has(digest); has {
		return true, nil
	}
	return s.primary.Has(digest)
}

// Get 获取Blob
func (s *CachedBlobStore) Get(digest string) (io.ReadCloser, int64, error) {
	// 先从缓存获取
	if reader, size, err := s.cache.Get(digest); err == nil {
		s.logger.Debug("从缓存获取Blob", zap.String("digest", digest))
		return reader, size, nil
	}

	// 从主存储获取
	reader, size, err := s.primary.Get(digest)
	if err != nil {
		return nil, 0, err
	}

	// 如果大小合适，加入缓存
	if size < s.maxCache/10 { // 单个文件不超过缓存的10%
		go s.addToCache(digest, reader, size)
	}

	return reader, size, nil
}

// addToCache 添加到缓�?
func (s *CachedBlobStore) addToCache(digest string, reader io.ReadCloser, size int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查缓存空�?
	if s.cacheSize+size > s.maxCache {
		return // 缓存已满
	}

	// 读取数据
	data, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		return
	}

	// 存入缓存
	s.cache.Put(digest, NewBytesReader(data), size)
	s.cacheSize += size
}

// Put 存储Blob
func (s *CachedBlobStore) Put(digest string, reader io.Reader, size int64) error {
	return s.primary.Put(digest, reader, size)
}

// Delete 删除Blob
func (s *CachedBlobStore) Delete(digest string) error {
	s.cache.Delete(digest)
	return s.primary.Delete(digest)
}

// List 列出所有Blob
func (s *CachedBlobStore) List() ([]string, error) {
	return s.primary.List()
}

// ClearCache 清空缓存
func (s *CachedBlobStore) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache = NewMemoryBlobStore()
	s.cacheSize = 0
}
