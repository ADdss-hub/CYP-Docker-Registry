// Package service provides business logic services for the container registry.
package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SBOMService provides Software Bill of Materials management services.
type SBOMService struct {
	storagePath string
	sboms       sync.Map // map[imageRef]*SBOM
	logger      *zap.Logger
	config      *SBOMConfig
}

// SBOMConfig holds SBOM configuration.
type SBOMConfig struct {
	Enabled       bool
	Generator     string // syft, trivy
	Format        string // spdx-json, cyclonedx-json
	AutoGenerate  bool
	GenerateOnPush bool
	StoragePath   string
	VulnScan      bool
	VulnScanner   string // trivy, grype
}

// SBOM represents a Software Bill of Materials.
type SBOM struct {
	ID            int64              `json:"id"`
	ImageRef      string             `json:"image_ref"`
	Digest        string             `json:"digest"`
	Format        string             `json:"format"`
	Generator     string             `json:"generator"`
	GeneratedAt   time.Time          `json:"generated_at"`
	Packages      []SBOMPackage      `json:"packages"`
	Dependencies  []SBOMDependency   `json:"dependencies,omitempty"`
	Vulnerabilities []Vulnerability  `json:"vulnerabilities,omitempty"`
	Metadata      map[string]string  `json:"metadata,omitempty"`
}

// SBOMPackage represents a package in the SBOM.
type SBOMPackage struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Type         string   `json:"type"` // npm, pip, apk, deb, etc.
	License      string   `json:"license,omitempty"`
	PURL         string   `json:"purl,omitempty"`
	CPE          string   `json:"cpe,omitempty"`
	Checksums    []string `json:"checksums,omitempty"`
}

// SBOMDependency represents a dependency relationship.
type SBOMDependency struct {
	Package    string   `json:"package"`
	DependsOn  []string `json:"depends_on"`
}

// Vulnerability represents a security vulnerability.
type Vulnerability struct {
	ID          string   `json:"id"`
	Package     string   `json:"package"`
	Version     string   `json:"version"`
	Severity    string   `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	FixedIn     string   `json:"fixed_in,omitempty"`
	References  []string `json:"references,omitempty"`
}

// GenerateSBOMRequest represents a request to generate SBOM.
type GenerateSBOMRequest struct {
	ImageRef string `json:"image_ref" binding:"required"`
	Format   string `json:"format,omitempty"`
}

// ScanVulnRequest represents a request to scan for vulnerabilities.
type ScanVulnRequest struct {
	ImageRef string `json:"image_ref" binding:"required"`
}

// VulnScanResult represents vulnerability scan results.
type VulnScanResult struct {
	ImageRef        string          `json:"image_ref"`
	ScannedAt       time.Time       `json:"scanned_at"`
	Scanner         string          `json:"scanner"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	Summary         VulnSummary     `json:"summary"`
}

// VulnSummary represents a summary of vulnerabilities.
type VulnSummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Total    int `json:"total"`
}

// NewSBOMService creates a new SBOMService instance.
func NewSBOMService(config *SBOMConfig, logger *zap.Logger) *SBOMService {
	if config == nil {
		config = &SBOMConfig{
			Enabled:   false,
			Generator: "syft",
			Format:    "spdx-json",
		}
	}

	s := &SBOMService{
		storagePath: config.StoragePath,
		logger:      logger,
		config:      config,
	}

	// Ensure storage directory exists
	if config.StoragePath != "" {
		os.MkdirAll(config.StoragePath, 0755)
	}

	return s
}

// GenerateSBOM generates a SBOM for an image.
func (s *SBOMService) GenerateSBOM(req *GenerateSBOMRequest) (*SBOM, error) {
	if !s.config.Enabled {
		return nil, errors.New("SBOM service is disabled")
	}

	format := req.Format
	if format == "" {
		format = s.config.Format
	}

	// In production, this would call syft/trivy to generate actual SBOM
	// For now, create a placeholder SBOM
	sbom := &SBOM{
		ImageRef:    req.ImageRef,
		Format:      format,
		Generator:   s.config.Generator,
		GeneratedAt: time.Now(),
		Packages:    []SBOMPackage{},
		Metadata: map[string]string{
			"tool":    s.config.Generator,
			"version": "1.0.0",
		},
	}

	// Store SBOM
	s.sboms.Store(req.ImageRef, sbom)

	// Persist to disk
	s.persistSBOM(sbom)

	if s.logger != nil {
		s.logger.Info("SBOM generated",
			zap.String("image", req.ImageRef),
			zap.String("format", format),
		)
	}

	return sbom, nil
}

