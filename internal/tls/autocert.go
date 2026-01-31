package tls

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// ACMEConfig represents ACME/Let's Encrypt configuration
type ACMEConfig struct {
	Domain      string
	Email       string
	AcceptTOS   bool
	Staging     bool // Use Let's Encrypt staging environment
	RenewBefore time.Duration
}

// AutoCertManager wraps autocert.Manager with additional functionality
type AutoCertManager struct {
	manager  *autocert.Manager
	certDir  string
	acmeDir  string
	email    string
	domains  []string
	client   *acme.Client
}

// NewAutoCertManager creates a new automatic certificate manager
func (m *Manager) NewAutoCertManager(config *ACMEConfig) (*AutoCertManager, error) {
	if !config.AcceptTOS {
		return nil, fmt.Errorf("must accept Let's Encrypt Terms of Service")
	}

	// Create ACME account directory
	acmeDir := filepath.Join(m.certDir, "acme")
	if err := os.MkdirAll(acmeDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create ACME directory: %w", err)
	}

	// Determine ACME directory URL
	directoryURL := acme.LetsEncryptURL
	if config.Staging {
		directoryURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
	}

	// Create ACME client
	accountKey, err := loadOrCreateAccountKey(acmeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load account key: %w", err)
	}

	client := &acme.Client{
		Key:          accountKey,
		DirectoryURL: directoryURL,
	}

	// Create autocert manager
	manager := &autocert.Manager{
		Prompt:      autocert.AcceptTOS,
		Cache:       autocert.DirCache(m.certDir),
		HostPolicy:  autocert.HostWhitelist(config.Domain),
		Email:       config.Email,
		RenewBefore: config.RenewBefore,
	}

	return &AutoCertManager{
		manager:  manager,
		certDir:  m.certDir,
		acmeDir:  acmeDir,
		email:    config.Email,
		domains:  []string{config.Domain},
		client:   client,
	}, nil
}

// GetCertificate obtains a certificate using ACME
func (a *AutoCertManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return a.manager.GetCertificate(hello)
}

// ObtainCertificate manually obtains a certificate for a domain
func (a *AutoCertManager) ObtainCertificate(ctx context.Context, domain string) error {
	// Create account if needed
	account, err := a.getOrCreateAccount(ctx)
	if err != nil {
		return fmt.Errorf("failed to get ACME account: %w", err)
	}

	// Create order for domain
	order, err := a.client.AuthorizeOrder(ctx, acme.DomainIDs(domain))
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	// Complete challenges
	for _, authURL := range order.AuthzURLs {
		auth, err := a.client.GetAuthorization(ctx, authURL)
		if err != nil {
			return fmt.Errorf("failed to get authorization: %w", err)
		}

		// Find HTTP-01 challenge
		var challenge *acme.Challenge
		for _, c := range auth.Challenges {
			if c.Type == "http-01" {
				challenge = c
				break
			}
		}

		if challenge == nil {
			return fmt.Errorf("no http-01 challenge found")
		}

		// Accept challenge
		if _, err := a.client.Accept(ctx, challenge); err != nil {
			return fmt.Errorf("failed to accept challenge: %w", err)
		}

		// Wait for challenge to complete
		if _, err := a.client.WaitAuthorization(ctx, authURL); err != nil {
			return fmt.Errorf("challenge failed: %w", err)
		}
	}

	// Generate CSR
	csr, certKey, err := a.generateCSR(domain)
	if err != nil {
		return fmt.Errorf("failed to generate CSR: %w", err)
	}

	// Finalize order
	der, _, err := a.client.CreateOrderCert(ctx, order.FinalizeURL, csr, true)
	if err != nil {
		return fmt.Errorf("failed to finalize order: %w", err)
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(der[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Save certificate and key
	certFile := filepath.Join(a.certDir, domain+".crt")
	keyFile := filepath.Join(a.certDir, domain+".key")

	if err := saveCertificatePEM(cert, certFile); err != nil {
		return fmt.Errorf("failed to save certificate: %w", err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(certKey)
	if err != nil {
		return fmt.Errorf("failed to marshal key: %w", err)
	}

	if err := savePrivateKeyPEM(keyBytes, keyFile); err != nil {
		return fmt.Errorf("failed to save key: %w", err)
	}

	fmt.Printf("✓ Certificate obtained for %s\n", domain)
	fmt.Printf("  Certificate: %s\n", certFile)
	fmt.Printf("  Key: %s\n", keyFile)
	fmt.Printf("  Valid until: %s\n", cert.NotAfter.Format(time.RFC3339))

	_ = account // Avoid unused variable warning
	return nil
}

// RenewCertificate renews a certificate before it expires
func (a *AutoCertManager) RenewCertificate(ctx context.Context, domain string) error {
	// For autocert, renewal is handled automatically
	// This method is for manual renewal if needed
	return a.ObtainCertificate(ctx, domain)
}

// getOrCreateAccount gets or creates an ACME account
func (a *AutoCertManager) getOrCreateAccount(ctx context.Context) (*acme.Account, error) {
	// Try to get existing account
	account, err := a.client.GetReg(ctx, "")
	if err == nil {
		return account, nil
	}

	// Create new account
	account = &acme.Account{
		Contact: []string{"mailto:" + a.email},
	}

	account, err = a.client.Register(ctx, account, autocert.AcceptTOS)
	if err != nil {
		return nil, fmt.Errorf("failed to register account: %w", err)
	}

	return account, nil
}

// generateCSR generates a Certificate Signing Request
func (a *AutoCertManager) generateCSR(domain string) ([]byte, *ecdsa.PrivateKey, error) {
	// Generate key
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	// Create CSR template
	template := &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: domain},
		DNSNames: []string{domain},
	}

	// Create CSR
	csr, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		return nil, nil, err
	}

	return csr, key, nil
}

// loadOrCreateAccountKey loads or creates an ACME account key
func loadOrCreateAccountKey(acmeDir string) (crypto.Signer, error) {
	keyPath := filepath.Join(acmeDir, "account.key")

	// Try to load existing key
	if keyData, err := os.ReadFile(keyPath); err == nil {
		block, _ := decodePEM(keyData)
		if block != nil {
			key, err := x509.ParseECPrivateKey(block.Bytes)
			if err == nil {
				return key, nil
			}
		}
	}

	// Generate new key
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	// Save key
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}

	keyPEM := encodePEM("EC PRIVATE KEY", keyBytes)
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return nil, err
	}

	return key, nil
}
