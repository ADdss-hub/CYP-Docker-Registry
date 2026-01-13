// Package sbom provides Software Bill of Materials and vulnerability scanning utilities.
package sbom

import (
	"time"
)

// Scanner provides vulnerability scanning capabilities.
type Scanner struct {
	scanner string
	dbPath  string
}

// ScannerConfig holds scanner configuration.
type ScannerConfig struct {
	Scanner string // trivy, grype
	DBPath  string
}

// Vulnerability represents a security vulnerability.
type Vulnerability struct {
	ID          string   `json:"id"`
	Package     string   `json:"package"`
	Version     string   `json:"version"`
	Severity    string   `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW, UNKNOWN
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	FixedIn     string   `json:"fixed_in,omitempty"`
	CVSS        float64  `json:"cvss,omitempty"`
	CVSSVector  string   `json:"cvss_vector,omitempty"`
	References  []string `json:"references,omitempty"`
	PublishedAt string   `json:"published_at,omitempty"`
}

// ScanResult represents vulnerability scan results.
type ScanResult struct {
	ImageRef        string          `json:"image_ref"`
	Digest          string          `json:"digest"`
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
	Unknown  int `json:"unknown"`
	Total    int `json:"total"`
}

// NewScanner creates a new Scanner instance.
func NewScanner(config *ScannerConfig) *Scanner {
	if config == nil {
		config = &ScannerConfig{
			Scanner: "trivy",
		}
	}

	return &Scanner{
		scanner: config.Scanner,
		dbPath:  config.DBPath,
	}
}

// Scan scans an image for vulnerabilities.
func (s *Scanner) Scan(imageRef, digest string) (*ScanResult, error) {
	result := &ScanResult{
		ImageRef:        imageRef,
		Digest:          digest,
		ScannedAt:       time.Now(),
		Scanner:         s.scanner,
		Vulnerabilities: []Vulnerability{},
		Summary:         VulnSummary{},
	}

	// In production, this would call trivy/grype to scan
	// For now, return empty results

	return result, nil
}

// ScanSBOM scans a SBOM for vulnerabilities.
func (s *Scanner) ScanSBOM(sbom *SBOM) (*ScanResult, error) {
	result := &ScanResult{
		ImageRef:        sbom.Image.Name,
		Digest:          sbom.Image.Digest,
		ScannedAt:       time.Now(),
		Scanner:         s.scanner,
		Vulnerabilities: []Vulnerability{},
		Summary:         VulnSummary{},
	}

	// Scan each package
	for _, pkg := range sbom.Packages {
		vulns := s.scanPackage(pkg)
		result.Vulnerabilities = append(result.Vulnerabilities, vulns...)
	}

	// Calculate summary
	result.Summary = s.calculateSummary(result.Vulnerabilities)

	return result, nil
}

// scanPackage scans a single package for vulnerabilities.
func (s *Scanner) scanPackage(pkg Package) []Vulnerability {
	// In production, this would query a vulnerability database
	// For now, return empty results
	return []Vulnerability{}
}

// calculateSummary calculates the vulnerability summary.
func (s *Scanner) calculateSummary(vulns []Vulnerability) VulnSummary {
	summary := VulnSummary{}

	for _, v := range vulns {
		switch v.Severity {
		case "CRITICAL":
			summary.Critical++
		case "HIGH":
			summary.High++
		case "MEDIUM":
			summary.Medium++
		case "LOW":
			summary.Low++
		default:
			summary.Unknown++
		}
	}

	summary.Total = len(vulns)
	return summary
}

// FilterBySeverity filters vulnerabilities by severity.
func (r *ScanResult) FilterBySeverity(minSeverity string) []Vulnerability {
	severityOrder := map[string]int{
		"CRITICAL": 4,
		"HIGH":     3,
		"MEDIUM":   2,
		"LOW":      1,
		"UNKNOWN":  0,
	}

	minLevel := severityOrder[minSeverity]
	var filtered []Vulnerability

	for _, v := range r.Vulnerabilities {
		if severityOrder[v.Severity] >= minLevel {
			filtered = append(filtered, v)
		}
	}

	return filtered
}

// HasCritical returns true if there are critical vulnerabilities.
func (r *ScanResult) HasCritical() bool {
	return r.Summary.Critical > 0
}

// HasHigh returns true if there are high severity vulnerabilities.
func (r *ScanResult) HasHigh() bool {
	return r.Summary.High > 0
}

// ShouldBlock returns true if the scan results should block deployment.
func (r *ScanResult) ShouldBlock(blockOnCritical, blockOnHigh bool) bool {
	if blockOnCritical && r.HasCritical() {
		return true
	}
	if blockOnHigh && r.HasHigh() {
		return true
	}
	return false
}

// GetVulnerabilityByID returns a vulnerability by ID.
func (r *ScanResult) GetVulnerabilityByID(id string) *Vulnerability {
	for _, v := range r.Vulnerabilities {
		if v.ID == id {
			return &v
		}
	}
	return nil
}

// GetVulnerabilitiesByPackage returns vulnerabilities for a specific package.
func (r *ScanResult) GetVulnerabilitiesByPackage(packageName string) []Vulnerability {
	var result []Vulnerability
	for _, v := range r.Vulnerabilities {
		if v.Package == packageName {
			result = append(result, v)
		}
	}
	return result
}
