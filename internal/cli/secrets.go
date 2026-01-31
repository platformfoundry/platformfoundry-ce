package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/platformfoundry/platformfoundry-ce/internal/secrets"
	"github.com/spf13/cobra"
)

var (
	secretsProvider  string
	secretsData      []string
	secretsVault     string
	secretsAWSRegion string
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Secrets management",
	Long:  `Manage secrets using local, Vault, or AWS Secrets Manager.`,
}

var getSecretCmd = &cobra.Command{
	Use:   "get <path>",
	Short: "Get a secret",
	Long:  `Retrieve a secret from the secrets manager.`,
	Example: `  pf secrets get database/prod --provider vault
  pf secrets get prod/db/password --provider aws --region us-east-1
  pf secrets get api/token --provider local`,
	Args: cobra.ExactArgs(1),
	RunE: runGetSecret,
}

var putSecretCmd = &cobra.Command{
	Use:   "put <path>",
	Short: "Store a secret",
	Long:  `Store a secret in the secrets manager.`,
	Example: `  pf secrets put database/prod --data "username=admin,password=secret" --provider vault
  pf secrets put api/token --data "value=abc123" --provider local
  pf secrets put prod/db --data "host=db.example.com,port=5432" --provider aws`,
	Args: cobra.ExactArgs(1),
	RunE: runPutSecret,
}

var deleteSecretCmd = &cobra.Command{
	Use:   "delete <path>",
	Short: "Delete a secret",
	Long:  `Remove a secret from the secrets manager.`,
	Example: `  pf secrets delete database/prod --provider vault
  pf secrets delete api/token --provider local`,
	Args: cobra.ExactArgs(1),
	RunE: runDeleteSecret,
}

var listSecretsCmd = &cobra.Command{
	Use:   "list [prefix]",
	Short: "List secrets",
	Long:  `List all secrets with an optional prefix filter.`,
	Example: `  pf secrets list --provider vault
  pf secrets list database/ --provider vault
  pf secrets list prod/ --provider aws`,
	RunE: runListSecrets,
}

var resolveSecretsCmd = &cobra.Command{
	Use:   "resolve <text>",
	Short: "Resolve secret references in text",
	Long:  `Resolve secret references in text. Secret references use the format ${secret:provider:path:key}.`,
	Example: `  pf secrets resolve '${secret:vault:database/prod:password}'
  pf secrets resolve 'DB_HOST=${secret:aws:prod/db:host}'`,
	Args: cobra.ExactArgs(1),
	RunE: runResolveSecrets,
}

func init() {
	// Common flags
	getSecretCmd.Flags().StringVar(&secretsProvider, "provider", "local", "Secrets provider (local, vault, aws)")
	getSecretCmd.Flags().StringVar(&secretsVault, "vault-addr", "http://127.0.0.1:8200", "Vault address")
	getSecretCmd.Flags().StringVar(&secretsAWSRegion, "region", "us-east-1", "AWS region")

	putSecretCmd.Flags().StringVar(&secretsProvider, "provider", "local", "Secrets provider (local, vault, aws)")
	putSecretCmd.Flags().StringSliceVar(&secretsData, "data", []string{}, "Secret data as key=value pairs (comma-separated)")
	putSecretCmd.Flags().StringVar(&secretsVault, "vault-addr", "http://127.0.0.1:8200", "Vault address")
	putSecretCmd.Flags().StringVar(&secretsAWSRegion, "region", "us-east-1", "AWS region")
	putSecretCmd.MarkFlagRequired("data")

	deleteSecretCmd.Flags().StringVar(&secretsProvider, "provider", "local", "Secrets provider (local, vault, aws)")
	deleteSecretCmd.Flags().StringVar(&secretsVault, "vault-addr", "http://127.0.0.1:8200", "Vault address")
	deleteSecretCmd.Flags().StringVar(&secretsAWSRegion, "region", "us-east-1", "AWS region")

	listSecretsCmd.Flags().StringVar(&secretsProvider, "provider", "local", "Secrets provider (local, vault, aws)")
	listSecretsCmd.Flags().StringVar(&secretsVault, "vault-addr", "http://127.0.0.1:8200", "Vault address")
	listSecretsCmd.Flags().StringVar(&secretsAWSRegion, "region", "us-east-1", "AWS region")

	// Add subcommands
	secretsCmd.AddCommand(getSecretCmd)
	secretsCmd.AddCommand(putSecretCmd)
	secretsCmd.AddCommand(deleteSecretCmd)
	secretsCmd.AddCommand(listSecretsCmd)
	secretsCmd.AddCommand(resolveSecretsCmd)
}

