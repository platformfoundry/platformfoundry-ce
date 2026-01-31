package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/platformfoundry/platformfoundry-ce/internal/backup"
	"github.com/spf13/cobra"
)

var (
	backupDir         string
	backupStateDir    string
	backupDescription string
	backupMaxBackups  int
	backupCompress    bool
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup and restore operations",
	Long:  `Manage backups for disaster recovery and state preservation.`,
}

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new backup",
	Long:  `Create a backup of the current platform state.`,
	Example: `  pf backup create --description "Pre-upgrade backup"
  pf backup create --dir /custom/backup/dir
  pf backup create --no-compress`,
	RunE: runBackupCreate,
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <backup-file>",
	Short: "Restore from a backup",
	Long:  `Restore platform state from a backup file.`,
	Example: `  pf backup restore /var/lib/platformfoundry/backups/backup-20240115-120000.tar.gz
  pf backup restore backup-20240115-120000.tar.gz`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: backupIDCompletion,
	RunE:              runBackupRestore,
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups",
	Long:  `List all available backups with metadata.`,
	Example: `  pf backup list
  pf backup list --dir /custom/backup/dir`,
	RunE: runBackupList,
}

var backupDeleteCmd = &cobra.Command{
	Use:   "delete <backup-file>",
	Short: "Delete a backup",
	Long:  `Delete a specific backup file.`,
	Example: `  pf backup delete backup-20240115-120000.tar.gz
  pf backup delete /var/lib/platformfoundry/backups/backup-20240115-120000.tar.gz`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: backupIDCompletion,
	RunE:              runBackupDelete,
}

var backupVerifyCmd = &cobra.Command{
	Use:   "verify <backup-file>",
	Short: "Verify backup integrity",
	Long:  `Verify the integrity of a backup file.`,
	Example: `  pf backup verify backup-20240115-120000.tar.gz
  pf backup verify /var/lib/platformfoundry/backups/backup-20240115-120000.tar.gz`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: backupIDCompletion,
	RunE:              runBackupVerify,
}

func init() {
	// Create command flags
	backupCreateCmd.Flags().StringVar(&backupDir, "dir", "/var/lib/platformfoundry/backups", "Backup directory")
	backupCreateCmd.Flags().StringVar(&backupStateDir, "state-dir", "/var/lib/platformfoundry/state", "State directory to backup")
	backupCreateCmd.Flags().StringVar(&backupDescription, "description", "", "Backup description")
	backupCreateCmd.Flags().BoolVar(&backupCompress, "compress", true, "Compress backup (gzip)")
	backupCreateCmd.Flags().IntVar(&backupMaxBackups, "max-backups", 10, "Maximum number of backups to keep")

	// Restore command flags
	backupRestoreCmd.Flags().StringVar(&backupDir, "dir", "/var/lib/platformfoundry/backups", "Backup directory")
	backupRestoreCmd.Flags().StringVar(&backupStateDir, "state-dir", "/var/lib/platformfoundry/state", "State directory to restore to")

	// List command flags
	backupListCmd.Flags().StringVar(&backupDir, "dir", "/var/lib/platformfoundry/backups", "Backup directory")

	// Delete command flags
	backupDeleteCmd.Flags().StringVar(&backupDir, "dir", "/var/lib/platformfoundry/backups", "Backup directory")

	// Verify command flags
	backupVerifyCmd.Flags().StringVar(&backupDir, "dir", "/var/lib/platformfoundry/backups", "Backup directory")

	// Add subcommands
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupDeleteCmd)
	backupCmd.AddCommand(backupVerifyCmd)
}

func runBackupCreate(cmd *cobra.Command, args []string) error {
	config := &backup.Config{
		BackupDir:   backupDir,
		StateDir:    backupStateDir,
		MaxBackups:  backupMaxBackups,
		Compression: backupCompress,
	}

	manager, err := backup.NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create backup manager: %w", err)
	}

	fmt.Println("Creating backup...")
	ctx := context.Background()

	backupPath, err := manager.CreateBackup(ctx, backupDescription)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	fmt.Println("✓ Backup created successfully!")
	fmt.Printf("  Location: %s\n", backupPath)

	if backupDescription != "" {
		fmt.Printf("  Description: %s\n", backupDescription)
	}

	return nil
}

