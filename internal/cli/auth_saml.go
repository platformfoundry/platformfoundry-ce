package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/platformfoundry/pf-ce/internal/auth"
	"github.com/spf13/cobra"
)

var (
	samlEntityID        string
	samlACSURL          string
	samlMetadataURL     string
	samlIDPMetadataPath string
	samlCertificate     string
	samlPrivateKey      string

	ssoClientID     string
	ssoClientSecret string
	ssoRedirectURL  string
	ssoTenantID     string
	ssoDomain       string
	ssoScopes       []string
	ssoPort         int
)

var samlCmd = &cobra.Command{
	Use:   "saml",
	Short: "SAML authentication",
	Long:  `Manage SAML authentication and generate service provider metadata.`,
}

var samlMetadataCmd = &cobra.Command{
	Use:   "metadata",
	Short: "Generate SAML service provider metadata",
	Long:  `Generate SAML service provider metadata XML for IdP configuration.`,
	Example: `  pf auth saml metadata --entity-id https://app.example.com --acs-url https://app.example.com/saml/acs
  pf auth saml metadata --entity-id https://app.example.com --acs-url https://app.example.com/saml/acs --cert saml-cert.pem --key saml-key.pem`,
	RunE: runSAMLMetadata,
}

var samlLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Initiate SAML login flow",
	Long:  `Generate SAML authentication URL to initiate login with IdP.`,
	Example: `  pf auth saml login --metadata-url https://idp.example.com/metadata
  pf auth saml login --idp-metadata /path/to/idp-metadata.xml`,
	RunE: runSAMLLogin,
}

var ssoCmd = &cobra.Command{
	Use:   "sso",
	Short: "SSO provider authentication",
	Long:  `Authenticate using SSO providers like Google, GitHub, Azure AD, or Okta.`,
}

var ssoLoginCmd = &cobra.Command{
	Use:   "login <provider>",
	Short: "Login via SSO provider",
	Long:  `Initiate OAuth login flow with an SSO provider.`,
	Example: `  pf auth sso login google --client-id xxx --client-secret yyy
  pf auth sso login github --client-id xxx --client-secret yyy
  pf auth sso login azuread --client-id xxx --client-secret yyy --tenant-id zzz
  pf auth sso login okta --client-id xxx --client-secret yyy --domain example.okta.com`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: ssoProviderCompletion,
	RunE:              runSSOLogin,
}

var ssoCallbackCmd = &cobra.Command{
	Use:   "callback",
	Short: "Handle SSO callback (internal use)",
	Long:  `Handle OAuth callback from SSO provider. This is called automatically by the browser.`,
	Hidden: true,
	RunE: runSSOCallback,
}

func init() {
	// SAML metadata flags
	samlMetadataCmd.Flags().StringVar(&samlEntityID, "entity-id", "", "Service provider entity ID (required)")
	samlMetadataCmd.Flags().StringVar(&samlACSURL, "acs-url", "", "Assertion Consumer Service URL (required)")
	samlMetadataCmd.Flags().StringVar(&samlCertificate, "cert", "", "Certificate file for signing (optional)")
	samlMetadataCmd.Flags().StringVar(&samlPrivateKey, "key", "", "Private key file for signing (optional)")
	samlMetadataCmd.MarkFlagRequired("entity-id")
	samlMetadataCmd.MarkFlagRequired("acs-url")

	// SAML login flags
	samlLoginCmd.Flags().StringVar(&samlEntityID, "entity-id", "", "Service provider entity ID (required)")
	samlLoginCmd.Flags().StringVar(&samlACSURL, "acs-url", "", "Assertion Consumer Service URL (required)")
	samlLoginCmd.Flags().StringVar(&samlMetadataURL, "metadata-url", "", "IdP metadata URL")
	samlLoginCmd.Flags().StringVar(&samlIDPMetadataPath, "idp-metadata", "", "Path to IdP metadata XML file")
	samlLoginCmd.Flags().StringVar(&samlCertificate, "cert", "", "Certificate file for signing (optional)")
	samlLoginCmd.Flags().StringVar(&samlPrivateKey, "key", "", "Private key file for signing (optional)")
	samlLoginCmd.MarkFlagRequired("entity-id")
	samlLoginCmd.MarkFlagRequired("acs-url")

	// SSO login flags
	ssoLoginCmd.Flags().StringVar(&ssoClientID, "client-id", "", "OAuth client ID (required)")
	ssoLoginCmd.Flags().StringVar(&ssoClientSecret, "client-secret", "", "OAuth client secret (required)")
	ssoLoginCmd.Flags().StringVar(&ssoRedirectURL, "redirect-url", "http://localhost:8080/auth/callback", "OAuth redirect URL")
	ssoLoginCmd.Flags().StringVar(&ssoTenantID, "tenant-id", "", "Azure AD tenant ID (for Azure AD)")
	ssoLoginCmd.Flags().StringVar(&ssoDomain, "domain", "", "Okta domain (for Okta)")
	ssoLoginCmd.Flags().StringSliceVar(&ssoScopes, "scopes", []string{}, "Custom OAuth scopes")
	ssoLoginCmd.Flags().IntVar(&ssoPort, "port", 8080, "Local callback server port")
	ssoLoginCmd.MarkFlagRequired("client-id")
	ssoLoginCmd.MarkFlagRequired("client-secret")

	// Add subcommands
	samlCmd.AddCommand(samlMetadataCmd)
	samlCmd.AddCommand(samlLoginCmd)
	ssoCmd.AddCommand(ssoLoginCmd)
	ssoCmd.AddCommand(ssoCallbackCmd)
	authCmd.AddCommand(samlCmd)
	authCmd.AddCommand(ssoCmd)
}

