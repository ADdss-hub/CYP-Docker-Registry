// Package service provides business logic services for the container registry.
package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SignatureService provides image signature management services.
type SignatureService struct {
	keyPath    string
	signatures sync.Map // map[imageRef]*SignatureInfo
	logger     *zap.Logger
	config     *SignatureConfig
}

// SignatureConfig holds signature configuration.
type SignatureConfig struct {
	Enabled          bool
	Mode             string // enforce, warn, disabled
	AutoSign         bool
	SignOnPush       bool
	RequireSignature bool
	KeyPath          string
	TrustedKeys      []string
}

// SignatureInfo represents signature information for an image.
type SignatureInfo struct {
	ImageRef      string            `json:"image_ref"`
	Digest        string            `json:"digest"`
	Signature     string            `json:"signature"`
	SignedBy      string            `json:"signed_by"`
	SignedAt      time.Time         `json:"signed_at"`
	KeyID         string            `json:"key_id"`
	Verified      bool              `json:"verified"`
	Attestations  []string          `json:"attestations,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// SignRequest represents a request to sign an image.
type SignRequest struct {
	ImageRef string `json:"image_ref" binding:"required"`
	KeyID    string `json:"key_id,omitempty"`
}

// VerifyRequest represents a request to verify an image signature.
type VerifyRequest struct {
	ImageRef string `json:"image_ref" binding:"required"`
}

// VerifyResult represents the result of signature verification.
type VerifyResult struct {
	ImageRef  string         `json:"image_ref"`
	Verified  bool           `json:"verified"`
	Signature *SignatureInfo `json:"signature,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// NewSignatureService creates a new SignatureService instance.
func NewSignatureService(config *SignatureConfig, logger *zap.Logger) *SignatureService {
	if config == nil {
		config = &SignatureConfig{
			Enabled: false,
			Mode:    "warn",
		}
	}

	s := &SignatureService{
		keyPath: config.KeyPath,
		logger:  logger,
		config:  config,
	}

	// Ensure key directory exists
	if config.KeyPath != "" {
		os.MkdirAll(config.KeyPath, 0700)
	}

	return s
}

// SignImage signs an image.
func (s *SignatureService) SignImage(req *SignRequest, userID int64, username string) (*SignatureInfo, error) {
	if !s.config.Enabled {
		return nil, errors.New("signature service is disabled")
	}

	// Generate signature
	digest := s.calculateDigest(req.ImageRef)
	signature := s.generateSignature(digest, req.KeyID)

	info := &SignatureInfo{
		ImageRef:  req.ImageRef,
		Digest:    digest,
		Signature: signature,
		SignedBy:  username,
		SignedAt:  time.Now(),
		KeyID:     req.KeyID,
		Verified:  true,
		Metadata: map[string]string{
			"user_id": string(rune(userID)),
		},
	}

	// Store signature
	s.signatures.Store(req.ImageRef, info)

	// Persist to disk
	s.persistSignature(info)

	if s.logger != nil {
		s.logger.Info("Image signed",
			zap.String("image", req.ImageRef),
			zap.String("signed_by", username),
		)
	}

	return info, nil
}

// VerifyImage verifies an image signature.
func (s *SignatureService) VerifyImage(req *VerifyRequest) (*VerifyResult, error) {
	result := &VerifyResult{
		ImageRef: req.ImageRef,
		Verified: false,
	}

	if !s.config.Enabled {
		result.Error = "signature service is disabled"
		return result, nil
	}

	// Look up signature
	info, ok := s.signatures.Load(req.ImageRef)
	if !ok {
		// Try to load from disk
		info = s.loadSignature(req.ImageRef)
		if info == nil {
			result.Error = "no signature found"
			return result, nil
		}
	}

	sigInfo := info.(*SignatureInfo)

	// Verify signature
	expectedDigest := s.calculateDigest(req.ImageRef)
	if sigInfo.Digest != expectedDigest {
		result.Error = "digest mismatch"
		return result, nil
	}

	// Verify signature value
	if !s.verifySignature(sigInfo.Digest, sigInfo.Signature, sigInfo.KeyID) {
		result.Error = "invalid signature"
		return result, nil
	}

	result.Verified = true
	result.Signature = sigInfo

	return result, nil
}

// GetSignature retrieves signature information for an image.
func (s *SignatureService) GetSignature(imageRef string) (*SignatureInfo, error) {
	info, ok := s.signatures.Load(imageRef)
	if !ok {
		info = s.loadSignature(imageRef)
		if info == nil {
			return nil, errors.New("signature not found")
		}
	}
	return info.(*SignatureInfo), nil
}

// ListSignatures lists all signatures.
func (s *SignatureService) ListSignatures(page, pageSize int) ([]*SignatureInfo, int, error) {
	var signatures []*SignatureInfo

	s.signatures.Range(func(key, value interface{}) bool {
		signatures = append(signatures, value.(*SignatureInfo))
		return true
	})

	total := len(signatures)

	// Pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		return []*SignatureInfo{}, total, nil
	}
	if end > total {
		end = total
	}

	return signatures[start:end], total, nil
}

// DeleteSignature deletes a signature.
func (s *SignatureService) DeleteSignature(imageRef string) error {
	s.signatures.Delete(imageRef)

	// Remove from disk
	filename := s.getSignatureFilename(imageRef)
	os.Remove(filename)

	return nil
}

// IsSignatureRequired checks if signature is required for an image.
func (s *SignatureService) IsSignatureRequired(imageRef string) bool {
	if !s.config.Enabled {
		return false
	}
	return s.config.RequireSignature && s.config.Mode == "enforce"
}

// calculateDigest calculates the digest of an image reference.
func (s *SignatureService) calculateDigest(imageRef string) string {
	hash := sha256.Sum256([]byte(imageRef))
	return "sha256:" + hex.EncodeToString(hash[:])
}

// generateSignature generates a signature for a digest.
func (s *SignatureService) generateSignature(digest, keyID string) string {
	// Simplified signature generation
	// In production, use proper cryptographic signing (cosign, etc.)
	data := digest + ":" + keyID + ":" + time.Now().String()
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// verifySignature verifies a signature.
func (s *SignatureService) verifySignature(digest, signature, keyID string) bool {
	// Simplified verification
	// In production, use proper cryptographic verification
	return len(signature) == 64 // SHA256 hex length
}

// persistSignature saves a signature to disk.
func (s *SignatureService) persistSignature(info *SignatureInfo) error {
	if s.keyPath == "" {
		return nil
	}

	filename := s.getSignatureFilename(info.ImageRef)
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// loadSignature loads a signature from disk.
func (s *SignatureService) loadSignature(imageRef string) *SignatureInfo {
	if s.keyPath == "" {
		return nil
	}

	filename := s.getSignatureFilename(imageRef)
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}

	var info SignatureInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil
	}

	// Cache it
	s.signatures.Store(imageRef, &info)

	return &info
}

// getSignatureFilename returns the filename for a signature.
func (s *SignatureService) getSignatureFilename(imageRef string) string {
	hash := sha256.Sum256([]byte(imageRef))
	filename := hex.EncodeToString(hash[:8]) + ".sig.json"
	return filepath.Join(s.keyPath, filename)
}

// AddAttestation adds an attestation to a signature.
func (s *SignatureService) AddAttestation(imageRef, attestationType string) error {
	info, ok := s.signatures.Load(imageRef)
	if !ok {
		return errors.New("signature not found")
	}

	sigInfo := info.(*SignatureInfo)
	sigInfo.Attestations = append(sigInfo.Attestations, attestationType)

	s.persistSignature(sigInfo)

	return nil
}