// GetSBOM retrieves a SBOM for an image.
func (s *SBOMService) GetSBOM(imageRef string) (*SBOM, error) {
	sbom, ok := s.sboms.Load(imageRef)
	if !ok {
		// Try to load from disk
		sbom = s.loadSBOM(imageRef)
		if sbom == nil {
			return nil, errors.New("SBOM not found")
		}
	}
	return sbom.(*SBOM), nil
}

// ListSBOMs lists all SBOMs.
func (s *SBOMService) ListSBOMs(page, pageSize int) ([]*SBOM, int, error) {
	var sboms []*SBOM

	s.sboms.Range(func(key, value interface{}) bool {
		sboms = append(sboms, value.(*SBOM))
		return true
	})

	total := len(sboms)

	// Pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		return []*SBOM{}, total, nil
	}
	if end > total {
		end = total
	}

	return sboms[start:end], total, nil
}

// DeleteSBOM deletes a SBOM.
func (s *SBOMService) DeleteSBOM(imageRef string) error {
	s.sboms.Delete(imageRef)

	// Remove from disk
	filename := s.getSBOMFilename(imageRef)
	os.Remove(filename)

	return nil
}

// ScanVulnerabilities scans an image for vulnerabilities.
func (s *SBOMService) ScanVulnerabilities(req *ScanVulnRequest) (*VulnScanResult, error) {
	if !s.config.Enabled || !s.config.VulnScan {
		return nil, errors.New("vulnerability scanning is disabled")
	}

	// In production, this would call trivy/grype to scan
	// For now, return empty results
	result := &VulnScanResult{
		ImageRef:        req.ImageRef,
		ScannedAt:       time.Now(),
		Scanner:         s.config.VulnScanner,
		Vulnerabilities: []Vulnerability{},
		Summary: VulnSummary{
			Critical: 0,
			High:     0,
			Medium:   0,
			Low:      0,
			Total:    0,
		},
	}

	// Update SBOM with vulnerabilities
	if sbom, ok := s.sboms.Load(req.ImageRef); ok {
		sbomData := sbom.(*SBOM)
		sbomData.Vulnerabilities = result.Vulnerabilities
		s.persistSBOM(sbomData)
	}

	if s.logger != nil {
		s.logger.Info("Vulnerability scan completed",
			zap.String("image", req.ImageRef),
			zap.Int("total", result.Summary.Total),
		)
	}

	return result, nil
}

// ExportSBOM exports a SBOM in the specified format.
func (s *SBOMService) ExportSBOM(imageRef, format string) ([]byte, error) {
	sbom, err := s.GetSBOM(imageRef)
	if err != nil {
		return nil, err
	}

	switch format {
	case "spdx-json", "json":
		return json.MarshalIndent(sbom, "", "  ")
	case "cyclonedx-json":
		// Convert to CycloneDX format
		return json.MarshalIndent(s.convertToCycloneDX(sbom), "", "  ")
	default:
		return json.MarshalIndent(sbom, "", "  ")
	}
}

// persistSBOM saves a SBOM to disk.
func (s *SBOMService) persistSBOM(sbom *SBOM) error {
	if s.storagePath == "" {
		return nil
	}

	filename := s.getSBOMFilename(sbom.ImageRef)
	data, err := json.MarshalIndent(sbom, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// loadSBOM loads a SBOM from disk.
func (s *SBOMService) loadSBOM(imageRef string) *SBOM {
	if s.storagePath == "" {
		return nil
	}

	filename := s.getSBOMFilename(imageRef)
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}

	var sbom SBOM
	if err := json.Unmarshal(data, &sbom); err != nil {
		return nil
	}

	// Cache it
	s.sboms.Store(imageRef, &sbom)

	return &sbom
}

// getSBOMFilename returns the filename for a SBOM.
func (s *SBOMService) getSBOMFilename(imageRef string) string {
	// Sanitize image ref for filename
	safe := ""
	for _, c := range imageRef {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			safe += string(c)
		} else {
			safe += "_"
		}
	}
	return filepath.Join(s.storagePath, safe+".sbom.json")
}

// convertToCycloneDX converts SBOM to CycloneDX format.
func (s *SBOMService) convertToCycloneDX(sbom *SBOM) map[string]interface{} {
	components := make([]map[string]interface{}, len(sbom.Packages))
	for i, pkg := range sbom.Packages {
		components[i] = map[string]interface{}{
			"type":    "library",
			"name":    pkg.Name,
			"version": pkg.Version,
			"purl":    pkg.PURL,
		}
	}

	return map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.4",
		"version":     1,
		"metadata": map[string]interface{}{
			"timestamp": sbom.GeneratedAt.Format(time.RFC3339),
			"tools": []map[string]string{
				{"name": sbom.Generator},
			},
		},
		"components": components,
	}
}