func runGetSecret(cmd *cobra.Command, args []string) error {
	path := args[0]

	manager, err := createSecretsManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	ctx := context.Background()
	secret, err := manager.GetSecret(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	fmt.Printf("Secret: %s\n", path)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Data:")
	for key, value := range secret.Data {
		// Mask the value for security
		masked := maskSecret(value)
		fmt.Printf("  %s: %s\n", key, masked)
	}

	if len(secret.Metadata) > 0 {
		fmt.Println("\nMetadata:")
		for key, value := range secret.Metadata {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	if secret.Version > 0 {
		fmt.Printf("\nVersion: %d\n", secret.Version)
	}

	if !secret.CreatedAt.IsZero() {
		fmt.Printf("Created: %s\n", secret.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	if !secret.UpdatedAt.IsZero() {
		fmt.Printf("Updated: %s\n", secret.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func runPutSecret(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Parse data
	data := make(map[string]string)
	for _, pair := range secretsData {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid data format: %s (use key=value)", pair)
		}
		data[parts[0]] = parts[1]
	}

	if len(data) == 0 {
		return fmt.Errorf("no data provided")
	}

	manager, err := createSecretsManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	ctx := context.Background()
	if err := manager.PutSecret(ctx, path, data); err != nil {
		return fmt.Errorf("failed to store secret: %w", err)
	}

	fmt.Printf("✓ Secret stored: %s\n", path)
	fmt.Printf("  Keys: %s\n", strings.Join(getKeys(data), ", "))

	return nil
}

func runDeleteSecret(cmd *cobra.Command, args []string) error {
	path := args[0]

	manager, err := createSecretsManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	ctx := context.Background()
	if err := manager.DeleteSecret(ctx, path); err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	fmt.Printf("✓ Secret deleted: %s\n", path)
	return nil
}

func runListSecrets(cmd *cobra.Command, args []string) error {
	prefix := ""
	if len(args) > 0 {
		prefix = args[0]
	}

	manager, err := createSecretsManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	ctx := context.Background()
	paths, err := manager.ListSecrets(ctx, prefix)
	if err != nil {
		return fmt.Errorf("failed to list secrets: %w", err)
	}

	if len(paths) == 0 {
		fmt.Println("No secrets found")
		return nil
	}

	fmt.Printf("Secrets (%d):\n", len(paths))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for _, path := range paths {
		fmt.Printf("  %s\n", path)
	}

	return nil
}

func runResolveSecrets(cmd *cobra.Command, args []string) error {
	text := args[0]

	// Find all secret references
	refs := secrets.FindSecretReferences(text)
	if len(refs) == 0 {
		fmt.Println("No secret references found")
		fmt.Printf("Input: %s\n", text)
		return nil
	}

	fmt.Printf("Found %d secret reference(s):\n", len(refs))
	for i, ref := range refs {
		fmt.Printf("  %d. %s (provider=%s, path=%s, key=%s)\n", i+1, ref.Raw, ref.Provider, ref.Path, ref.Key)
	}
	fmt.Println()

	// Create managers for each provider
	managers := make(map[string]secrets.Manager)
	defer func() {
		for _, mgr := range managers {
			mgr.Close()
		}
	}()

	ctx := context.Background()
	resolved := text

	// Resolve each reference
	for _, ref := range refs {
		// Get or create manager for this provider
		manager, exists := managers[ref.Provider]
		if !exists {
			var err error
			manager, err = createSecretsManagerForProvider(ref.Provider)
			if err != nil {
				return fmt.Errorf("failed to create manager for provider %s: %w", ref.Provider, err)
			}
			managers[ref.Provider] = manager
		}

		// Get secret
		secret, err := manager.GetSecret(ctx, ref.Path)
		if err != nil {
			return fmt.Errorf("failed to resolve %s: %w", ref.Raw, err)
		}

		// Get value
		value, ok := secret.Data[ref.Key]
		if !ok {
			return fmt.Errorf("secret %s does not contain key %s", ref.Path, ref.Key)
		}

		// Replace in text
		resolved = strings.ReplaceAll(resolved, ref.Raw, value)
	}

	fmt.Println("Resolved:")
	fmt.Printf("  %s\n", resolved)

	return nil
}

// Helper functions

func createSecretsManager() (secrets.Manager, error) {
	return createSecretsManagerForProvider(secretsProvider)
}

func createSecretsManagerForProvider(provider string) (secrets.Manager, error) {
	config := &secrets.Config{
		Provider: provider,
	}

	switch provider {
	case "vault":
		config.Vault = &secrets.VaultConfig{
			Address: secretsVault,
		}
	case "aws":
		config.AWS = &secrets.AWSConfig{
			Region: secretsAWSRegion,
		}
	case "local":
		config.Local = &secrets.LocalConfig{}
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	return secrets.NewManager(config)
}

func maskSecret(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

func getKeys(data map[string]string) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}
