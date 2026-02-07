package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
	dsig "github.com/russellhaering/goxmldsig"
)

// SAMLConfig represents SAML configuration
type SAMLConfig struct {
	// EntityID is the service provider entity ID (your app's URL)
	EntityID string `yaml:"entityId" json:"entityId"`

	// ACSURL is the Assertion Consumer Service URL (callback URL)
	ACSURL string `yaml:"acsUrl" json:"acsUrl"`

	// MetadataURL is the IdP metadata URL
	MetadataURL string `yaml:"metadataUrl" json:"metadataUrl"`

	// Certificate for signing requests (optional)
	Certificate string `yaml:"certificate,omitempty" json:"certificate,omitempty"`

	// PrivateKey for signing requests (optional)
	PrivateKey string `yaml:"privateKey,omitempty" json:"privateKey,omitempty"`

	// IDPMetadataPath is the path to cached IdP metadata
	IDPMetadataPath string `yaml:"idpMetadataPath,omitempty" json:"idpMetadataPath,omitempty"`

	// AllowIdpInitiated allows IdP-initiated logins
	AllowIdpInitiated bool `yaml:"allowIdpInitiated,omitempty" json:"allowIdpInitiated,omitempty"`

	// DefaultRole for SAML users
	DefaultRole string `yaml:"defaultRole,omitempty" json:"defaultRole,omitempty"`

	// AttributeMappings maps SAML attributes to user fields
	AttributeMappings map[string]string `yaml:"attributeMappings,omitempty" json:"attributeMappings,omitempty"`
}

// SAMLManager handles SAML authentication
type SAMLManager struct {
	config      *SAMLConfig
	sp          *saml2.SAMLServiceProvider
	userStore   *UserStore
	jwtManager  *JWTManager
	metadataDir string
}

// SAMLResponse represents a parsed SAML response
type SAMLResponse struct {
	Email        string
	Username     string
	DisplayName  string
	Roles        []string
	Attributes   map[string]string
	Organization string
}

// NewSAMLManager creates a new SAML manager
func NewSAMLManager(config *SAMLConfig, userStore *UserStore, jwtManager *JWTManager) (*SAMLManager, error) {
	if config.EntityID == "" {
		return nil, fmt.Errorf("SAML entity ID is required")
	}
	if config.ACSURL == "" {
		return nil, fmt.Errorf("SAML ACS URL is required")
	}

	// Set up metadata directory
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	metadataDir := filepath.Join(home, ".platformfoundry", "saml")
	if err := os.MkdirAll(metadataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create SAML metadata directory: %w", err)
	}

	manager := &SAMLManager{
		config:      config,
		userStore:   userStore,
		jwtManager:  jwtManager,
		metadataDir: metadataDir,
	}

	// Initialize SAML service provider
	if err := manager.initServiceProvider(); err != nil {
		return nil, fmt.Errorf("failed to initialize SAML service provider: %w", err)
	}

	return manager, nil
}

// initServiceProvider initializes the SAML service provider
func (m *SAMLManager) initServiceProvider() error {
	// Load or fetch IdP metadata
	metadata, err := m.loadIdPMetadata()
	if err != nil {
		return fmt.Errorf("failed to load IdP metadata: %w", err)
	}

	// Parse certificates
	certStore := dsig.MemoryX509CertificateStore{
		Roots: []*x509.Certificate{},
	}

	for _, cert := range metadata.IDPSSODescriptor.KeyDescriptors {
		for _, certData := range cert.KeyInfo.X509Data.X509Certificates {
			certBytes, err := base64.StdEncoding.DecodeString(certData.Data)
			if err != nil {
				continue
			}
			parsedCert, err := x509.ParseCertificate(certBytes)
			if err != nil {
				continue
			}
			certStore.Roots = append(certStore.Roots, parsedCert)
		}
	}

	// Create service provider
	sp := &saml2.SAMLServiceProvider{
		IdentityProviderSSOURL:      metadata.IDPSSODescriptor.SingleSignOnServices[0].Location,
		IdentityProviderIssuer:      metadata.EntityID,
		ServiceProviderIssuer:       m.config.EntityID,
		AssertionConsumerServiceURL: m.config.ACSURL,
		SignAuthnRequests:           false,
		AudienceURI:                 m.config.EntityID,
		IDPCertificateStore:         &certStore,
		AllowMissingAttributes:      true,
	}

	// Configure signing if certificate and key provided
	if m.config.Certificate != "" && m.config.PrivateKey != "" {
		keyPair, err := m.loadSigningKeyPair()
		if err != nil {
			return fmt.Errorf("failed to load signing key pair: %w", err)
		}
		sp.SignAuthnRequests = true
		sp.SPKeyStore = keyPair
	}

	m.sp = sp
	return nil
}

