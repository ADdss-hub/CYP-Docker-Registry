// Package registry provides container image registry functionality.
package registry

import (
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

// Layer represents an image layer.
type Layer struct {
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
	MediaType string `json:"media_type"`
}

// ImageManifest represents image metadata.
type ImageManifest struct {
	Name      string    `json:"name"`
	Tag       string    `json:"tag"`
	Digest    string    `json:"digest"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	Layers    []Layer   `json:"layers"`
}

// TagInfo represents tag information for an image.
type TagInfo struct {
	Digest    string    `json:"digest"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	Layers    []Layer   `json:"layers"`
}

// ImageStore represents the image metadata store structure.
type ImageStore struct {
	Images map[string]map[string]*TagInfo `json:"images"` // name -> tag -> TagInfo
}

// Storage handles blob and metadata storage operations.
type Storage struct {
	blobPath string
	metaPath string
	mu       sync.RWMutex
}

// NewStorage creates a new Storage instance.
func NewStorage(blobPath, metaPath string) (*Storage, error) {
	// Ensure directories exist
	if err := os.MkdirAll(blobPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create blob directory: %w", err)
	}
	if err := os.MkdirAll(metaPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create meta directory: %w", err)
	}

	return &Storage{
		blobPath: blobPath,
		metaPath: metaPath,
	}, nil
}


// SaveBlob saves blob data and returns its digest.
func (s *Storage) SaveBlob(data io.Reader) (string, int64, error) {
	// Create temp file first
	tempFile, err := os.CreateTemp(s.blobPath, "blob-*.tmp")
	if err != nil {
		return "", 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		tempFile.Close()
		os.Remove(tempPath) // Clean up temp file if not renamed
	}()

	// Calculate hash while writing
	hash := sha256.New()
	writer := io.MultiWriter(tempFile, hash)

	size, err := io.Copy(writer, data)
	if err != nil {
		return "", 0, fmt.Errorf("failed to write blob: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return "", 0, fmt.Errorf("failed to close temp file: %w", err)
	}

	// Generate digest
	digest := "sha256:" + hex.EncodeToString(hash.Sum(nil))

	// Move to final location
	finalPath := s.getBlobPath(digest)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create blob directory: %w", err)
	}

	if err := os.Rename(tempPath, finalPath); err != nil {
		return "", 0, fmt.Errorf("failed to move blob: %w", err)
	}

	return digest, size, nil
}

// SaveBlobWithDigest saves blob data with a known digest.
func (s *Storage) SaveBlobWithDigest(digest string, data io.Reader) (int64, error) {
	finalPath := s.getBlobPath(digest)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return 0, fmt.Errorf("failed to create blob directory: %w", err)
	}

	file, err := os.Create(finalPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create blob file: %w", err)
	}
	defer file.Close()

	size, err := io.Copy(file, data)
	if err != nil {
		os.Remove(finalPath)
		return 0, fmt.Errorf("failed to write blob: %w", err)
	}

	return size, nil
}

// GetBlob retrieves blob data by digest.
func (s *Storage) GetBlob(digest string) (io.ReadCloser, int64, error) {
	blobPath := s.getBlobPath(digest)
	file, err := os.Open(blobPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, fmt.Errorf("blob not found: %s", digest)
		}
		return nil, 0, fmt.Errorf("failed to open blob: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, fmt.Errorf("failed to stat blob: %w", err)
	}

	return file, stat.Size(), nil
}

// DeleteBlob removes a blob by digest.
func (s *Storage) DeleteBlob(digest string) error {
	blobPath := s.getBlobPath(digest)
	if err := os.Remove(blobPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete blob: %w", err)
	}
	return nil
}

// BlobExists checks if a blob exists.
func (s *Storage) BlobExists(digest string) bool {
	blobPath := s.getBlobPath(digest)
	_, err := os.Stat(blobPath)
	return err == nil
}

// getBlobPath returns the file path for a blob digest.
func (s *Storage) getBlobPath(digest string) string {
	// Use first 2 chars of hash for directory sharding
	hash := digest
	if len(digest) > 7 && digest[:7] == "sha256:" {
		hash = digest[7:]
	}
	if len(hash) < 2 {
		return filepath.Join(s.blobPath, hash)
	}
	return filepath.Join(s.blobPath, hash[:2], hash)
}


// getMetaFilePath returns the path to the metadata file.
func (s *Storage) getMetaFilePath() string {
	return filepath.Join(s.metaPath, "images.json")
}

// LoadMetadata loads image metadata from JSON file.
func (s *Storage) LoadMetadata() (*ImageStore, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.loadMetadataUnsafe()
}

// loadMetadataUnsafe loads metadata without locking (internal use).
func (s *Storage) loadMetadataUnsafe() (*ImageStore, error) {
	metaFile := s.getMetaFilePath()
	data, err := os.ReadFile(metaFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty store if file doesn't exist
			return &ImageStore{
				Images: make(map[string]map[string]*TagInfo),
			}, nil
		}
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var store ImageStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	if store.Images == nil {
		store.Images = make(map[string]map[string]*TagInfo)
	}

	return &store, nil
}

