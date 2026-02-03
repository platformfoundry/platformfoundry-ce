package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/platformfoundry/pf-ce/internal/auth"
	"github.com/platformfoundry/pf-ce/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	authUsername string
	authPassword string
	authEmail    string
	authRoles    []string
	authOrg      string
	authToken    string
	apiKeyName   string
	apiKeyExpiry string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication and user management",
	Long:  `Manage authentication, users, and API keys for Platform Foundry.`,
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Platform Foundry",
	Long:  `Authenticate and obtain a session token.`,
	Example: `  pf auth login
  pf auth login --token <jwt>
  pf auth login --username admin`,
	RunE: runLogin,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from Platform Foundry",
	Long:  `Clear stored credentials and session token.`,
	RunE:  runLogout,
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current authenticated user",
	Long:  `Display information about the currently authenticated user.`,
	RunE:  runWhoami,
}

var createUserCmd = &cobra.Command{
	Use:   "create-user <username>",
	Short: "Create a new user",
	Long:  `Create a new Platform Foundry user (requires admin role).`,
	Example: `  pf auth create-user alice --email alice@example.com --roles platform-engineer
  pf auth create-user bob --email bob@example.com --roles admin --org acme`,
	Args: cobra.ExactArgs(1),
	RunE: runCreateUser,
}

var listUsersCmd = &cobra.Command{
	Use:   "list-users",
	Short: "List all users",
	Long:  `List all Platform Foundry users (requires admin role).`,
	RunE:  runListUsers,
}

var resetPasswordCmd = &cobra.Command{
	Use:   "reset-password <username>",
	Short: "Reset user password",
	Long:  `Reset a user's password (requires admin role).`,
	Args:  cobra.ExactArgs(1),
	RunE:  runResetPassword,
}

var createAPIKeyCmd = &cobra.Command{
	Use:   "create-api-key",
	Short: "Create a new API key",
	Long:  `Generate an API key for programmatic access.`,
	Example: `  pf auth create-api-key --name "CI/CD Pipeline"
  pf auth create-api-key --name "Automation" --expiry 90d`,
	RunE: runCreateAPIKey,
}

var listAPIKeysCmd = &cobra.Command{
	Use:   "list-api-keys",
	Short: "List your API keys",
	Long:  `List all API keys for the current user.`,
	RunE:  runListAPIKeys,
}