// BuildAuthURL builds the SAML authentication URL
func (m *SAMLManager) BuildAuthURL(relayState string) (string, error) {
	if m.sp == nil {
		return "", fmt.Errorf("SAML service provider not initialized")
	}

	authURL, err := m.sp.BuildAuthURL(relayState)
	if err != nil {
		return "", fmt.Errorf("failed to build auth URL: %w", err)
	}

	return authURL, nil
}

// ValidateResponse validates a SAML response and returns user info
func (m *SAMLManager) ValidateResponse(samlResponse string) (*SAMLResponse, error) {
	if m.sp == nil {
		return nil, fmt.Errorf("SAML service provider not initialized")
	}

	// Validate the SAML assertion
	assertionInfo, err := m.sp.RetrieveAssertionInfo(samlResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to validate SAML response: %w", err)
	}

	// Parse warnings
	if assertionInfo.WarningInfo != nil && assertionInfo.WarningInfo.InvalidTime {
		return nil, fmt.Errorf("SAML assertion time is invalid")
	}

	// Extract attributes
	response := &SAMLResponse{
		Attributes: make(map[string]string),
	}

	// Get attribute mappings
	emailAttr := m.config.AttributeMappings["email"]
	if emailAttr == "" {
		emailAttr = "email"
	}
	usernameAttr := m.config.AttributeMappings["username"]
	if usernameAttr == "" {
		usernameAttr = "username"
	}
	displayNameAttr := m.config.AttributeMappings["displayName"]
	if displayNameAttr == "" {
		displayNameAttr = "displayName"
	}

	// Extract standard attributes
	if assertionInfo.Values != nil {
		response.Email = assertionInfo.Values.Get(emailAttr)
		response.Username = assertionInfo.Values.Get(usernameAttr)
		response.DisplayName = assertionInfo.Values.Get(displayNameAttr)

		// Store all attributes
		for key, values := range assertionInfo.Values {
			if len(values.Values) > 0 {
				response.Attributes[key] = values.Values[0].Value
			}
		}
	}

	// Use NameID as fallback for username
	if response.Username == "" && assertionInfo.NameID != "" {
		response.Username = assertionInfo.NameID
	}

	// Use username as fallback for email if not provided
	if response.Email == "" {
		response.Email = response.Username
	}

	return response, nil
}

// HandleCallback processes SAML callback and returns JWT token
func (m *SAMLManager) HandleCallback(r *http.Request) (string, error) {
	// Parse SAML response
	samlResponse := r.FormValue("SAMLResponse")
	if samlResponse == "" {
		return "", fmt.Errorf("missing SAMLResponse parameter")
	}

	// Validate SAML response
	response, err := m.ValidateResponse(samlResponse)
	if err != nil {
		return "", fmt.Errorf("SAML validation failed: %w", err)
	}

	// Get or create user
	user, err := m.getOrCreateSAMLUser(response)
	if err != nil {
		return "", fmt.Errorf("failed to get or create user: %w", err)
	}

	// Generate JWT token
	token, err := m.jwtManager.GenerateToken(user)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}

