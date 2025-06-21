package config

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupManager handles configuration backups
type BackupManager struct {
	backupDir       string
	maxBackups      int
	retentionDays   int
	encryptionKey   []byte
	compressBackups bool
}

// NewBackupManager creates a new backup manager
func NewBackupManager(backupDir string) *BackupManager {
	return &BackupManager{
		backupDir:       backupDir,
		maxBackups:      50,
		retentionDays:   30,
		compressBackups: true,
	}
}

// CreateBackup creates a new configuration backup
func (bm *BackupManager) CreateBackup(config *Config, name, description, createdBy string) (*ConfigBackup, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(bm.backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	// Generate backup ID
	backupID := bm.generateBackupID()
	
	// Create backup metadata
	backup := &ConfigBackup{
		ID:          backupID,
		Name:        name,
		Description: description,
		Config:      config,
		CreatedAt:   time.Now(),
		CreatedBy:   createdBy,
		Version:     "1.0.0",
		Compressed:  bm.compressBackups,
		Encrypted:   bm.encryptionKey != nil,
		Tags:        []string{},
		Metadata:    make(map[string]string),
	}
	
	// Serialize configuration
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize config: %w", err)
	}
	
	// Calculate checksum before compression/encryption
	backup.Checksum = bm.calculateChecksum(configData)
	
	// Compress if enabled
	if bm.compressBackups {
		configData, err = bm.compressData(configData)
		if err != nil {
			return nil, fmt.Errorf("failed to compress backup: %w", err)
		}
	}
	
	// Encrypt if key is set
	if bm.encryptionKey != nil {
		configData, err = bm.encryptData(configData)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt backup: %w", err)
		}
	}
	
	backup.Size = int64(len(configData))
	
	// Save backup data
	dataFile := filepath.Join(bm.backupDir, fmt.Sprintf("%s.data", backupID))
	if err := ioutil.WriteFile(dataFile, configData, 0600); err != nil {
		return nil, fmt.Errorf("failed to write backup data: %w", err)
	}
	
	// Save backup metadata
	metaFile := filepath.Join(bm.backupDir, fmt.Sprintf("%s.meta", backupID))
	metaData, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		os.Remove(dataFile) // Clean up data file
		return nil, fmt.Errorf("failed to serialize backup metadata: %w", err)
	}
	
	if err := ioutil.WriteFile(metaFile, metaData, 0644); err != nil {
		os.Remove(dataFile) // Clean up data file
		return nil, fmt.Errorf("failed to write backup metadata: %w", err)
	}
	
	// Clean up old backups
	if err := bm.cleanupOldBackups(); err != nil {
		// Log error but don't fail the backup
		fmt.Printf("Warning: failed to cleanup old backups: %v\n", err)
	}
	
	return backup, nil
}

// ListBackups lists all available backups
func (bm *BackupManager) ListBackups() ([]*ConfigBackup, error) {
	files, err := ioutil.ReadDir(bm.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*ConfigBackup{}, nil
		}
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}
	
	backups := []*ConfigBackup{}
	
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".meta") {
			metaData, err := ioutil.ReadFile(filepath.Join(bm.backupDir, file.Name()))
			if err != nil {
				continue
			}
			
			var backup ConfigBackup
			if err := json.Unmarshal(metaData, &backup); err != nil {
				continue
			}
			
			// Don't include the full config in the list
			backup.Config = nil
			backups = append(backups, &backup)
		}
	}
	
	// Sort by creation date (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})
	
	return backups, nil
}

// GetBackup retrieves a specific backup
func (bm *BackupManager) GetBackup(backupID string) (*ConfigBackup, error) {
	// Read metadata
	metaFile := filepath.Join(bm.backupDir, fmt.Sprintf("%s.meta", backupID))
	metaData, err := ioutil.ReadFile(metaFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("backup not found: %s", backupID)
		}
		return nil, fmt.Errorf("failed to read backup metadata: %w", err)
	}
	
	var backup ConfigBackup
	if err := json.Unmarshal(metaData, &backup); err != nil {
		return nil, fmt.Errorf("failed to parse backup metadata: %w", err)
	}
	
	// Read data
	dataFile := filepath.Join(bm.backupDir, fmt.Sprintf("%s.data", backupID))
	configData, err := ioutil.ReadFile(dataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup data: %w", err)
	}
	
	// Decrypt if needed
	if backup.Encrypted {
		if bm.encryptionKey == nil {
			return nil, fmt.Errorf("backup is encrypted but no encryption key is set")
		}
		configData, err = bm.decryptData(configData)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt backup: %w", err)
		}
	}
	
	// Decompress if needed
	if backup.Compressed {
		configData, err = bm.decompressData(configData)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress backup: %w", err)
		}
	}
	
	// Verify checksum
	checksum := bm.calculateChecksum(configData)
	if checksum != backup.Checksum {
		return nil, fmt.Errorf("backup checksum mismatch: expected %s, got %s", backup.Checksum, checksum)
	}
	
	// Parse configuration
	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse backup configuration: %w", err)
	}
	
	backup.Config = &config
	
	return &backup, nil
}

