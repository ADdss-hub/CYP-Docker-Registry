// Package registry provides container image registry functionality.
package registry

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// ImageList represents a paginated list of images.
type ImageList struct {
	Images     []*ImageManifest `json:"images"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// Service provides registry operations.
type Service struct {
	storage *Storage
}

// NewService creates a new registry service.
func NewService(storage *Storage) *Service {
	return &Service{
		storage: storage,
	}
}

// PushManifest stores an image manifest.
func (s *Service) PushManifest(name, tag string, manifestData []byte) (*ImageManifest, error) {
	// Parse manifest to extract layer information
	var rawManifest struct {
		SchemaVersion int    `json:"schemaVersion"`
		MediaType     string `json:"mediaType"`
		Config        struct {
			MediaType string `json:"mediaType"`
			Size      int64  `json:"size"`
			Digest    string `json:"digest"`
		} `json:"config"`
		Layers []struct {
			MediaType string `json:"mediaType"`
			Size      int64  `json:"size"`
			Digest    string `json:"digest"`
		} `json:"layers"`
	}

	if err := json.Unmarshal(manifestData, &rawManifest); err != nil {
		return nil, fmt.Errorf("invalid manifest format: %w", err)
	}

	// Calculate manifest digest
	hash := sha256.Sum256(manifestData)
	digest := "sha256:" + hex.EncodeToString(hash[:])

	// Calculate total size
	var totalSize int64
	var layers []Layer
	for _, l := range rawManifest.Layers {
		totalSize += l.Size
		layers = append(layers, Layer{
			Digest:    l.Digest,
			Size:      l.Size,
			MediaType: l.MediaType,
		})
	}

	// Store manifest as blob
	if _, err := s.storage.SaveBlobWithDigest(digest, bytes.NewReader(manifestData)); err != nil {
		return nil, fmt.Errorf("failed to store manifest: %w", err)
	}

	// Create image manifest
	manifest := &ImageManifest{
		Name:      name,
		Tag:       tag,
		Digest:    digest,
		Size:      totalSize,
		CreatedAt: time.Now().UTC(),
		Layers:    layers,
	}

	// Save metadata
	if err := s.storage.SaveImage(manifest); err != nil {
		return nil, fmt.Errorf("failed to save image metadata: %w", err)
	}

	return manifest, nil
}

// PullManifest retrieves an image manifest.
func (s *Service) PullManifest(name, tag string) ([]byte, *ImageManifest, error) {
	// Get image metadata
	manifest, err := s.storage.GetImage(name, tag)
	if err != nil {
		return nil, nil, err
	}

	// Get manifest blob
	reader, _, err := s.storage.GetBlob(manifest.Digest)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read manifest: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read manifest data: %w", err)
	}

	return data, manifest, nil
}

// DeleteImage removes an image and its associated data.
func (s *Service) DeleteImage(name, tag string) error {
	// Get image metadata first
	manifest, err := s.storage.GetImage(name, tag)
	if err != nil {
		return err
	}

	// Delete manifest blob
	if err := s.storage.DeleteBlob(manifest.Digest); err != nil {
		// Log but don't fail - blob might be shared
	}

	// Delete layer blobs (only if not shared by other images)
	// For simplicity, we'll skip layer deletion here
	// A proper implementation would track blob references

	// Delete metadata
	return s.storage.DeleteImage(name, tag)
}

// ListImages returns a paginated list of images.
func (s *Service) ListImages(page, pageSize int) (*ImageList, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	images, total, err := s.storage.ListImages(page, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return &ImageList{
		Images:     images,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// SearchImages searches images by keyword.
func (s *Service) SearchImages(keyword string, page, pageSize int) (*ImageList, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	images, total, err := s.storage.SearchImages(keyword, page, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return &ImageList{
		Images:     images,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}


// PushBlob stores a blob and returns its digest.
func (s *Service) PushBlob(data io.Reader) (string, int64, error) {
	return s.storage.SaveBlob(data)
}

// PushBlobWithDigest stores a blob with a known digest.
func (s *Service) PushBlobWithDigest(digest string, data io.Reader) (int64, error) {
	return s.storage.SaveBlobWithDigest(digest, data)
}

// PullBlob retrieves a blob by digest.
func (s *Service) PullBlob(digest string) (io.ReadCloser, int64, error) {
	return s.storage.GetBlob(digest)
}

// BlobExists checks if a blob exists.
func (s *Service) BlobExists(digest string) bool {
	return s.storage.BlobExists(digest)
}

// DeleteBlob removes a blob by digest.
func (s *Service) DeleteBlob(digest string) error {
	return s.storage.DeleteBlob(digest)
}

// GetImage retrieves image metadata.
func (s *Service) GetImage(name, tag string) (*ImageManifest, error) {
	return s.storage.GetImage(name, tag)
}

// GetStorage returns the underlying storage (for advanced operations).
func (s *Service) GetStorage() *Storage {
	return s.storage
}
