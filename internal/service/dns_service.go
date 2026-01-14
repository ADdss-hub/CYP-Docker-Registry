// Package service provides business logic services for CYP-Docker-Registry.
package service

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"go.uber.org/zap"
)

// DNSService provides DNS resolution services.
type DNSService struct {
	logger   *zap.Logger
	resolver *net.Resolver
	timeout  time.Duration
}

// DNSRecord represents a DNS record.
type DNSRecord struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl,omitempty"`
}

// DNSResolveResult represents the result of a DNS resolution.
type DNSResolveResult struct {
	Domain    string       `json:"domain"`
	Records   []*DNSRecord `json:"records"`
	ResolveAt time.Time    `json:"resolve_at"`
	Duration  int64        `json:"duration_ms"`
}

// NewDNSService creates a new DNSService instance.
func NewDNSService(logger *zap.Logger) *DNSService {
	return &DNSService{
		logger: logger,
		resolver: &net.Resolver{
			PreferGo: true,
		},
		timeout: 10 * time.Second,
	}
}

// Resolve resolves a domain name and returns all available records.
func (s *DNSService) Resolve(domain string) (*DNSResolveResult, error) {
	if domain == "" {
		return nil, errors.New("域名不能为空")
	}

	// Clean domain
	domain = strings.TrimSpace(domain)
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.Split(domain, "/")[0]
	domain = strings.Split(domain, ":")[0]

	if !isValidDomain(domain) {
		return nil, errors.New("无效的域名格式")
	}

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	result := &DNSResolveResult{
		Domain:    domain,
		Records:   make([]*DNSRecord, 0),
		ResolveAt: startTime,
	}

	// Resolve A records (IPv4)
	ips, err := s.resolver.LookupIP(ctx, "ip4", domain)
	if err == nil {
		for _, ip := range ips {
			result.Records = append(result.Records, &DNSRecord{
				Type:  "A",
				Value: ip.String(),
			})
		}
	}

	// Resolve AAAA records (IPv6)
	ips6, err := s.resolver.LookupIP(ctx, "ip6", domain)
	if err == nil {
		for _, ip := range ips6 {
			result.Records = append(result.Records, &DNSRecord{
				Type:  "AAAA",
				Value: ip.String(),
			})
		}
	}

	// Resolve CNAME records
	cname, err := s.resolver.LookupCNAME(ctx, domain)
	if err == nil && cname != "" && cname != domain+"." {
		result.Records = append(result.Records, &DNSRecord{
			Type:  "CNAME",
			Value: strings.TrimSuffix(cname, "."),
		})
	}

	// Resolve MX records
	mxRecords, err := s.resolver.LookupMX(ctx, domain)
	if err == nil {
		for _, mx := range mxRecords {
			result.Records = append(result.Records, &DNSRecord{
				Type:  "MX",
				Value: strings.TrimSuffix(mx.Host, "."),
				TTL:   int(mx.Pref),
			})
		}
	}

	// Resolve TXT records
	txtRecords, err := s.resolver.LookupTXT(ctx, domain)
	if err == nil {
		for _, txt := range txtRecords {
			result.Records = append(result.Records, &DNSRecord{
				Type:  "TXT",
				Value: txt,
			})
		}
	}

	// Resolve NS records
	nsRecords, err := s.resolver.LookupNS(ctx, domain)
	if err == nil {
		for _, ns := range nsRecords {
			result.Records = append(result.Records, &DNSRecord{
				Type:  "NS",
				Value: strings.TrimSuffix(ns.Host, "."),
			})
		}
	}

	result.Duration = time.Since(startTime).Milliseconds()

	if len(result.Records) == 0 {
		return nil, errors.New("无法解析该域名")
	}

	s.logger.Info("DNS解析完成",
		zap.String("domain", domain),
		zap.Int("records", len(result.Records)),
		zap.Int64("duration_ms", result.Duration),
	)

	return result, nil
}

// ResolveIP resolves a domain to IP addresses only.
func (s *DNSService) ResolveIP(domain string) ([]string, error) {
	if domain == "" {
		return nil, errors.New("域名不能为空")
	}

	domain = strings.TrimSpace(domain)
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.Split(domain, "/")[0]
	domain = strings.Split(domain, ":")[0]

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	ips, err := s.resolver.LookupHost(ctx, domain)
	if err != nil {
		return nil, errors.New("域名解析失败: " + err.Error())
	}

	return ips, nil
}

// isValidDomain checks if a domain name is valid.
func isValidDomain(domain string) bool {
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}

	// Check for valid characters
	for _, c := range domain {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '.') {
			return false
		}
	}

	// Must contain at least one dot
	if !strings.Contains(domain, ".") {
		return false
	}

	// Cannot start or end with dot or hyphen
	if domain[0] == '.' || domain[0] == '-' ||
		domain[len(domain)-1] == '.' || domain[len(domain)-1] == '-' {
		return false
	}

	return true
}
