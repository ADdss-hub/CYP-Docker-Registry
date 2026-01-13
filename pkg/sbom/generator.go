// Package sbom provides Software Bill of Materials generation utilities.
package sbom

import (
	"encoding/json"
	"time"
)

// Generator provides SBOM generation capabilities.
type Generator struct {
	format    string
	generator string
}

// GeneratorConfig holds generator configuration.
type GeneratorConfig struct {
	Format    string // spdx-json, cyclonedx-json
	Generator string // syft, trivy
}

// SBOM represents a Software Bill of Materials.
type SBOM struct {
	Format      string            `json:"format"`
	Version     string            `json:"version"`
	Generator   string            `json:"generator"`
	GeneratedAt time.Time         `json:"generated_at"`
	Image       ImageInfo         `json:"image"`
	Packages    []Package         `json:"packages"`
	Files       []File            `json:"files,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ImageInfo represents image information.
type ImageInfo struct {
	Name   string `json:"name"`
	Tag    string `json:"tag"`
	Digest string `json:"digest"`
	OS     string `json:"os"`
	Arch   string `json:"arch"`
}

// Package represents a software package.
type Package struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Type         string   `json:"type"` // npm, pip, apk, deb, rpm, etc.
	License      string   `json:"license,omitempty"`
	PURL         string   `json:"purl,omitempty"`
	CPE          string   `json:"cpe,omitempty"`
	Supplier     string   `json:"supplier,omitempty"`
	Description  string   `json:"description,omitempty"`
	Homepage     string   `json:"homepage,omitempty"`
	Checksums    []string `json:"checksums,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// File represents a file in the image.
type File struct {
	Path     string `json:"path"`
	Checksum string `json:"checksum"`
	Size     int64  `json:"size"`
	Type     string `json:"type"`
}

// NewGenerator creates a new Generator instance.
func NewGenerator(config *GeneratorConfig) *Generator {
	if config == nil {
		config = &GeneratorConfig{
			Format:    "spdx-json",
			Generator: "cyp-docker-registry",
		}
	}

	return &Generator{
		format:    config.Format,
		generator: config.Generator,
	}
}

// Generate generates a SBOM for an image.
func (g *Generator) Generate(imageRef, digest string) (*SBOM, error) {
	sbom := &SBOM{
		Format:      g.format,
		Version:     "1.0",
		Generator:   g.generator,
		GeneratedAt: time.Now(),
		Image: ImageInfo{
			Name:   imageRef,
			Digest: digest,
		},
		Packages: []Package{},
		Metadata: map[string]string{
			"tool":    g.generator,
			"version": "1.0.0",
		},
	}

	// In production, this would analyze the actual image layers
	// For now, return a placeholder SBOM

	return sbom, nil
}

// Export exports the SBOM in the specified format.
func (g *Generator) Export(sbom *SBOM, format string) ([]byte, error) {
	if format == "" {
		format = g.format
	}

	switch format {
	case "spdx-json":
		return g.exportSPDX(sbom)
	case "cyclonedx-json":
		return g.exportCycloneDX(sbom)
	default:
		return json.MarshalIndent(sbom, "", "  ")
	}
}

// exportSPDX exports SBOM in SPDX format.
func (g *Generator) exportSPDX(sbom *SBOM) ([]byte, error) {
	spdx := map[string]interface{}{
		"spdxVersion":       "SPDX-2.3",
		"dataLicense":       "CC0-1.0",
		"SPDXID":            "SPDXRef-DOCUMENT",
		"name":              sbom.Image.Name,
		"documentNamespace": "https://cyp-docker-registry.local/sbom/" + sbom.Image.Digest,
		"creationInfo": map[string]interface{}{
			"created": sbom.GeneratedAt.Format(time.RFC3339),
			"creators": []string{
				"Tool: " + sbom.Generator,
			},
		},
		"packages": g.convertToSPDXPackages(sbom.Packages),
	}

	return json.MarshalIndent(spdx, "", "  ")
}

// exportCycloneDX exports SBOM in CycloneDX format.
func (g *Generator) exportCycloneDX(sbom *SBOM) ([]byte, error) {
	cdx := map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.4",
		"version":     1,
		"metadata": map[string]interface{}{
			"timestamp": sbom.GeneratedAt.Format(time.RFC3339),
			"tools": []map[string]string{
				{"name": sbom.Generator, "version": "1.0.0"},
			},
			"component": map[string]interface{}{
				"type":    "container",
				"name":    sbom.Image.Name,
				"version": sbom.Image.Tag,
			},
		},
		"components": g.convertToCycloneDXComponents(sbom.Packages),
	}

	return json.MarshalIndent(cdx, "", "  ")
}

// convertToSPDXPackages converts packages to SPDX format.
func (g *Generator) convertToSPDXPackages(packages []Package) []map[string]interface{} {
	result := make([]map[string]interface{}, len(packages))
	for i, pkg := range packages {
		result[i] = map[string]interface{}{
			"SPDXID":           "SPDXRef-Package-" + pkg.Name,
			"name":             pkg.Name,
			"versionInfo":      pkg.Version,
			"downloadLocation": "NOASSERTION",
			"filesAnalyzed":    false,
		}
		if pkg.License != "" {
			result[i]["licenseConcluded"] = pkg.License
		}
		if pkg.PURL != "" {
			result[i]["externalRefs"] = []map[string]string{
				{
					"referenceCategory": "PACKAGE-MANAGER",
					"referenceType":     "purl",
					"referenceLocator":  pkg.PURL,
				},
			}
		}
	}
	return result
}

// convertToCycloneDXComponents converts packages to CycloneDX format.
func (g *Generator) convertToCycloneDXComponents(packages []Package) []map[string]interface{} {
	result := make([]map[string]interface{}, len(packages))
	for i, pkg := range packages {
		result[i] = map[string]interface{}{
			"type":    "library",
			"name":    pkg.Name,
			"version": pkg.Version,
		}
		if pkg.PURL != "" {
			result[i]["purl"] = pkg.PURL
		}
		if pkg.License != "" {
			result[i]["licenses"] = []map[string]interface{}{
				{"license": map[string]string{"id": pkg.License}},
			}
		}
	}
	return result
}

// AddPackage adds a package to the SBOM.
func (s *SBOM) AddPackage(pkg Package) {
	s.Packages = append(s.Packages, pkg)
}

// AddFile adds a file to the SBOM.
func (s *SBOM) AddFile(file File) {
	s.Files = append(s.Files, file)
}

// GetPackageCount returns the number of packages.
func (s *SBOM) GetPackageCount() int {
	return len(s.Packages)
}

// GetFileCount returns the number of files.
func (s *SBOM) GetFileCount() int {
	return len(s.Files)
}
