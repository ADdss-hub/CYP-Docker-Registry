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
	// First, try to detect manifest type
	var baseManifest struct {
		SchemaVersion int    `json:"schemaVersion"`
		MediaType     string `json:"mediaType"`
	}

	if err := json.Unmarshal(manifestData, &baseManifest); err != nil {
		return nil, fmt.Errorf("invalid manifest format: %w", err)
	}

	// Calculate manifest digest
	hash := sha256.Sum256(manifestData)
	digest := "sha256:" + hex.EncodeToString(hash[:])

	var totalSize int64
	var layers []Layer

	// Check if this is a manifest list/index (multi-arch image)
	if baseManifest.MediaType == "application/vnd.docker.distribution.manifest.list.v2+json" ||
		baseManifest.MediaType == "application/vnd.oci.image.index.v1+json" {
		// Parse as manifest list
		var manifestList struct {
			Manifests []struct {
				MediaType string `json:"mediaType"`
				Size      int64  `json:"size"`
				Digest    string `json:"digest"`
				Platform  struct {
					Architecture string `json:"architecture"`
					OS           string `json:"os"`
				} `json:"platform"`
			} `json:"manifests"`
		}

		if err := json.Unmarshal(manifestData, &manifestList); err != nil {
			return nil, fmt.Errorf("invalid manifest list format: %w", err)
		}

		// For manifest list, we need to resolve the actual manifest for the target platform
		// Try to find linux/amd64 manifest first, then fall back to first available
		var targetDigest string
		var targetSize int64
		for _, m := range manifestList.Manifests {
			totalSize += m.Size
			if m.Platform.OS == "linux" && m.Platform.Architecture == "amd64" {
				targetDigest = m.Digest
				targetSize = m.Size
			}
			// Add each platform manifest as a "layer" for display purposes
			layers = append(layers, Layer{
				Digest:    m.Digest,
				Size:      m.Size,
				MediaType: m.MediaType,
			})
		}

		// If we found a target manifest, try to resolve its layers
		if targetDigest != "" {
			resolvedLayers, resolvedSize := s.resolveManifestLayers(targetDigest)
			if len(resolvedLayers) > 0 {
				layers = resolvedLayers
				totalSize = resolvedSize
			}
		} else if len(manifestList.Manifests) > 0 {
			// Fall back to first manifest
			targetDigest = manifestList.Manifests[0].Digest
			targetSize = manifestList.Manifests[0].Size
			resolvedLayers, resolvedSize := s.resolveManifestLayers(targetDigest)
			if len(resolvedLayers) > 0 {
				layers = resolvedLayers
				totalSize = resolvedSize
			} else {
				totalSize = targetSize
			}
		}
	} else {
		// Parse as regular manifest (v2 or OCI)
		var rawManifest struct {
			Config struct {
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

		// Calculate total size from layers
		for _, l := range rawManifest.Layers {
			totalSize += l.Size
			layers = append(layers, Layer{
				Digest:    l.Digest,
				Size:      l.Size,
				MediaType: l.MediaType,
			})
		}
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

// resolveManifestLayers tries to resolve layers from a manifest digest
func (s *Service) resolveManifestLayers(digest string) ([]Layer, int64) {
	reader, _, err := s.storage.GetBlob(digest)
	if err != nil {
		return nil, 0
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, 0
	}

	var manifest struct {
		Layers []struct {
			MediaType string `json:"mediaType"`
			Size      int64  `json:"size"`
			Digest    string `json:"digest"`
		} `json:"layers"`
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, 0
	}

	var layers []Layer
	var totalSize int64
	for _, l := range manifest.Layers {
		totalSize += l.Size
		layers = append(layers, Layer{
			Digest:    l.Digest,
			Size:      l.Size,
			MediaType: l.MediaType,
		})
	}

	return layers, totalSize
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
