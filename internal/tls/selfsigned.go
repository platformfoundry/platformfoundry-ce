package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

// SelfSignedConfig represents configuration for self-signed certificate
type SelfSignedConfig struct {
	CommonName   string
	Organization string
	Country      string
	Province     string
	Locality     string
	DNSNames     []string
	IPAddresses  []net.IP
	ValidFor     time.Duration
}

// GenerateSelfSigned generates a self-signed certificate
func (m *Manager) GenerateSelfSigned(config *SelfSignedConfig) (certFile, keyFile string, err error) {
	// Generate private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Set defaults
	if config.CommonName == "" {
		config.CommonName = "localhost"
	}
	if config.Organization == "" {
		config.Organization = "Platform Foundry"
	}
	if config.ValidFor == 0 {
		config.ValidFor = 365 * 24 * time.Hour // 1 year
	}

	// Add localhost if no DNS names provided
	if len(config.DNSNames) == 0 {
		config.DNSNames = []string{"localhost"}
	}

	// Add loopback IP if no IPs provided
	if len(config.IPAddresses) == 0 {
		config.IPAddresses = []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
	}

	// Generate serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Create certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(config.ValidFor)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   config.CommonName,
			Organization: []string{config.Organization},
			Country:      []string{config.Country},
			Province:     []string{config.Province},
			Locality:     []string{config.Locality},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              config.DNSNames,
		IPAddresses:           config.IPAddresses,
	}

	// Create self-signed certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %w", err)
	}

	// Marshal private key
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Get file paths
	certFile, keyFile = m.GetCertPath(config.CommonName)

	// Save certificate
	certPEM := encodePEM("CERTIFICATE", certBytes)
	if err := saveFile(certFile, certPEM, 0644); err != nil {
		return "", "", fmt.Errorf("failed to save certificate: %w", err)
	}

	// Save private key
	keyPEM := encodePEM("EC PRIVATE KEY", keyBytes)
	if err := saveFile(keyFile, keyPEM, 0600); err != nil {
		return "", "", fmt.Errorf("failed to save private key: %w", err)
	}

	return certFile, keyFile, nil
}

// saveFile writes data to a file with specified permissions
func saveFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}
