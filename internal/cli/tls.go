package cli

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/tls"
	"github.com/spf13/cobra"
)

var (
	tlsDomain       string
	tlsEmail        string
	tlsOrg          string
	tlsDNSNames     []string
	tlsIPAddresses  []string
	tlsValidFor     string
	tlsStaging      bool
	tlsAcceptTOS    bool
	tlsRenewBefore  string
)

var tlsCmd = &cobra.Command{
	Use:   "tls",
	Short: "TLS certificate management",
	Long:  `Manage TLS certificates for Platform Foundry.`,
}

var genSelfSignedCmd = &cobra.Command{
	Use:   "gen-selfsigned",
	Short: "Generate self-signed certificate",
	Long:  `Generate a self-signed TLS certificate for development or testing.`,
	Example: `  pf tls gen-selfsigned --domain localhost
  pf tls gen-selfsigned --domain example.local --dns example.local,*.example.local
  pf tls gen-selfsigned --domain myapp.dev --ip 127.0.0.1,192.168.1.100`,
	RunE: runGenSelfSigned,
}

var obtainCertCmd = &cobra.Command{
	Use:   "obtain <domain>",
	Short: "Obtain certificate from Let's Encrypt",
	Long:  `Obtain a TLS certificate from Let's Encrypt using ACME protocol.`,
	Example: `  pf tls obtain example.com --email admin@example.com --accept-tos
  pf tls obtain api.example.com --email admin@example.com --accept-tos --staging`,
	Args: cobra.ExactArgs(1),
	RunE: runObtainCert,
}

var listCertsCmd = &cobra.Command{
	Use:   "list",
	Short: "List TLS certificates",
	Long:  `List all TLS certificates managed by Platform Foundry.`,
	RunE:  runListCerts,
}

var showCertCmd = &cobra.Command{
	Use:   "show <domain>",
	Short: "Show certificate details",
	Long:  `Display detailed information about a TLS certificate.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runShowCert,
}

var renewCertCmd = &cobra.Command{
	Use:   "renew <domain>",
	Short: "Renew a certificate",
	Long:  `Renew a TLS certificate before it expires.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRenewCert,
}

