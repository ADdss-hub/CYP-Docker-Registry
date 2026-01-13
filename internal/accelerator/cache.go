// Package accelerator provides image acceleration and caching functionality.
package accelerator

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheEntry represents a cached item.
type CacheEntry struct {
	Digest      string    `json:"digest"`
	Size        int64     `json:"size"`
	LastAccess  time.Time `json:"last_access"`
	AccessCount int       `json:"access_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// CacheStats represents cache statistics.
type CacheStats struct {
	TotalSize    int64 `json:"total_size"`
	MaxSize      int64 `json:"max_size"`
	EntryCount   int   `json:"entry_count"`
	HitCount     int64 `json:"hit_count"`
	MissCount    int64 `json:"miss_count"`
	HitRate      float64 `json:"hit_rate"`
}

// CacheIndex represents the cache index stored on disk.
type CacheIndex struct {
	Entries map[string]*CacheEntry `json:"entries"`
}

// LRUCache implements an LRU cache for image layers.
type LRUCache struct {
	cachePath   string
	maxSize     int64
	mu          sync.RWMutex
	entries     map[string]*list.Element
	lruList     *list.List
	currentSize int64
	hitCount    int64
	missCount   int64
}

// lruItem represents an item in the LRU list.
type lruItem struct {
	entry *CacheEntry
}

// NewLRUCache creates a new LRU cache instance.
func NewLRUCache(cachePath string, maxSize int64) (*LRUCache, error) {
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &LRUCache{
		cachePath: cachePath,
		maxSize:   maxSize,
		entries:   make(map[string]*list.Element),
		lruList:   list.New(),
	}

	// Load existing cache index
	if err := cache.loadIndex(); err != nil {
		// Index load failure is not fatal, start fresh
		cache.entries = make(map[string]*list.Element)
		cache.lruList = list.New()
	}

	return cache, nil
}


// Get retrieves a cached blob by digest.
func (c *LRUCache) Get(digest string) (io.ReadCloser, int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.entries[digest]
	if !ok {
		c.missCount++
		return nil, 0, fmt.Errorf("cache miss: %s", digest)
	}

	// Update access time and move to front
	item := elem.Value.(*lruItem)
	item.entry.LastAccess = time.Now()
	item.entry.AccessCount++
	c.lruList.MoveToFront(elem)

	// Open the cached file
	filePath := c.getBlobPath(digest)
	file, err := os.Open(filePath)
	if err != nil {
		c.missCount++
		// Remove from cache if file doesn't exist
		c.removeEntry(digest)
		return nil, 0, fmt.Errorf("cache file not found: %w", err)
	}

	c.hitCount++
	return file, item.entry.Size, nil
}

// Put stores a blob in the cache.
func (c *LRUCache) Put(digest string, data io.Reader) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already cached
	if _, ok := c.entries[digest]; ok {
		return 0, nil // Already cached
	}

	// Write to temp file first
	tempFile, err := os.CreateTemp(c.cachePath, "cache-*.tmp")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		tempFile.Close()
		os.Remove(tempPath)
	}()

	// Calculate hash while writing
	hash := sha256.New()
	writer := io.MultiWriter(tempFile, hash)

	size, err := io.Copy(writer, data)
	if err != nil {
		return 0, fmt.Errorf("failed to write cache: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return 0, fmt.Errorf("failed to close temp file: %w", err)
	}

	// Verify digest if provided
	calculatedDigest := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if digest != "" && digest != calculatedDigest {
		return 0, fmt.Errorf("digest mismatch: expected %s, got %s", digest, calculatedDigest)
	}

	// Evict entries if needed to make room
	for c.currentSize+size > c.maxSize && c.lruList.Len() > 0 {
		c.evictOldest()
	}

	// Move to final location
	finalPath := c.getBlobPath(digest)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return 0, fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := os.Rename(tempPath, finalPath); err != nil {
		return 0, fmt.Errorf("failed to move cache file: %w", err)
	}

	// Add to cache index
	entry := &CacheEntry{
		Digest:      digest,
		Size:        size,
		LastAccess:  time.Now(),
		AccessCount: 1,
		CreatedAt:   time.Now(),
	}

	elem := c.lruList.PushFront(&lruItem{entry: entry})
	c.entries[digest] = elem
	c.currentSize += size

	// Save index
	c.saveIndex()

	return size, nil
}

// PutWithReader stores a blob and returns a reader for the cached data.
func (c *LRUCache) PutWithReader(digest string, data io.Reader) (io.ReadCloser, int64, error) {
	writtenSize, err := c.Put(digest, data)
	if err != nil {
		return nil, 0, err
	}

	// Return a reader for the cached file
	reader, size, err := c.Get(digest)
	if err != nil {
		return nil, writtenSize, err
	}
	return reader, size, nil
}

// Exists checks if a blob is cached.
func (c *LRUCache) Exists(digest string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.entries[digest]
	return ok
}

// Delete removes a blob from the cache.
func (c *LRUCache) Delete(digest string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.removeEntry(digest)
}

// Clear removes all entries from the cache.
func (c *LRUCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove all files
	for digest := range c.entries {
		filePath := c.getBlobPath(digest)
		os.Remove(filePath)
	}

	// Reset state
	c.entries = make(map[string]*list.Element)
	c.lruList = list.New()
	c.currentSize = 0
	c.hitCount = 0
	c.missCount = 0

	// Save empty index
	return c.saveIndex()
}

// Stats returns cache statistics.
func (c *LRUCache) Stats() *CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hitRate := float64(0)
	total := c.hitCount + c.missCount
	if total > 0 {
		hitRate = float64(c.hitCount) / float64(total)
	}

	return &CacheStats{
		TotalSize:  c.currentSize,
		MaxSize:    c.maxSize,
		EntryCount: len(c.entries),
		HitCount:   c.hitCount,
		MissCount:  c.missCount,
		HitRate:    hitRate,
	}
}


// evictOldest removes the least recently used entry.
func (c *LRUCache) evictOldest() {
	elem := c.lruList.Back()
	if elem == nil {
		return
	}

	item := elem.Value.(*lruItem)
	c.removeEntry(item.entry.Digest)
}

// removeEntry removes an entry from the cache (internal, no lock).
func (c *LRUCache) removeEntry(digest string) error {
	elem, ok := c.entries[digest]
	if !ok {
		return nil
	}

	item := elem.Value.(*lruItem)
	c.currentSize -= item.entry.Size
	c.lruList.Remove(elem)
	delete(c.entries, digest)

	// Remove file
	filePath := c.getBlobPath(digest)
	os.Remove(filePath)

	return nil
}

// getBlobPath returns the file path for a cached blob.
func (c *LRUCache) getBlobPath(digest string) string {
	hash := digest
	if len(digest) > 7 && digest[:7] == "sha256:" {
		hash = digest[7:]
	}
	if len(hash) < 2 {
		return filepath.Join(c.cachePath, hash)
	}
	return filepath.Join(c.cachePath, hash[:2], hash)
}

// getIndexPath returns the path to the cache index file.
func (c *LRUCache) getIndexPath() string {
	return filepath.Join(c.cachePath, "cache_index.json")
}

// loadIndex loads the cache index from disk.
func (c *LRUCache) loadIndex() error {
	indexPath := c.getIndexPath()
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var index CacheIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return err
	}

	// Rebuild LRU list from index, sorted by last access time
	type entryWithTime struct {
		entry *CacheEntry
	}
	var entries []*entryWithTime
	for _, entry := range index.Entries {
		// Verify file exists
		filePath := c.getBlobPath(entry.Digest)
		if _, err := os.Stat(filePath); err == nil {
			entries = append(entries, &entryWithTime{entry: entry})
		}
	}

	// Sort by last access time (oldest first, so newest ends up at front)
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].entry.LastAccess.After(entries[j].entry.LastAccess) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Add to LRU list (oldest first, so they're at the back)
	for _, e := range entries {
		elem := c.lruList.PushFront(&lruItem{entry: e.entry})
		c.entries[e.entry.Digest] = elem
		c.currentSize += e.entry.Size
	}

	return nil
}

// saveIndex saves the cache index to disk.
func (c *LRUCache) saveIndex() error {
	index := CacheIndex{
		Entries: make(map[string]*CacheEntry),
	}

	for digest, elem := range c.entries {
		item := elem.Value.(*lruItem)
		index.Entries[digest] = item.entry
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.getIndexPath(), data, 0644)
}

// GetEntries returns all cache entries (for testing/debugging).
func (c *LRUCache) GetEntries() []*CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var entries []*CacheEntry
	for elem := c.lruList.Front(); elem != nil; elem = elem.Next() {
		item := elem.Value.(*lruItem)
		entries = append(entries, item.entry)
	}
	return entries
}

// GetLRUOrder returns digests in LRU order (most recent first).
func (c *LRUCache) GetLRUOrder() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var order []string
	for elem := c.lruList.Front(); elem != nil; elem = elem.Next() {
		item := elem.Value.(*lruItem)
		order = append(order, item.entry.Digest)
	}
	return order
}

// CurrentSize returns the current cache size.
func (c *LRUCache) CurrentSize() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentSize
}

// MaxSize returns the maximum cache size.
func (c *LRUCache) MaxSize() int64 {
	return c.maxSize
}