func runBackupRestore(cmd *cobra.Command, args []string) error {
	backupFile := args[0]

	// If not an absolute path, prepend backup directory
	if !filepath.IsAbs(backupFile) {
		backupFile = filepath.Join(backupDir, backupFile)
	}

	config := &backup.Config{
		BackupDir:   backupDir,
		StateDir:    backupStateDir,
		Compression: backupCompress,
	}

	manager, err := backup.NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create backup manager: %w", err)
	}

	// Confirm before restoring
	fmt.Printf("⚠ WARNING: This will restore state from backup:\n")
	fmt.Printf("  Backup: %s\n", backupFile)
	fmt.Printf("  Target: %s\n\n", backupStateDir)
	fmt.Print("Do you want to continue? (yes/no): ")

	var response string
	fmt.Scanln(&response)

	if response != "yes" && response != "y" {
		fmt.Println("Restore cancelled")
		return nil
	}

	fmt.Println("\nRestoring from backup...")
	ctx := context.Background()

	if err := manager.RestoreBackup(ctx, backupFile); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	fmt.Println("✓ Backup restored successfully!")
	fmt.Printf("  State directory: %s\n", backupStateDir)

	return nil
}

func runBackupList(cmd *cobra.Command, args []string) error {
	config := &backup.Config{
		BackupDir: backupDir,
	}

	manager, err := backup.NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create backup manager: %w", err)
	}

	backups, err := manager.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		fmt.Println("No backups found")
		return nil
	}

	fmt.Printf("Available Backups (%d):\n", len(backups))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i, b := range backups {
		backupName := fmt.Sprintf("backup-%s.tar", b.Timestamp.Format("20060102-150405"))
		if b.Compressed {
			backupName += ".gz"
		}

		fmt.Printf("%d. %s\n", i+1, backupName)
		fmt.Printf("   Timestamp: %s\n", b.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Size: %s\n", formatSize(b.Size))
		fmt.Printf("   Compressed: %v\n", b.Compressed)

		if b.Description != "" {
			fmt.Printf("   Description: %s\n", b.Description)
		}

		if len(b.Files) > 0 {
			fmt.Printf("   Files: %d\n", len(b.Files))
		}

		if i < len(backups)-1 {
			fmt.Println()
		}
	}

	return nil
}

func runBackupDelete(cmd *cobra.Command, args []string) error {
	backupFile := args[0]

	// If not an absolute path, prepend backup directory
	if !filepath.IsAbs(backupFile) {
		backupFile = filepath.Join(backupDir, backupFile)
	}

	config := &backup.Config{
		BackupDir: backupDir,
	}

	manager, err := backup.NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create backup manager: %w", err)
	}

	// Confirm before deleting
	fmt.Printf("⚠ WARNING: This will permanently delete the backup:\n")
	fmt.Printf("  Backup: %s\n\n", backupFile)
	fmt.Print("Do you want to continue? (yes/no): ")

	var response string
	fmt.Scanln(&response)

	if response != "yes" && response != "y" {
		fmt.Println("Delete cancelled")
		return nil
	}

	if err := manager.DeleteBackup(backupFile); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	fmt.Println("✓ Backup deleted successfully!")

	return nil
}

func runBackupVerify(cmd *cobra.Command, args []string) error {
	backupFile := args[0]

	// If not an absolute path, prepend backup directory
	if !filepath.IsAbs(backupFile) {
		backupFile = filepath.Join(backupDir, backupFile)
	}

	config := &backup.Config{
		BackupDir: backupDir,
	}

	manager, err := backup.NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create backup manager: %w", err)
	}

	fmt.Printf("Verifying backup: %s\n", backupFile)

	if err := manager.VerifyBackup(backupFile); err != nil {
		fmt.Println("✗ Backup verification failed!")
		return fmt.Errorf("verification failed: %w", err)
	}

	fmt.Println("✓ Backup verified successfully!")
	fmt.Println("  Integrity: OK")

	return nil
}

// Helper functions

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