var deleteCertCmd = &cobra.Command{
	Use:   "delete <domain>",
	Short: "Delete a certificate",
	Long:  `Delete a TLS certificate and its private key.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDeleteCert,
}

func init() {
	// Gen self-signed flags
	genSelfSignedCmd.Flags().StringVar(&tlsDomain, "domain", "localhost", "Domain name (Common Name)")
	genSelfSignedCmd.Flags().StringVar(&tlsOrg, "org", "Platform Foundry", "Organization name")
	genSelfSignedCmd.Flags().StringSliceVar(&tlsDNSNames, "dns", []string{}, "Additional DNS names (comma-separated)")
	genSelfSignedCmd.Flags().StringSliceVar(&tlsIPAddresses, "ip", []string{}, "IP addresses (comma-separated)")
	genSelfSignedCmd.Flags().StringVar(&tlsValidFor, "valid-for", "365d", "Certificate validity duration (e.g., 365d, 1y)")

	// Obtain cert flags
	obtainCertCmd.Flags().StringVar(&tlsEmail, "email", "", "Email address for ACME account (required)")
	obtainCertCmd.Flags().BoolVar(&tlsAcceptTOS, "accept-tos", false, "Accept Let's Encrypt Terms of Service (required)")
	obtainCertCmd.Flags().BoolVar(&tlsStaging, "staging", false, "Use Let's Encrypt staging environment (for testing)")
	obtainCertCmd.Flags().StringVar(&tlsRenewBefore, "renew-before", "720h", "Renew certificate this duration before expiry")
	obtainCertCmd.MarkFlagRequired("email")
	obtainCertCmd.MarkFlagRequired("accept-tos")

	// Renew cert flags
	renewCertCmd.Flags().StringVar(&tlsEmail, "email", "", "Email address for ACME account (required)")
	renewCertCmd.Flags().BoolVar(&tlsStaging, "staging", false, "Use Let's Encrypt staging environment")
	renewCertCmd.MarkFlagRequired("email")

	// Add subcommands
	tlsCmd.AddCommand(genSelfSignedCmd)
	tlsCmd.AddCommand(obtainCertCmd)
	tlsCmd.AddCommand(listCertsCmd)
	tlsCmd.AddCommand(showCertCmd)
	tlsCmd.AddCommand(renewCertCmd)
	tlsCmd.AddCommand(deleteCertCmd)
}

func runGenSelfSigned(cmd *cobra.Command, args []string) error {
	// Parse validity duration
	validFor, err := parseDuration(tlsValidFor)
	if err != nil {
		return fmt.Errorf("invalid validity duration: %w", err)
	}

	// Parse IP addresses
	var ipAddresses []net.IP
	for _, ipStr := range tlsIPAddresses {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return fmt.Errorf("invalid IP address: %s", ipStr)
		}
		ipAddresses = append(ipAddresses, ip)
	}

	// Create TLS manager
	manager, err := tls.NewManager("", false)
	if err != nil {
		return fmt.Errorf("failed to create TLS manager: %w", err)
	}

	// Prepare configuration
	config := &tls.SelfSignedConfig{
		CommonName:   tlsDomain,
		Organization: tlsOrg,
		DNSNames:     tlsDNSNames,
		IPAddresses:  ipAddresses,
		ValidFor:     validFor,
	}

	// Generate certificate
	fmt.Printf("Generating self-signed certificate for %s...\n", tlsDomain)
	certFile, keyFile, err := manager.GenerateSelfSigned(config)
	if err != nil {
		return fmt.Errorf("failed to generate certificate: %w", err)
	}

	fmt.Println("✓ Self-signed certificate generated successfully")
	fmt.Println()
	fmt.Printf("  Certificate: %s\n", certFile)
	fmt.Printf("  Private Key: %s\n", keyFile)
	fmt.Println()
	fmt.Println("  Use in server configuration:")
	fmt.Printf("    tls:\n")
	fmt.Printf("      enabled: true\n")
	fmt.Printf("      certFile: %s\n", certFile)
	fmt.Printf("      keyFile: %s\n", keyFile)

	return nil
}

func runObtainCert(cmd *cobra.Command, args []string) error {
	domain := args[0]

	// Parse renew-before duration
	renewBefore, err := time.ParseDuration(tlsRenewBefore)
	if err != nil {
		return fmt.Errorf("invalid renew-before duration: %w", err)
	}

	// Create TLS manager
	manager, err := tls.NewManager("", true)
	if err != nil {
		return fmt.Errorf("failed to create TLS manager: %w", err)
	}

	// Create ACME config
	acmeConfig := &tls.ACMEConfig{
		Domain:      domain,
		Email:       tlsEmail,
		AcceptTOS:   tlsAcceptTOS,
		Staging:     tlsStaging,
		RenewBefore: renewBefore,
	}

	// Create autocert manager
	autoCert, err := manager.NewAutoCertManager(acmeConfig)
	if err != nil {
		return fmt.Errorf("failed to create autocert manager: %w", err)
	}

	fmt.Printf("Obtaining certificate from Let's Encrypt for %s...\n", domain)
	if tlsStaging {
		fmt.Println("(Using staging environment)")
	}
	fmt.Println()
	fmt.Println("This requires:")
	fmt.Println("  1. Domain must be publicly accessible")
	fmt.Println("  2. Port 80 must be open for HTTP-01 challenge")
	fmt.Println("  3. DNS must point to this server")
	fmt.Println()

	ctx := context.Background()
	if err := autoCert.ObtainCertificate(ctx, domain); err != nil {
		return fmt.Errorf("failed to obtain certificate: %w", err)
	}

	return nil
}

func runListCerts(cmd *cobra.Command, args []string) error {
	manager, err := tls.NewManager("", false)
	if err != nil {
		return fmt.Errorf("failed to create TLS manager: %w", err)
	}

	certs, err := manager.ListCertificates()
	if err != nil {
		return fmt.Errorf("failed to list certificates: %w", err)
	}

	if len(certs) == 0 {
		fmt.Println("No certificates found")
		return nil
	}

	fmt.Println("TLS Certificates:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	now := time.Now()
	for _, cert := range certs {
		status := "valid"
		if now.After(cert.NotAfter) {
			status = "expired"
		} else if now.Add(30 * 24 * time.Hour).After(cert.NotAfter) {
			status = "expiring soon"
		}

		certType := "CA issued"
		if cert.SelfSigned {
			certType = "self-signed"
		}

		fmt.Printf("  %s (%s, %s)\n", cert.Domain, certType, status)
		fmt.Printf("    Issuer: %s\n", cert.Issuer)
		fmt.Printf("    Valid: %s - %s\n", cert.NotBefore.Format("2006-01-02"), cert.NotAfter.Format("2006-01-02"))

		if len(cert.DNSNames) > 0 {
			fmt.Printf("    DNS Names: %s\n", strings.Join(cert.DNSNames, ", "))
		}
		if len(cert.IPAddresses) > 0 {
			fmt.Printf("    IP Addresses: %s\n", strings.Join(cert.IPAddresses, ", "))
		}

		// Calculate days until expiry
		daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)
		if daysUntilExpiry >= 0 {
			fmt.Printf("    Expires in: %d days\n", daysUntilExpiry)
		}

		fmt.Println()
	}

	return nil
}

func runShowCert(cmd *cobra.Command, args []string) error {
	domain := args[0]

	manager, err := tls.NewManager("", false)
	if err != nil {
		return fmt.Errorf("failed to create TLS manager: %w", err)
	}

	certFile, _ := manager.GetCertPath(domain)
	cert, err := manager.GetCertInfo(certFile)
	if err != nil {
		return fmt.Errorf("failed to get certificate info: %w", err)
	}

	fmt.Printf("Certificate: %s\n", domain)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Domain (CN):      %s\n", cert.Domain)
	fmt.Printf("  Issuer:           %s\n", cert.Issuer)
	fmt.Printf("  Type:             %s\n", map[bool]string{true: "Self-signed", false: "CA issued"}[cert.SelfSigned])
	fmt.Printf("  Valid From:       %s\n", cert.NotBefore.Format(time.RFC3339))
	fmt.Printf("  Valid Until:      %s\n", cert.NotAfter.Format(time.RFC3339))

	if len(cert.DNSNames) > 0 {
		fmt.Printf("  DNS Names:        %s\n", strings.Join(cert.DNSNames, ", "))
	}
	if len(cert.IPAddresses) > 0 {
		fmt.Printf("  IP Addresses:     %s\n", strings.Join(cert.IPAddresses, ", "))
	}

	// Status
	now := time.Now()
	if now.After(cert.NotAfter) {
		fmt.Println("  Status:           ❌ EXPIRED")
	} else if now.Add(30 * 24 * time.Hour).After(cert.NotAfter) {
		fmt.Println("  Status:           ⚠️  EXPIRING SOON")
	} else {
		fmt.Println("  Status:           ✓ Valid")
	}

	// Days until expiry
	daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)
	if daysUntilExpiry >= 0 {
		fmt.Printf("  Expires in:       %d days\n", daysUntilExpiry)
	}

	fmt.Println()
	fmt.Printf("  Certificate file: %s\n", certFile)

	return nil
}

func runRenewCert(cmd *cobra.Command, args []string) error {
	domain := args[0]

	manager, err := tls.NewManager("", true)
	if err != nil {
		return fmt.Errorf("failed to create TLS manager: %w", err)
	}

	acmeConfig := &tls.ACMEConfig{
		Domain:      domain,
		Email:       tlsEmail,
		AcceptTOS:   true,
		Staging:     tlsStaging,
		RenewBefore: 30 * 24 * time.Hour,
	}

	autoCert, err := manager.NewAutoCertManager(acmeConfig)
	if err != nil {
		return fmt.Errorf("failed to create autocert manager: %w", err)
	}

	fmt.Printf("Renewing certificate for %s...\n", domain)
	ctx := context.Background()
	if err := autoCert.RenewCertificate(ctx, domain); err != nil {
		return fmt.Errorf("failed to renew certificate: %w", err)
	}

	fmt.Println("✓ Certificate renewed successfully")
	return nil
}

func runDeleteCert(cmd *cobra.Command, args []string) error {
	domain := args[0]

	manager, err := tls.NewManager("", false)
	if err != nil {
		return fmt.Errorf("failed to create TLS manager: %w", err)
	}

	fmt.Printf("Deleting certificate for %s...\n", domain)
	if err := manager.DeleteCertificate(domain); err != nil {
		return fmt.Errorf("failed to delete certificate: %w", err)
	}

	fmt.Println("✓ Certificate deleted successfully")
	return nil
}