func runSAMLMetadata(cmd *cobra.Command, args []string) error {
	// Create SAML config
	config := &auth.SAMLConfig{
		EntityID:    samlEntityID,
		ACSURL:      samlACSURL,
		Certificate: samlCertificate,
		PrivateKey:  samlPrivateKey,
	}

	// Create SAML manager (without user store and JWT manager for metadata generation)
	userStore, err := auth.NewUserStore("")
	if err != nil {
		return fmt.Errorf("failed to create user store: %w", err)
	}

	jwtManager := auth.NewJWTManager("temp-secret-key", "pf", 24*time.Hour)

	manager, err := auth.NewSAMLManager(config, userStore, jwtManager)
	if err != nil {
		return fmt.Errorf("failed to create SAML manager: %w", err)
	}

	// Get metadata
	metadata, err := manager.GetMetadata()
	if err != nil {
		return fmt.Errorf("failed to generate metadata: %w", err)
	}

	fmt.Println("SAML Service Provider Metadata:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(metadata)
	fmt.Println()
	fmt.Println("Copy this XML and configure it in your IdP.")

	return nil
}

func runSAMLLogin(cmd *cobra.Command, args []string) error {
	if samlMetadataURL == "" && samlIDPMetadataPath == "" {
		return fmt.Errorf("either --metadata-url or --idp-metadata must be provided")
	}

	// Create SAML config
	config := &auth.SAMLConfig{
		EntityID:        samlEntityID,
		ACSURL:          samlACSURL,
		MetadataURL:     samlMetadataURL,
		IDPMetadataPath: samlIDPMetadataPath,
		Certificate:     samlCertificate,
		PrivateKey:      samlPrivateKey,
	}

	// Create SAML manager
	userStore, err := auth.NewUserStore("")
	if err != nil {
		return fmt.Errorf("failed to create user store: %w", err)
	}

	jwtManager := auth.NewJWTManager("temp-secret-key", "pf", 24*time.Hour)

	manager, err := auth.NewSAMLManager(config, userStore, jwtManager)
	if err != nil {
		return fmt.Errorf("failed to create SAML manager: %w", err)
	}

	// Build auth URL
	authURL, err := manager.BuildAuthURL("")
	if err != nil {
		return fmt.Errorf("failed to build auth URL: %w", err)
	}

	fmt.Println("SAML Login URL:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("Open this URL in your browser to login via SAML.")

	return nil
}

func runSSOLogin(cmd *cobra.Command, args []string) error {
	providerName := args[0]

	// Validate provider
	var providerType auth.SSOProviderType
	switch providerName {
	case "google":
		providerType = auth.SSOProviderGoogle
	case "github":
		providerType = auth.SSOProviderGitHub
	case "azure-ad":
		providerType = auth.SSOProviderAzureAD
		if ssoTenantID == "" {
			return fmt.Errorf("--tenant-id is required for Azure AD")
		}
	case "okta":
		providerType = auth.SSOProviderOkta
		if ssoDomain == "" {
			return fmt.Errorf("--domain is required for Okta")
		}
	default:
		return fmt.Errorf("unsupported provider: %s (use google, github, azure-ad, or okta)", providerName)
	}

	// Create SSO config
	config := &auth.SSOConfig{
		Provider:     providerType,
		ClientID:     ssoClientID,
		ClientSecret: ssoClientSecret,
		RedirectURL:  ssoRedirectURL,
		TenantID:     ssoTenantID,
		Domain:       ssoDomain,
		Scopes:       ssoScopes,
	}

	// Create SSO provider
	provider, err := auth.NewSSOProvider(config)
	if err != nil {
		return fmt.Errorf("failed to create SSO provider: %w", err)
	}

	// Generate state token
	state := fmt.Sprintf("pf-sso-%d", time.Now().Unix())

	// Get auth URL
	authURL := provider.GetAuthURL(state)

	fmt.Printf("SSO Login URL (%s):\n", providerName)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(authURL)
	fmt.Println()

	// Start local callback server
	tokenChan := make(chan string, 1)
	errChan := make(chan error, 1)

	server := &http.Server{
		Addr: fmt.Sprintf(":%d", ssoPort),
	}

	http.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		returnedState := r.URL.Query().Get("state")

		if returnedState != state {
			errChan <- fmt.Errorf("state mismatch: expected %s, got %s", state, returnedState)
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		if code == "" {
			errChan <- fmt.Errorf("no authorization code received")
			http.Error(w, "No authorization code", http.StatusBadRequest)
			return
		}

		// Exchange code for token
		ctx := context.Background()
		token, err := provider.ExchangeCode(ctx, code)
		if err != nil {
			errChan <- fmt.Errorf("failed to exchange code: %w", err)
			http.Error(w, "Failed to exchange code", http.StatusInternalServerError)
			return
		}

		// Get user info
		userInfo, err := provider.GetUserInfo(ctx, token)
		if err != nil {
			errChan <- fmt.Errorf("failed to get user info: %w", err)
			http.Error(w, "Failed to get user info", http.StatusInternalServerError)
			return
		}

		// Normalize user info
		response := provider.NormalizeUserInfo(userInfo)

		// Store token (simplified - in production, generate JWT)
		tokenChan <- fmt.Sprintf("%s|%s|%s", response.Username, response.Email, response.DisplayName)

		fmt.Fprintf(w, `<html><body>
			<h1>Login Successful!</h1>
			<p>You have successfully logged in as <strong>%s</strong>.</p>
			<p>You can close this window now.</p>
		</body></html>`, response.Email)
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	fmt.Printf("Waiting for OAuth callback on http://localhost:%d/auth/callback ...\n", ssoPort)
	fmt.Println("Opening browser...")

	// Try to open browser
	openBrowser(authURL)

	// Wait for callback or error
	select {
	case tokenInfo := <-tokenChan:
		fmt.Println("\n✓ Login successful!")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		parts := splitN(tokenInfo, "|", 3)
		fmt.Printf("Username: %s\n", parts[0])
		fmt.Printf("Email: %s\n", parts[1])
		fmt.Printf("Display Name: %s\n", parts[2])

		// Shutdown server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)

		return nil

	case err := <-errChan:
		// Shutdown server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)

		return fmt.Errorf("SSO login failed: %w", err)

	case <-time.After(5 * time.Minute):
		// Timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)

		return fmt.Errorf("SSO login timeout after 5 minutes")
	}
}

func runSSOCallback(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("this command is for internal use only")
}

// Helper functions

func openBrowser(url string) {
	var err error
	switch os.Getenv("GOOS") {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		fmt.Printf("Could not open browser automatically. Please open manually:\n%s\n", url)
	}
}

func splitN(s, sep string, n int) []string {
	parts := make([]string, n)
	idx := 0
	for i := 0; i < n-1; i++ {
		sepIdx := indexOf(s[idx:], sep)
		if sepIdx == -1 {
			parts[i] = s[idx:]
			return parts[:i+1]
		}
		parts[i] = s[idx : idx+sepIdx]
		idx += sepIdx + len(sep)
	}
	parts[n-1] = s[idx:]
	return parts
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