// getOrCreateSAMLUser gets an existing user or creates a new one from SAML response
func (m *SAMLManager) getOrCreateSAMLUser(response *SAMLResponse) (*User, error) {
	// Try to get existing user
	user, err := m.userStore.GetUser(response.Username)
	if err == nil {
		// User exists, update attributes
		user.Email = response.Email
		user.UpdatedAt = time.Now()
		if err := m.userStore.UpdateUser(user); err != nil {
			return nil, err
		}
		return user, nil
	}

	// Create new user
	roles := response.Roles
	if len(roles) == 0 && m.config.DefaultRole != "" {
		roles = []string{m.config.DefaultRole}
	}
	if len(roles) == 0 {
		roles = []string{"developer"}
	}

	now := time.Now()
	user = &User{
		Username:     response.Username,
		Email:        response.Email,
		Roles:        roles,
		Organization: response.Organization,
		CreatedAt:    now,
		UpdatedAt:    now,
		Enabled:      true,
		PasswordHash: "", // SAML users don't have passwords
	}

	m.userStore.users[user.Username] = user
	if err := m.userStore.save(); err != nil {
		return nil, err
	}

	return user, nil
}

// loadIdPMetadata loads IdP metadata from URL or file
func (m *SAMLManager) loadIdPMetadata() (*types.EntityDescriptor, error) {
	var metadataBytes []byte
	var err error

	// Try to load from cache first
	cachePath := filepath.Join(m.metadataDir, "idp-metadata.xml")
	if m.config.IDPMetadataPath != "" {
		cachePath = m.config.IDPMetadataPath
	}

	// Load from file if exists
	if _, statErr := os.Stat(cachePath); statErr == nil {
		metadataBytes, err = os.ReadFile(cachePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read metadata file: %w", err)
		}
	} else if m.config.MetadataURL != "" {
		// Fetch from URL
		metadataBytes, err = m.fetchMetadata(m.config.MetadataURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch metadata: %w", err)
		}

		// Cache metadata
		if err := os.WriteFile(cachePath, metadataBytes, 0644); err != nil {
			// Non-fatal, just log
			fmt.Printf("Warning: failed to cache IdP metadata: %v\n", err)
		}
	} else {
		return nil, fmt.Errorf("either metadata URL or metadata file path must be provided")
	}

	// Parse metadata
	var metadata types.EntityDescriptor
	if err := xml.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// fetchMetadata fetches IdP metadata from URL
func (m *SAMLManager) fetchMetadata(metadataURL string) ([]byte, error) {
	resp, err := http.Get(metadataURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch metadata: HTTP %d", resp.StatusCode)
	}

	var buf []byte
	buf = make([]byte, resp.ContentLength)
	if _, err := resp.Body.Read(buf); err != nil && err.Error() != "EOF" {
		return nil, err
	}

	return buf, nil
}

// loadSigningKeyPair loads certificate and private key for signing
func (m *SAMLManager) loadSigningKeyPair() (dsig.X509KeyStore, error) {
	// Read certificate
	certPEM, err := os.ReadFile(m.config.Certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Read private key
	keyPEM, err := os.ReadFile(m.config.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode private key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		// Try PKCS8
		keyInterface, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		var ok bool
		key, ok = keyInterface.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
	}

	// Create keystore using RandomKeyStoreKeyPair wrapper
	return &simpleKeyStore{
		privateKey: key,
		cert:       cert,
	}, nil
}

// simpleKeyStore implements dsig.X509KeyStore
type simpleKeyStore struct {
	privateKey *rsa.PrivateKey
	cert       *x509.Certificate
}

func (s *simpleKeyStore) GetKeyPair() (*rsa.PrivateKey, []byte, error) {
	return s.privateKey, s.cert.Raw, nil
}

func (s *simpleKeyStore) GetChain() ([]*x509.Certificate, error) {
	return []*x509.Certificate{s.cert}, nil
}

// GetMetadata returns the service provider metadata
func (m *SAMLManager) GetMetadata() (string, error) {
	metadata := fmt.Sprintf(`<?xml version="1.0"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="%s">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <AssertionConsumerService
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
      Location="%s"
      index="1" />
  </SPSSODescriptor>
</EntityDescriptor>`, m.config.EntityID, m.config.ACSURL)

	return metadata, nil
}

// BuildLoginURL builds a SAML login URL with relay state
func BuildLoginURL(baseURL string, relayState string) string {
	if relayState == "" {
		return baseURL
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	q := u.Query()
	q.Set("RelayState", relayState)
	u.RawQuery = q.Encode()

	return u.String()
}