// DeleteBackup deletes a specific backup
func (bm *BackupManager) DeleteBackup(backupID string) error {
	// Remove data file
	dataFile := filepath.Join(bm.backupDir, fmt.Sprintf("%s.data", backupID))
	if err := os.Remove(dataFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove backup data: %w", err)
	}
	
	// Remove metadata file
	metaFile := filepath.Join(bm.backupDir, fmt.Sprintf("%s.meta", backupID))
	if err := os.Remove(metaFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove backup metadata: %w", err)
	}
	
	return nil
}

// ExportBackup exports a backup to a single file
func (bm *BackupManager) ExportBackup(backupID string, exportPath string) error {
	backup, err := bm.GetBackup(backupID)
	if err != nil {
		return err
	}
	
	// Create export data
	exportData := map[string]interface{}{
		"backup":  backup,
		"version": "1.0",
		"exported_at": time.Now(),
		"format": "qt1-backup",
	}
	
	// Serialize export data
	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize export data: %w", err)
	}
	
	// Compress the export
	if bm.compressBackups {
		data, err = bm.compressData(data)
		if err != nil {
			return fmt.Errorf("failed to compress export: %w", err)
		}
	}
	
	// Write to file
	if err := ioutil.WriteFile(exportPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}
	
	return nil
}

// ImportBackup imports a backup from an exported file
func (bm *BackupManager) ImportBackup(importPath string) (*ConfigBackup, error) {
	// Read import file
	data, err := ioutil.ReadFile(importPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read import file: %w", err)
	}
	
	// Try to decompress (will fail gracefully if not compressed)
	if decompressed, err := bm.decompressData(data); err == nil {
		data = decompressed
	}
	
	// Parse import data
	var importData map[string]interface{}
	if err := json.Unmarshal(data, &importData); err != nil {
		return nil, fmt.Errorf("failed to parse import data: %w", err)
	}
	
	// Verify format
	if format, ok := importData["format"].(string); !ok || format != "qt1-backup" {
		return nil, fmt.Errorf("invalid import format")
	}
	
	// Extract backup data
	backupData, ok := importData["backup"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing backup data in import")
	}
	
	// Convert to ConfigBackup
	backupJSON, err := json.Marshal(backupData)
	if err != nil {
		return nil, fmt.Errorf("failed to process backup data: %w", err)
	}
	
	var backup ConfigBackup
	if err := json.Unmarshal(backupJSON, &backup); err != nil {
		return nil, fmt.Errorf("failed to parse backup: %w", err)
	}
	
	// Generate new backup ID for imported backup
	backup.ID = bm.generateBackupID()
	backup.Metadata["imported_from"] = filepath.Base(importPath)
	backup.Metadata["imported_at"] = time.Now().Format(time.RFC3339)
	
	// Save the imported backup
	return bm.saveImportedBackup(&backup)
}

// cleanupOldBackups removes old backups based on retention policy
func (bm *BackupManager) cleanupOldBackups() error {
	backups, err := bm.ListBackups()
	if err != nil {
		return err
	}
	
	now := time.Now()
	deletedCount := 0
	
	// Remove backups older than retention period
	for _, backup := range backups {
		age := now.Sub(backup.CreatedAt)
		if age > time.Duration(bm.retentionDays)*24*time.Hour {
			if err := bm.DeleteBackup(backup.ID); err != nil {
				fmt.Printf("Warning: failed to delete old backup %s: %v\n", backup.ID, err)
			} else {
				deletedCount++
			}
		}
	}
	
	// Remove excess backups if over limit
	if len(backups)-deletedCount > bm.maxBackups {
		// Backups are already sorted by date (newest first)
		for i := bm.maxBackups; i < len(backups)-deletedCount; i++ {
			if err := bm.DeleteBackup(backups[i].ID); err != nil {
				fmt.Printf("Warning: failed to delete excess backup %s: %v\n", backups[i].ID, err)
			}
		}
	}
	
	return nil
}

