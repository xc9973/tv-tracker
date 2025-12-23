package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupService handles database backup operations
// Requirements: 11.1, 11.2, 11.3
type BackupService struct {
	dbPath     string
	backupDir  string
	maxBackups int
}

// NewBackupService creates a new BackupService
func NewBackupService(dbPath, backupDir string) *BackupService {
	return &BackupService{
		dbPath:     dbPath,
		backupDir:  backupDir,
		maxBackups: 4, // Keep last 4 weekly backups
	}
}

// Backup creates a backup of the database
// Requirements: 11.1, 11.2
func (b *BackupService) Backup() (string, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(b.backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("2006-01-02_150405")
	backupName := fmt.Sprintf("tv_tracker_backup_%s.db", timestamp)
	backupPath := filepath.Join(b.backupDir, backupName)

	// Copy database file
	if err := copyFile(b.dbPath, backupPath); err != nil {
		return "", fmt.Errorf("failed to copy database: %w", err)
	}

	// Clean old backups
	if err := b.CleanOldBackups(); err != nil {
		// Log but don't fail - backup was successful
		fmt.Printf("Warning: failed to clean old backups: %v\n", err)
	}

	return backupPath, nil
}

// GetLastBackupTime returns the time of the most recent backup
// Requirements: 11.4
func (b *BackupService) GetLastBackupTime() (time.Time, error) {
	backups, err := b.listBackups()
	if err != nil {
		return time.Time{}, err
	}

	if len(backups) == 0 {
		return time.Time{}, nil
	}

	// Get the most recent backup
	latestBackup := backups[len(backups)-1]
	info, err := os.Stat(latestBackup)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to stat backup file: %w", err)
	}

	return info.ModTime(), nil
}


// CleanOldBackups removes old backups, keeping only the most recent ones
// Requirements: 11.3
func (b *BackupService) CleanOldBackups() error {
	backups, err := b.listBackups()
	if err != nil {
		return err
	}

	// If we have more backups than maxBackups, delete the oldest ones
	if len(backups) > b.maxBackups {
		toDelete := backups[:len(backups)-b.maxBackups]
		for _, backup := range toDelete {
			if err := os.Remove(backup); err != nil {
				return fmt.Errorf("failed to delete old backup %s: %w", backup, err)
			}
		}
	}

	return nil
}

// listBackups returns a sorted list of backup files (oldest first)
func (b *BackupService) listBackups() ([]string, error) {
	if _, err := os.Stat(b.backupDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(b.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "tv_tracker_backup_") && strings.HasSuffix(entry.Name(), ".db") {
			backups = append(backups, filepath.Join(b.backupDir, entry.Name()))
		}
	}

	// Sort by filename (which includes timestamp, so oldest first)
	sort.Strings(backups)

	return backups, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}
