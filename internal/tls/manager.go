package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager handles TLS certificate management
type Manager struct {
	certDir     string
	certCache   map[string]*tls.Certificate
	cacheMu     sync.RWMutex
	autoRenew   bool
	renewBefore time.Duration
}

// Config represents TLS configuration
type Config struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	CertFile    string `yaml:"certFile,omitempty" json:"certFile,omitempty"`
	KeyFile     string `yaml:"keyFile,omitempty" json:"keyFile,omitempty"`
	AutoTLS     bool   `yaml:"autoTLS,omitempty" json:"autoTLS,omitempty"`
	Domain      string `yaml:"domain,omitempty" json:"domain,omitempty"`
	Email       string `yaml:"email,omitempty" json:"email,omitempty"`
	RenewBefore string `yaml:"renewBefore,omitempty" json:"renewBefore,omitempty"` // e.g., "720h" (30 days)
}

// CertInfo represents certificate information
type CertInfo struct {
	Domain      string    `json:"domain"`
	Issuer      string    `json:"issuer"`
	NotBefore   time.Time `json:"notBefore"`
	NotAfter    time.Time `json:"notAfter"`
	DNSNames    []string  `json:"dnsNames,omitempty"`
	IPAddresses []string  `json:"ipAddresses,omitempty"`
	IsCA        bool      `json:"isCA"`
	SelfSigned  bool      `json:"selfSigned"`
}

// NewManager creates a new TLS manager
func NewManager(certDir string, autoRenew bool) (*Manager, error) {
	if certDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		certDir = filepath.Join(home, ".platformfoundry", "certs")
	}

	// Ensure cert directory exists
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cert directory: %w", err)
	}

	return &Manager{
		certDir:     certDir,
		certCache:   make(map[string]*tls.Certificate),
		autoRenew:   autoRenew,
		renewBefore: 30 * 24 * time.Hour, // 30 days default
	}, nil
}

// LoadCertificate loads a certificate from files
func (m *Manager) LoadCertificate(certFile, keyFile string) (*tls.Certificate, error) {
	// Try cache first
	cacheKey := certFile + ":" + keyFile
	m.cacheMu.RLock()
	if cert, exists := m.certCache[cacheKey]; exists {
		m.cacheMu.RUnlock()
		return cert, nil
	}
	m.cacheMu.RUnlock()

	// Load certificate and key
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	// Parse certificate for validation
	if len(cert.Certificate) > 0 {
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}

		// Check if certificate is expired
		now := time.Now()
		if now.Before(x509Cert.NotBefore) {
			return nil, fmt.Errorf("certificate not yet valid (valid from %s)", x509Cert.NotBefore)
		}
		if now.After(x509Cert.NotAfter) {
			return nil, fmt.Errorf("certificate expired on %s", x509Cert.NotAfter)
		}

		// Check if renewal is needed
		// Note: Renewal logic would be implemented here based on certificate source
		_ = m.autoRenew && now.Add(m.renewBefore).After(x509Cert.NotAfter)
	}

	// Cache certificate
	m.cacheMu.Lock()
	m.certCache[cacheKey] = &cert
	m.cacheMu.Unlock()

	return &cert, nil
}

// GetCertInfo extracts information from a certificate file
func (m *Manager) GetCertInfo(certFile string) (*CertInfo, error) {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	cert, err := parseCertificatePEM(certPEM)
	if err != nil {
		return nil, err
	}

	// Check if self-signed
	selfSigned := cert.Subject.String() == cert.Issuer.String()

	// Extract IP addresses
	ipAddresses := make([]string, len(cert.IPAddresses))
	for i, ip := range cert.IPAddresses {
		ipAddresses[i] = ip.String()
	}

	return &CertInfo{
		Domain:      cert.Subject.CommonName,
		Issuer:      cert.Issuer.CommonName,
		NotBefore:   cert.NotBefore,
		NotAfter:    cert.NotAfter,
		DNSNames:    cert.DNSNames,
		IPAddresses: ipAddresses,
		IsCA:        cert.IsCA,
		SelfSigned:  selfSigned,
	}, nil
}

// GetTLSConfig creates a TLS configuration
func (m *Manager) GetTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := m.LoadCertificate(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{*cert},
		MinVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		PreferServerCipherSuites: true,
	}, nil
}

// GetCertPath returns the path for storing a certificate
func (m *Manager) GetCertPath(domain string) (certFile, keyFile string) {
	certFile = filepath.Join(m.certDir, domain+".crt")
	keyFile = filepath.Join(m.certDir, domain+".key")
	return
}

// ClearCache clears the certificate cache
func (m *Manager) ClearCache() {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	m.certCache = make(map[string]*tls.Certificate)
}

// ListCertificates lists all certificates in the cert directory
func (m *Manager) ListCertificates() ([]*CertInfo, error) {
	files, err := os.ReadDir(m.certDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cert directory: %w", err)
	}

	var certs []*CertInfo
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".crt" {
			certPath := filepath.Join(m.certDir, file.Name())
			info, err := m.GetCertInfo(certPath)
			if err != nil {
				continue // Skip invalid certificates
			}
			certs = append(certs, info)
		}
	}

	return certs, nil
}

// DeleteCertificate removes a certificate and its key
func (m *Manager) DeleteCertificate(domain string) error {
	certFile, keyFile := m.GetCertPath(domain)

	// Remove from cache
	cacheKey := certFile + ":" + keyFile
	m.cacheMu.Lock()
	delete(m.certCache, cacheKey)
	m.cacheMu.Unlock()

	// Delete files
	if err := os.Remove(certFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete certificate: %w", err)
	}
	if err := os.Remove(keyFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	return nil
}

// parseCertificatePEM parses a PEM-encoded certificate
func parseCertificatePEM(pemData []byte) (*x509.Certificate, error) {
	block, _ := decodePEM(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, nil
}
