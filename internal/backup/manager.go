package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Manager handles backup and restore operations
type Manager struct {
	backupDir   string
	stateDir    string
	maxBackups  int
	compression bool
}

// Config represents backup configuration
type Config struct {
	BackupDir   string `yaml:"backupDir" json:"backupDir"`
	StateDir    string `yaml:"stateDir" json:"stateDir"`
	MaxBackups  int    `yaml:"maxBackups" json:"maxBackups"`
	Compression bool   `yaml:"compression" json:"compression"`
}

// Metadata represents backup metadata
type Metadata struct {
	Version     string    `json:"version"`
	Timestamp   time.Time `json:"timestamp"`
	Source      string    `json:"source"`
	Description string    `json:"description,omitempty"`
	Size        int64     `json:"size"`
	Compressed  bool      `json:"compressed"`
	Files       []string  `json:"files"`
}

// DefaultConfig returns default backup configuration
func DefaultConfig() *Config {
	return &Config{
		BackupDir:   "/var/lib/platformfoundry/backups",
		StateDir:    "/var/lib/platformfoundry/state",
		MaxBackups:  10,
		Compression: true,
	}
}

// NewManager creates a new backup manager
func NewManager(config *Config) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(config.BackupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	return &Manager{
		backupDir:   config.BackupDir,
		stateDir:    config.StateDir,
		maxBackups:  config.MaxBackups,
		compression: config.Compression,
	}, nil
}

// CreateBackup creates a new backup
func (m *Manager) CreateBackup(ctx context.Context, description string) (string, error) {
	timestamp := time.Now()
	backupName := fmt.Sprintf("backup-%s.tar", timestamp.Format("20060102-150405"))
	if m.compression {
		backupName += ".gz"
	}

	backupPath := filepath.Join(m.backupDir, backupName)

	// Create backup file
	file, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	// Create writer chain
	var writer io.Writer = file
	var gzWriter *gzip.Writer

	if m.compression {
		gzWriter = gzip.NewWriter(file)
		writer = gzWriter
		defer gzWriter.Close()
	}

	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	// Collect files to backup
	files := make([]string, 0)
	var totalSize int64

	// Walk state directory
	err = filepath.Walk(m.stateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(m.stateDir, path)
		if err != nil {
			return err
		}

		files = append(files, relPath)
		totalSize += info.Size()

		// Add file to tar
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Copy file contents
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		if _, err := io.Copy(tarWriter, srcFile); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		os.Remove(backupPath)
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	// Close writers to flush
	tarWriter.Close()
	if gzWriter != nil {
		gzWriter.Close()
	}
	file.Close()

	// Create metadata
	metadata := &Metadata{
		Version:     "1.0",
		Timestamp:   timestamp,
		Source:      m.stateDir,
		Description: description,
		Size:        totalSize,
		Compressed:  m.compression,
		Files:       files,
	}

	metadataPath := backupPath + ".json"
	if err := m.saveMetadata(metadataPath, metadata); err != nil {
		// Non-fatal, just log
		fmt.Printf("Warning: failed to save metadata: %v\n", err)
	}

	// Cleanup old backups
	if err := m.cleanupOldBackups(); err != nil {
		// Non-fatal, just log
		fmt.Printf("Warning: failed to cleanup old backups: %v\n", err)
	}

	return backupPath, nil
}

// RestoreBackup restores from a backup
func (m *Manager) RestoreBackup(ctx context.Context, backupPath string) error {
	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	// Open backup file
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	// Create reader chain
	var reader io.Reader = file

	// Check if compressed
	if filepath.Ext(backupPath) == ".gz" {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	tarReader := tar.NewReader(reader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Create target path
		targetPath := filepath.Join(m.stateDir, header.Name)

		// Ensure parent directory exists
		parentDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Extract file
		targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		if _, err := io.Copy(targetFile, tarReader); err != nil {
			targetFile.Close()
			return fmt.Errorf("failed to extract file: %w", err)
		}

		targetFile.Close()
	}

	return nil
}

// ListBackups lists all available backups
func (m *Manager) ListBackups() ([]*Metadata, error) {
	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	backups := make([]*Metadata, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Check for metadata file
		if filepath.Ext(name) != ".json" && (filepath.Ext(name) == ".tar" || filepath.Ext(filepath.Base(name[:len(name)-3])) == ".tar") {
			metadataPath := filepath.Join(m.backupDir, name+".json")
			if _, err := os.Stat(metadataPath); err == nil {
				metadata, err := m.loadMetadata(metadataPath)
				if err == nil {
					backups = append(backups, metadata)
				}
			} else {
				// Create minimal metadata
				info, err := entry.Info()
				if err != nil {
					continue
				}

				backups = append(backups, &Metadata{
					Timestamp:  info.ModTime(),
					Size:       info.Size(),
					Compressed: filepath.Ext(name) == ".gz",
				})
			}
		}
	}

	return backups, nil
}

// DeleteBackup deletes a backup
func (m *Manager) DeleteBackup(backupPath string) error {
	// Delete backup file
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	// Delete metadata file
	metadataPath := backupPath + ".json"
	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		// Non-fatal
		fmt.Printf("Warning: failed to delete metadata: %v\n", err)
	}

	return nil
}

// cleanupOldBackups removes old backups exceeding maxBackups
func (m *Manager) cleanupOldBackups() error {
	if m.maxBackups <= 0 {
		return nil
	}

	backups, err := m.ListBackups()
	if err != nil {
		return err
	}

	// Sort by timestamp (oldest first)
	for i := 0; i < len(backups); i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[j].Timestamp.Before(backups[i].Timestamp) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	// Delete oldest backups
	if len(backups) > m.maxBackups {
		for i := 0; i < len(backups)-m.maxBackups; i++ {
			backupName := fmt.Sprintf("backup-%s.tar", backups[i].Timestamp.Format("20060102-150405"))
			if backups[i].Compressed {
				backupName += ".gz"
			}
			backupPath := filepath.Join(m.backupDir, backupName)
			m.DeleteBackup(backupPath)
		}
	}

	return nil
}

// saveMetadata saves backup metadata
func (m *Manager) saveMetadata(path string, metadata *Metadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// loadMetadata loads backup metadata
func (m *Manager) loadMetadata(path string) (*Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// VerifyBackup verifies backup integrity
func (m *Manager) VerifyBackup(backupPath string) error {
	// Open backup file
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	// Create reader chain
	var reader io.Reader = file

	if filepath.Ext(backupPath) == ".gz" {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	tarReader := tar.NewReader(reader)

	// Read through all entries
	for {
		_, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("backup verification failed: %w", err)
		}

		// Read and discard contents to verify integrity
		if _, err := io.Copy(io.Discard, tarReader); err != nil {
			return fmt.Errorf("backup verification failed: %w", err)
		}
	}

	return nil
}