// saveImportedBackup saves an imported backup
func (bm *BackupManager) saveImportedBackup(backup *ConfigBackup) (*ConfigBackup, error) {
	if backup.Config == nil {
		return nil, fmt.Errorf("backup has no configuration data")
	}
	
	// Use CreateBackup to ensure proper handling
	return bm.CreateBackup(
		backup.Config,
		backup.Name+" (imported)",
		backup.Description,
		backup.CreatedBy,
	)
}

// generateBackupID generates a unique backup ID
func (bm *BackupManager) generateBackupID() string {
	timestamp := time.Now().Format("20060102-150405")
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomStr := base64.URLEncoding.EncodeToString(randomBytes)[:6]
	return fmt.Sprintf("backup-%s-%s", timestamp, randomStr)
}

// calculateChecksum calculates SHA256 checksum
func (bm *BackupManager) calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// compressData compresses data using gzip
func (bm *BackupManager) compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	
	if err := writer.Close(); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// decompressData decompresses gzip data
func (bm *BackupManager) decompressData(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	
	return ioutil.ReadAll(reader)
}

// encryptData encrypts data using AES
func (bm *BackupManager) encryptData(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(bm.encryptionKey)
	if err != nil {
		return nil, err
	}
	
	// Create GCM cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	
	// Encrypt data
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	
	return ciphertext, nil
}

// decryptData decrypts AES encrypted data
func (bm *BackupManager) decryptData(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(bm.encryptionKey)
	if err != nil {
		return nil, err
	}
	
	// Create GCM cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	
	// Extract nonce and ciphertext
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	
	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	
	return plaintext, nil
}

// SetEncryptionKey sets the encryption key for backups
func (bm *BackupManager) SetEncryptionKey(key string) error {
	if key == "" {
		bm.encryptionKey = nil
		return nil
	}
	
	// Hash the key to ensure it's the right length
	hash := sha256.Sum256([]byte(key))
	bm.encryptionKey = hash[:]
	
	return nil
}

// SetRetentionPolicy sets the retention policy
func (bm *BackupManager) SetRetentionPolicy(maxBackups, retentionDays int) {
	if maxBackups > 0 {
		bm.maxBackups = maxBackups
	}
	if retentionDays > 0 {
		bm.retentionDays = retentionDays
	}
}

// SetCompression enables or disables compression
func (bm *BackupManager) SetCompression(enabled bool) {
	bm.compressBackups = enabled
}

// VerifyBackup verifies backup integrity
func (bm *BackupManager) VerifyBackup(backupID string) error {
	backup, err := bm.GetBackup(backupID)
	if err != nil {
		return fmt.Errorf("failed to load backup: %w", err)
	}
	
	// Validate the configuration
	if backup.Config == nil {
		return fmt.Errorf("backup has no configuration data")
	}
	
	// Additional validation could be performed here
	
	return nil
}

// CreateScheduledBackup creates a backup with scheduling metadata
func (bm *BackupManager) CreateScheduledBackup(config *Config, schedule string) (*ConfigBackup, error) {
	name := fmt.Sprintf("Scheduled backup (%s)", schedule)
	description := fmt.Sprintf("Automated backup created by schedule: %s", schedule)
	
	backup, err := bm.CreateBackup(config, name, description, "system")
	if err != nil {
		return nil, err
	}
	
	// Add scheduling metadata
	backup.Tags = append(backup.Tags, "scheduled", schedule)
	backup.Metadata["schedule"] = schedule
	backup.Metadata["next_backup"] = calculateNextBackupTime(schedule).Format(time.RFC3339)
	
	// Update metadata file
	metaFile := filepath.Join(bm.backupDir, fmt.Sprintf("%s.meta", backup.ID))
	metaData, _ := json.MarshalIndent(backup, "", "  ")
	ioutil.WriteFile(metaFile, metaData, 0644)
	
	return backup, nil
}

// calculateNextBackupTime calculates the next backup time based on schedule
func calculateNextBackupTime(schedule string) time.Time {
	now := time.Now()
	
	switch schedule {
	case "hourly":
		return now.Add(time.Hour)
	case "daily":
		return now.Add(24 * time.Hour)
	case "weekly":
		return now.Add(7 * 24 * time.Hour)
	case "monthly":
		return now.AddDate(0, 1, 0)
	default:
		// Default to daily
		return now.Add(24 * time.Hour)
	}
}