var revokeAPIKeyCmd = &cobra.Command{
	Use:   "revoke-api-key <key-id>",
	Short: "Revoke an API key",
	Long:  `Disable an API key to prevent further use.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRevokeAPIKey,
}

func init() {
	// Login flags
	loginCmd.Flags().StringVar(&authUsername, "username", "", "Username")
	loginCmd.Flags().StringVar(&authPassword, "password", "", "Password (not recommended, use interactive prompt)")
	loginCmd.Flags().StringVar(&authToken, "token", "", "JWT token to store")

	// Create user flags
	createUserCmd.Flags().StringVar(&authEmail, "email", "", "User email")
	createUserCmd.Flags().StringSliceVar(&authRoles, "roles", []string{"developer"}, "User roles (admin, platform-engineer, developer)")
	createUserCmd.Flags().StringVar(&authOrg, "org", "", "Organization")

	// API key flags
	createAPIKeyCmd.Flags().StringVar(&apiKeyName, "name", "", "API key name (required)")
	createAPIKeyCmd.Flags().StringVar(&apiKeyExpiry, "expiry", "", "Expiration duration (e.g., 30d, 1y, 0 for no expiry)")
	createAPIKeyCmd.MarkFlagRequired("name")

	// Add subcommands
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(whoamiCmd)
	authCmd.AddCommand(createUserCmd)
	authCmd.AddCommand(listUsersCmd)
	authCmd.AddCommand(resetPasswordCmd)
	authCmd.AddCommand(createAPIKeyCmd)
	authCmd.AddCommand(listAPIKeysCmd)
	authCmd.AddCommand(revokeAPIKeyCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	tokenStore, err := auth.NewTokenStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize token store: %w", err)
	}

	// If token provided, just store it
	if authToken != "" {
		// Validate token first
		jwtManager := auth.NewJWTManager(getJWTSecret(), "platformfoundry.io", 24*time.Hour)
		claims, err := jwtManager.ValidateToken(authToken)
		if err != nil {
			return fmt.Errorf("invalid token: %w", err)
		}

		expiresAt := claims.ExpiresAt.Time
		if err := tokenStore.SaveToken(authToken, claims.Username, expiresAt); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		fmt.Printf("✓ Logged in as %s\n", claims.Username)
		return nil
	}

	// Interactive login
	username := authUsername
	password := authPassword

	// Prompt for username if not provided
	if username == "" {
		fmt.Print("Username: ")
		reader := bufio.NewReader(os.Stdin)
		username, _ = reader.ReadString('\n')
		username = strings.TrimSpace(username)
	}

	// Prompt for password if not provided
	if password == "" {
		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		password = string(passwordBytes)
	}

	// Authenticate
	userStore, err := auth.NewUserStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize user store: %w", err)
	}

	user, err := userStore.Authenticate(username, password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Generate JWT token
	jwtManager := auth.NewJWTManager(getJWTSecret(), "platformfoundry.io", 24*time.Hour)
	token, err := jwtManager.GenerateToken(user)
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	// Save token
	expiresAt := time.Now().Add(24 * time.Hour)
	if err := tokenStore.SaveToken(token, user.Username, expiresAt); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Printf("✓ Logged in as %s\n", user.Username)
	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	tokenStore, err := auth.NewTokenStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize token store: %w", err)
	}

	if err := tokenStore.ClearToken(); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	fmt.Println("✓ Logged out successfully")
	return nil
}

func runWhoami(cmd *cobra.Command, args []string) error {
	tokenStore, err := auth.NewTokenStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize token store: %w", err)
	}

	tokenInfo, err := tokenStore.GetToken()
	if err != nil {
		return fmt.Errorf("not logged in: %w", err)
	}

	// Validate and decode token
	jwtManager := auth.NewJWTManager(getJWTSecret(), "platformfoundry.io", 24*time.Hour)
	claims, err := jwtManager.ValidateToken(tokenInfo.Token)
	if err != nil {
		return fmt.Errorf("token invalid: %w", err)
	}

	fmt.Printf("Logged in as: %s\n", claims.Username)
	if claims.Email != "" {
		fmt.Printf("Email: %s\n", claims.Email)
	}
	if len(claims.Roles) > 0 {
		fmt.Printf("Roles: %s\n", strings.Join(claims.Roles, ", "))
	}
	if claims.Organization != "" {
		fmt.Printf("Organization: %s\n", claims.Organization)
	}
	fmt.Printf("Token expires: %s\n", claims.ExpiresAt.Time.Format(time.RFC3339))

	return nil
}

func runCreateUser(cmd *cobra.Command, args []string) error {
	username := args[0]

	// Get password interactively
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	password := string(passwordBytes)

	// Confirm password
	fmt.Print("Confirm password: ")
	confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("failed to read password confirmation: %w", err)
	}

	if password != string(confirmBytes) {
		return fmt.Errorf("passwords do not match")
	}

	// Create user
	userStore, err := auth.NewUserStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize user store: %w", err)
	}

	if err := userStore.CreateUser(username, authEmail, password, authRoles, authOrg); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("✓ User %s created successfully\n", username)
	return nil
}

func runListUsers(cmd *cobra.Command, args []string) error {
	userStore, err := auth.NewUserStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize user store: %w", err)
	}

	users := userStore.ListUsers()
	if len(users) == 0 {
		fmt.Println("No users found")
		return nil
	}

	fmt.Println("Users:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for _, user := range users {
		status := "enabled"
		if !user.Enabled {
			status = "disabled"
		}
		fmt.Printf("  %s (%s)\n", user.Username, status)
		if user.Email != "" {
			fmt.Printf("    Email: %s\n", user.Email)
		}
		fmt.Printf("    Roles: %s\n", strings.Join(user.Roles, ", "))
		if user.Organization != "" {
			fmt.Printf("    Organization: %s\n", user.Organization)
		}
		fmt.Println()
	}

	return nil
}

func runResetPassword(cmd *cobra.Command, args []string) error {
	username := args[0]

	// Get new password
	fmt.Print("New password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	password := string(passwordBytes)

	// Confirm password
	fmt.Print("Confirm password: ")
	confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("failed to read password confirmation: %w", err)
	}

	if password != string(confirmBytes) {
		return fmt.Errorf("passwords do not match")
	}

	// Update password
	userStore, err := auth.NewUserStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize user store: %w", err)
	}

	if err := userStore.UpdatePassword(username, password); err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	fmt.Printf("✓ Password reset for user %s\n", username)
	return nil
}

func runCreateAPIKey(cmd *cobra.Command, args []string) error {
	// Get current user
	tokenStore, err := auth.NewTokenStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize token store: %w", err)
	}

	tokenInfo, err := tokenStore.GetToken()
	if err != nil {
		return fmt.Errorf("not logged in: %w", err)
	}

	jwtManager := auth.NewJWTManager(getJWTSecret(), "platformfoundry.io", 24*time.Hour)
	claims, err := jwtManager.ValidateToken(tokenInfo.Token)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// Parse expiry
	var expiryDuration time.Duration
	if apiKeyExpiry != "" && apiKeyExpiry != "0" {
		expiryDuration, err = parseDuration(apiKeyExpiry)
		if err != nil {
			return fmt.Errorf("invalid expiry duration: %w", err)
		}
	}

	// Create API key
	apiKeyStore, err := auth.NewAPIKeyStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize API key store: %w", err)
	}

	rawKey, apiKey, err := apiKeyStore.CreateAPIKey(
		apiKeyName,
		claims.Username,
		claims.Roles,
		claims.Organization,
		expiryDuration,
	)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	fmt.Println("✓ API key created successfully")
	fmt.Println()
	fmt.Println("  IMPORTANT: Save this key securely. It will not be shown again.")
	fmt.Println()
	fmt.Printf("  API Key: %s\n", rawKey)
	fmt.Println()
	fmt.Printf("  Name: %s\n", apiKey.Name)
	fmt.Printf("  ID: %s\n", apiKey.ID)
	if apiKey.ExpiresAt != nil {
		fmt.Printf("  Expires: %s\n", apiKey.ExpiresAt.Format(time.RFC3339))
	} else {
		fmt.Println("  Expires: Never")
	}
	fmt.Println()
	fmt.Println("  Use it in API requests:")
	fmt.Println("    curl -H \"X-API-Key: " + rawKey + "\" https://api.platformfoundry.io/...")

	return nil
}

func runListAPIKeys(cmd *cobra.Command, args []string) error {
	// Get current user
	tokenStore, err := auth.NewTokenStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize token store: %w", err)
	}

	tokenInfo, err := tokenStore.GetToken()
	if err != nil {
		return fmt.Errorf("not logged in: %w", err)
	}

	jwtManager := auth.NewJWTManager(getJWTSecret(), "platformfoundry.io", 24*time.Hour)
	claims, err := jwtManager.ValidateToken(tokenInfo.Token)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// List API keys
	apiKeyStore, err := auth.NewAPIKeyStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize API key store: %w", err)
	}

	keys := apiKeyStore.ListAPIKeys(claims.Username)
	if len(keys) == 0 {
		fmt.Println("No API keys found")
		return nil
	}

	fmt.Println("API Keys:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for _, key := range keys {
		status := "active"
		if !key.Enabled {
			status = "revoked"
		}
		fmt.Printf("  %s (%s)\n", key.Name, status)
		fmt.Printf("    ID: %s\n", key.ID)
		fmt.Printf("    Created: %s\n", key.CreatedAt.Format(time.RFC3339))
		if key.ExpiresAt != nil {
			fmt.Printf("    Expires: %s\n", key.ExpiresAt.Format(time.RFC3339))
		}
		if key.LastUsedAt != nil {
			fmt.Printf("    Last used: %s\n", key.LastUsedAt.Format(time.RFC3339))
		}
		fmt.Println()
	}

	return nil
}

func runRevokeAPIKey(cmd *cobra.Command, args []string) error {
	keyID := args[0]

	apiKeyStore, err := auth.NewAPIKeyStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize API key store: %w", err)
	}

	if err := apiKeyStore.RevokeAPIKey(keyID); err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	fmt.Printf("✓ API key %s revoked\n", keyID)
	return nil
}

// Helper functions

func getJWTSecret() string {
	// First check environment variable (highest priority)
	if secret := os.Getenv("PF_JWT_SECRET"); secret != "" {
		return secret
	}

	// Try to load from security config file
	securityConfig, err := config.LoadSecurityConfigOrDefault()
	if err == nil && securityConfig.Auth.JWT.SecretKey != "" {
		return securityConfig.Auth.JWT.SecretKey
	}

	// Try to load from dedicated secret file
	secretPaths := []string{
		filepath.Join(os.Getenv("HOME"), ".pf", "jwt-secret"),
		filepath.Join(os.Getenv("HOME"), ".platformfoundry", "jwt-secret"),
		"/etc/platformfoundry/jwt-secret",
	}

	for _, path := range secretPaths {
		if data, err := os.ReadFile(path); err == nil {
			secret := strings.TrimSpace(string(data))
			if secret != "" {
				return secret
			}
		}
	}

	// Fallback to default (should be changed in production)
	return "change-this-secret-in-production"
}

func parseDuration(s string) (time.Duration, error) {
	// Support common formats: 30d, 1y, 90d, 1h, etc.
	if strings.HasSuffix(s, "d") {
		days := strings.TrimSuffix(s, "d")
		var n int
		_, err := fmt.Sscanf(days, "%d", &n)
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	if strings.HasSuffix(s, "y") {
		years := strings.TrimSuffix(s, "y")
		var n int
		_, err := fmt.Sscanf(years, "%d", &n)
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * 365 * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