// SaveMetadata saves image metadata to JSON file.
func (s *Storage) SaveMetadata(store *ImageStore) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveMetadataUnsafe(store)
}

// saveMetadataUnsafe saves metadata without locking (internal use).
func (s *Storage) saveMetadataUnsafe(store *ImageStore) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metaFile := s.getMetaFilePath()
	if err := os.WriteFile(metaFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// SaveImage saves image manifest metadata.
func (s *Storage) SaveImage(manifest *ImageManifest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	store, err := s.loadMetadataUnsafe()
	if err != nil {
		return err
	}

	// Initialize image map if needed
	if store.Images[manifest.Name] == nil {
		store.Images[manifest.Name] = make(map[string]*TagInfo)
	}

	// Save tag info
	store.Images[manifest.Name][manifest.Tag] = &TagInfo{
		Digest:    manifest.Digest,
		Size:      manifest.Size,
		CreatedAt: manifest.CreatedAt,
		Layers:    manifest.Layers,
	}

	return s.saveMetadataUnsafe(store)
}

// GetImage retrieves image manifest metadata.
func (s *Storage) GetImage(name, tag string) (*ImageManifest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	store, err := s.loadMetadataUnsafe()
	if err != nil {
		return nil, err
	}

	tags, ok := store.Images[name]
	if !ok {
		return nil, fmt.Errorf("image not found: %s", name)
	}

	tagInfo, ok := tags[tag]
	if !ok {
		return nil, fmt.Errorf("tag not found: %s:%s", name, tag)
	}

	return &ImageManifest{
		Name:      name,
		Tag:       tag,
		Digest:    tagInfo.Digest,
		Size:      tagInfo.Size,
		CreatedAt: tagInfo.CreatedAt,
		Layers:    tagInfo.Layers,
	}, nil
}

// DeleteImage removes image metadata.
func (s *Storage) DeleteImage(name, tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	store, err := s.loadMetadataUnsafe()
	if err != nil {
		return err
	}

	tags, ok := store.Images[name]
	if !ok {
		return fmt.Errorf("image not found: %s", name)
	}

	if _, ok := tags[tag]; !ok {
		return fmt.Errorf("tag not found: %s:%s", name, tag)
	}

	delete(tags, tag)

	// Remove image entry if no tags left
	if len(tags) == 0 {
		delete(store.Images, name)
	}

	return s.saveMetadataUnsafe(store)
}

// ListImages returns all images with pagination.
func (s *Storage) ListImages(page, pageSize int) ([]*ImageManifest, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	store, err := s.loadMetadataUnsafe()
	if err != nil {
		return nil, 0, err
	}

	// Collect all images
	var images []*ImageManifest
	for name, tags := range store.Images {
		for tag, info := range tags {
			images = append(images, &ImageManifest{
				Name:      name,
				Tag:       tag,
				Digest:    info.Digest,
				Size:      info.Size,
				CreatedAt: info.CreatedAt,
				Layers:    info.Layers,
			})
		}
	}

	total := len(images)

	// Apply pagination
	start := (page - 1) * pageSize
	if start >= total {
		return []*ImageManifest{}, total, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	return images[start:end], total, nil
}

// SearchImages searches images by keyword.
func (s *Storage) SearchImages(keyword string, page, pageSize int) ([]*ImageManifest, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	store, err := s.loadMetadataUnsafe()
	if err != nil {
		return nil, 0, err
	}

	// Collect matching images
	var images []*ImageManifest
	for name, tags := range store.Images {
		for tag, info := range tags {
			// Match keyword in name or tag
			if containsIgnoreCase(name, keyword) || containsIgnoreCase(tag, keyword) {
				images = append(images, &ImageManifest{
					Name:      name,
					Tag:       tag,
					Digest:    info.Digest,
					Size:      info.Size,
					CreatedAt: info.CreatedAt,
					Layers:    info.Layers,
				})
			}
		}
	}

	total := len(images)

	// Apply pagination
	start := (page - 1) * pageSize
	if start >= total {
		return []*ImageManifest{}, total, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	return images[start:end], total, nil
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || 
		 len(substr) == 0 ||
		 findIgnoreCase(s, substr) >= 0)
}

// findIgnoreCase finds substr in s (case-insensitive).
func findIgnoreCase(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(s) < len(substr) {
		return -1
	}
	
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			pc := substr[j]
			// Convert to lowercase for comparison
			if sc >= 'A' && sc <= 'Z' {
				sc += 'a' - 'A'
			}
			if pc >= 'A' && pc <= 'Z' {
				pc += 'a' - 'A'
			}
			if sc != pc {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// GetBlobPath returns the blob storage path (for external use).
func (s *Storage) GetBlobPath() string {
	return s.blobPath
}

// GetMetaPath returns the metadata storage path (for external use).
func (s *Storage) GetMetaPath() string {
	return s.metaPath
}
