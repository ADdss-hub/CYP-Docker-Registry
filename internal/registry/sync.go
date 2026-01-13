// Package registry provides container image registry functionality.
package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SyncStatus represents the status of a sync operation.
type SyncStatus string

const (
	SyncStatusPending   SyncStatus = "pending"
	SyncStatusRunning   SyncStatus = "running"
	SyncStatusCompleted SyncStatus = "completed"
	SyncStatusFailed    SyncStatus = "failed"
)

// SyncRecord represents a sync operation history record.
type SyncRecord struct {
	ID            string     `json:"id"`
	ImageName     string     `json:"image_name"`
	ImageTag      string     `json:"image_tag"`
	SourceDigest  string     `json:"source_digest"`
	TargetRegistry string    `json:"target_registry"`
	TargetImage   string     `json:"target_image"`
	TargetTag     string     `json:"target_tag"`
	Status        SyncStatus `json:"status"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	BytesSynced   int64      `json:"bytes_synced"`
}

// SyncHistory represents the sync history storage structure.
type SyncHistory struct {
	Records []*SyncRecord `json:"records"`
}

// SyncService handles image synchronization to public registries.
type SyncService struct {
	storage           *Storage
	credentialManager *CredentialManager
	historyPath       string
	httpClient        *http.Client
	mu                sync.RWMutex
}

// NewSyncService creates a new SyncService.
func NewSyncService(storage *Storage, credentialManager *CredentialManager, historyPath string) (*SyncService, error) {
	if err := os.MkdirAll(historyPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sync history directory: %w", err)
	}

	return &SyncService{
		storage:           storage,
		credentialManager: credentialManager,
		historyPath:       historyPath,
		httpClient: &http.Client{
			Timeout: 30 * time.Minute, // Long timeout for large images
		},
	}, nil
}


// getHistoryFilePath returns the path to the sync history file.
func (ss *SyncService) getHistoryFilePath() string {
	return filepath.Join(ss.historyPath, "sync_history.json")
}

// loadHistory loads sync history from disk.
func (ss *SyncService) loadHistory() (*SyncHistory, error) {
	filePath := ss.getHistoryFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &SyncHistory{
				Records: make([]*SyncRecord, 0),
			}, nil
		}
		return nil, fmt.Errorf("failed to read sync history: %w", err)
	}

	var history SyncHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("failed to parse sync history: %w", err)
	}

	if history.Records == nil {
		history.Records = make([]*SyncRecord, 0)
	}

	return &history, nil
}

// saveHistory saves sync history to disk.
func (ss *SyncService) saveHistory(history *SyncHistory) error {
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sync history: %w", err)
	}

	filePath := ss.getHistoryFilePath()
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write sync history: %w", err)
	}

	return nil
}

// generateSyncID generates a unique ID for a sync operation.
func generateSyncID() string {
	return fmt.Sprintf("sync-%d", time.Now().UnixNano())
}

// addRecord adds a new sync record to history.
func (ss *SyncService) addRecord(record *SyncRecord) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	history, err := ss.loadHistory()
	if err != nil {
		return err
	}

	history.Records = append(history.Records, record)

	// Keep only last 1000 records
	if len(history.Records) > 1000 {
		history.Records = history.Records[len(history.Records)-1000:]
	}

	return ss.saveHistory(history)
}

// updateRecord updates an existing sync record.
func (ss *SyncService) updateRecord(record *SyncRecord) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	history, err := ss.loadHistory()
	if err != nil {
		return err
	}

	for i, r := range history.Records {
		if r.ID == record.ID {
			history.Records[i] = record
			return ss.saveHistory(history)
		}
	}

	return fmt.Errorf("sync record not found: %s", record.ID)
}


// SyncRequest represents a request to sync an image to a public registry.
type SyncRequest struct {
	ImageName      string `json:"image_name"`
	ImageTag       string `json:"image_tag"`
	TargetRegistry string `json:"target_registry"`
	TargetImage    string `json:"target_image,omitempty"` // Optional, defaults to ImageName
	TargetTag      string `json:"target_tag,omitempty"`   // Optional, defaults to ImageTag
}

// SyncImage synchronizes a local image to a public registry.
func (ss *SyncService) SyncImage(req *SyncRequest) (*SyncRecord, error) {
	// Validate request
	if req.ImageName == "" || req.ImageTag == "" || req.TargetRegistry == "" {
		return nil, fmt.Errorf("image_name, image_tag, and target_registry are required")
	}

	// Set defaults
	if req.TargetImage == "" {
		req.TargetImage = req.ImageName
	}
	if req.TargetTag == "" {
		req.TargetTag = req.ImageTag
	}

	// Get source image
	manifest, err := ss.storage.GetImage(req.ImageName, req.ImageTag)
	if err != nil {
		return nil, fmt.Errorf("source image not found: %w", err)
	}

	// Get credentials for target registry
	cred, err := ss.credentialManager.GetCredential(req.TargetRegistry)
	if err != nil {
		return nil, fmt.Errorf("credentials not found for registry %s: %w", req.TargetRegistry, err)
	}

	// Create sync record
	record := &SyncRecord{
		ID:             generateSyncID(),
		ImageName:      req.ImageName,
		ImageTag:       req.ImageTag,
		SourceDigest:   manifest.Digest,
		TargetRegistry: req.TargetRegistry,
		TargetImage:    req.TargetImage,
		TargetTag:      req.TargetTag,
		Status:         SyncStatusRunning,
		StartedAt:      time.Now().UTC(),
	}

	if err := ss.addRecord(record); err != nil {
		return nil, fmt.Errorf("failed to create sync record: %w", err)
	}

	// Perform sync in background
	go ss.performSync(record, manifest, cred)

	return record, nil
}

// performSync performs the actual sync operation.
func (ss *SyncService) performSync(record *SyncRecord, manifest *ImageManifest, cred *Credential) {
	var totalBytes int64
	var syncErr error

	defer func() {
		now := time.Now().UTC()
		record.CompletedAt = &now
		record.BytesSynced = totalBytes

		if syncErr != nil {
			record.Status = SyncStatusFailed
			record.ErrorMessage = syncErr.Error()
		} else {
			record.Status = SyncStatusCompleted
		}

		ss.updateRecord(record)
	}()

	// Push each layer to target registry
	for _, layer := range manifest.Layers {
		layerBytes, err := ss.pushLayer(record.TargetRegistry, record.TargetImage, layer.Digest, cred)
		if err != nil {
			syncErr = fmt.Errorf("failed to push layer %s: %w", layer.Digest, err)
			return
		}
		totalBytes += layerBytes
	}

	// Push manifest to target registry
	manifestData, _, err := ss.storage.GetBlob(manifest.Digest)
	if err != nil {
		syncErr = fmt.Errorf("failed to read manifest: %w", err)
		return
	}
	defer manifestData.Close()

	manifestBytes, err := io.ReadAll(manifestData)
	if err != nil {
		syncErr = fmt.Errorf("failed to read manifest data: %w", err)
		return
	}

	if err := ss.pushManifest(record.TargetRegistry, record.TargetImage, record.TargetTag, manifestBytes, cred); err != nil {
		syncErr = fmt.Errorf("failed to push manifest: %w", err)
		return
	}

	totalBytes += int64(len(manifestBytes))
}


// pushLayer pushes a layer to the target registry.
func (ss *SyncService) pushLayer(registryURL, imageName, digest string, cred *Credential) (int64, error) {
	// Check if layer already exists
	exists, err := ss.checkBlobExists(registryURL, imageName, digest, cred)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, nil // Layer already exists, skip
	}

	// Get layer data from local storage
	reader, size, err := ss.storage.GetBlob(digest)
	if err != nil {
		return 0, fmt.Errorf("failed to get local blob: %w", err)
	}
	defer reader.Close()

	// Start upload
	uploadURL, err := ss.startBlobUpload(registryURL, imageName, cred)
	if err != nil {
		return 0, fmt.Errorf("failed to start upload: %w", err)
	}

	// Upload blob
	if err := ss.uploadBlob(uploadURL, digest, reader, size, cred); err != nil {
		return 0, fmt.Errorf("failed to upload blob: %w", err)
	}

	return size, nil
}

// checkBlobExists checks if a blob exists in the target registry.
func (ss *SyncService) checkBlobExists(registryURL, imageName, digest string, cred *Credential) (bool, error) {
	url := fmt.Sprintf("%s/v2/%s/blobs/%s", registryURL, imageName, digest)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, err
	}

	ss.setAuthHeader(req, cred)

	resp, err := ss.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// startBlobUpload initiates a blob upload and returns the upload URL.
func (ss *SyncService) startBlobUpload(registryURL, imageName string, cred *Credential) (string, error) {
	url := fmt.Sprintf("%s/v2/%s/blobs/uploads/", registryURL, imageName)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}

	ss.setAuthHeader(req, cred)

	resp, err := ss.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to start upload: %s - %s", resp.Status, string(body))
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("no upload location returned")
	}

	// Handle relative URLs
	if location[0] == '/' {
		location = registryURL + location
	}

	return location, nil
}

// uploadBlob uploads blob data to the given URL.
func (ss *SyncService) uploadBlob(uploadURL, digest string, data io.Reader, size int64, cred *Credential) error {
	// Add digest query parameter
	if uploadURL[len(uploadURL)-1] == '/' {
		uploadURL = uploadURL[:len(uploadURL)-1]
	}
	if len(uploadURL) > 0 && uploadURL[len(uploadURL)-1] != '?' {
		uploadURL += "?"
	}
	uploadURL += "digest=" + digest

	req, err := http.NewRequest("PUT", uploadURL, data)
	if err != nil {
		return err
	}

	req.ContentLength = size
	req.Header.Set("Content-Type", "application/octet-stream")
	ss.setAuthHeader(req, cred)

	resp, err := ss.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload blob: %s - %s", resp.Status, string(body))
	}

	return nil
}


// pushManifest pushes a manifest to the target registry.
func (ss *SyncService) pushManifest(registryURL, imageName, tag string, manifestData []byte, cred *Credential) error {
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", registryURL, imageName, tag)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(manifestData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
	ss.setAuthHeader(req, cred)

	resp, err := ss.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to push manifest: %s - %s", resp.Status, string(body))
	}

	return nil
}

// setAuthHeader sets the authorization header for registry requests.
func (ss *SyncService) setAuthHeader(req *http.Request, cred *Credential) {
	if cred != nil && cred.Username != "" && cred.Password != "" {
		req.SetBasicAuth(cred.Username, cred.Password)
	}
}

// GetSyncHistory returns sync history with pagination.
func (ss *SyncService) GetSyncHistory(page, pageSize int) ([]*SyncRecord, int, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	history, err := ss.loadHistory()
	if err != nil {
		return nil, 0, err
	}

	total := len(history.Records)

	// Reverse order (newest first)
	records := make([]*SyncRecord, len(history.Records))
	for i, r := range history.Records {
		records[len(history.Records)-1-i] = r
	}

	// Apply pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	start := (page - 1) * pageSize
	if start >= total {
		return []*SyncRecord{}, total, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	return records[start:end], total, nil
}

// GetSyncRecord returns a specific sync record by ID.
func (ss *SyncService) GetSyncRecord(id string) (*SyncRecord, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	history, err := ss.loadHistory()
	if err != nil {
		return nil, err
	}

	for _, r := range history.Records {
		if r.ID == id {
			return r, nil
		}
	}

	return nil, fmt.Errorf("sync record not found: %s", id)
}

// GetSyncHistoryByImage returns sync history for a specific image.
func (ss *SyncService) GetSyncHistoryByImage(imageName, imageTag string) ([]*SyncRecord, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	history, err := ss.loadHistory()
	if err != nil {
		return nil, err
	}

	var records []*SyncRecord
	for _, r := range history.Records {
		if r.ImageName == imageName && (imageTag == "" || r.ImageTag == imageTag) {
			records = append(records, r)
		}
	}

	// Reverse order (newest first)
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}

	return records, nil
}

// RetrySync retries a failed sync operation.
func (ss *SyncService) RetrySync(syncID string) (*SyncRecord, error) {
	record, err := ss.GetSyncRecord(syncID)
	if err != nil {
		return nil, err
	}

	if record.Status != SyncStatusFailed {
		return nil, fmt.Errorf("can only retry failed sync operations")
	}

	// Create new sync request from the failed record
	return ss.SyncImage(&SyncRequest{
		ImageName:      record.ImageName,
		ImageTag:       record.ImageTag,
		TargetRegistry: record.TargetRegistry,
		TargetImage:    record.TargetImage,
		TargetTag:      record.TargetTag,
	})
}
