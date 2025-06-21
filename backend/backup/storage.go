package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileSystemBackupStorage implements BackupStorage using local filesystem
type FileSystemBackupStorage struct {
	basePath string
}

// NewFileSystemBackupStorage creates a new filesystem backup storage
func NewFileSystemBackupStorage(basePath string) (*FileSystemBackupStorage, error) {
	// Ensure the base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	return &FileSystemBackupStorage{
		basePath: basePath,
	}, nil
}

// Store saves backup data to storage and returns the path
func (s *FileSystemBackupStorage) Store(ctx context.Context, backupID string, data io.Reader) (string, error) {
	// Create directory structure: basePath/YYYY/MM/DD/
	backupPath := s.getBackupPath(backupID)
	
	// Ensure directory exists
	dir := filepath.Dir(backupPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	// Create backup file
	file, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()
	
	// Copy data to file
	_, err = io.Copy(file, data)
	if err != nil {
		// Clean up on error
		os.Remove(backupPath)
		return "", fmt.Errorf("failed to write backup data: %w", err)
	}
	
	return backupPath, nil
}

// Retrieve gets backup data from storage
func (s *FileSystemBackupStorage) Retrieve(ctx context.Context, backupID string) (io.ReadCloser, error) {
	backupPath := s.getBackupPath(backupID)
	
	file, err := os.Open(backupPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("backup file not found: %s", backupID)
		}
		return nil, fmt.Errorf("failed to open backup file: %w", err)
	}
	
	return file, nil
}

// Delete removes backup data from storage
func (s *FileSystemBackupStorage) Delete(ctx context.Context, backupID string) error {
	backupPath := s.getBackupPath(backupID)
	
	err := os.Remove(backupPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("backup file not found: %s", backupID)
		}
		return fmt.Errorf("failed to delete backup file: %w", err)
	}
	
	// Try to remove empty directories
	s.cleanupEmptyDirectories(filepath.Dir(backupPath))
	
	return nil
}

// Exists checks if backup data exists in storage
func (s *FileSystemBackupStorage) Exists(ctx context.Context, backupID string) (bool, error) {
	backupPath := s.getBackupPath(backupID)
	
	_, err := os.Stat(backupPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check backup file: %w", err)
	}
	
	return true, nil
}

// GetSize returns the size of backup data in storage
func (s *FileSystemBackupStorage) GetSize(ctx context.Context, backupID string) (int64, error) {
	backupPath := s.getBackupPath(backupID)
	
	stat, err := os.Stat(backupPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("backup file not found: %s", backupID)
		}
		return 0, fmt.Errorf("failed to get backup file size: %w", err)
	}
	
	return stat.Size(), nil
}

// ListBackups returns a list of backup IDs in storage
func (s *FileSystemBackupStorage) ListBackups(ctx context.Context) ([]string, error) {
	var backupIDs []string
	
	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && strings.HasSuffix(path, ".sql") {
			// Extract backup ID from filename
			filename := filepath.Base(path)
			backupID := strings.TrimSuffix(filename, ".sql")
			backupIDs = append(backupIDs, backupID)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to list backup files: %w", err)
	}
	
	return backupIDs, nil
}

// getBackupPath returns the full path for a backup file
func (s *FileSystemBackupStorage) getBackupPath(backupID string) string {
	// Simple approach: store directly in base path with .sql extension
	// In production, you might want date-based directory structure
	filename := fmt.Sprintf("%s.sql", backupID)
	return filepath.Join(s.basePath, filename)
}

// cleanupEmptyDirectories removes empty directories up the tree
func (s *FileSystemBackupStorage) cleanupEmptyDirectories(dirPath string) {
	// Don't remove the base path
	if dirPath == s.basePath {
		return
	}
	
	// Check if directory is empty
	entries, err := os.ReadDir(dirPath)
	if err != nil || len(entries) > 0 {
		return
	}
	
	// Remove empty directory
	if err := os.Remove(dirPath); err == nil {
		// Recursively check parent directory
		s.cleanupEmptyDirectories(filepath.Dir(dirPath))
	}
}

// S3BackupStorage implements BackupStorage using Amazon S3 (placeholder)
type S3BackupStorage struct {
	bucket string
	region string
	prefix string
}

// NewS3BackupStorage creates a new S3 backup storage (placeholder implementation)
func NewS3BackupStorage(bucket, region, prefix string) *S3BackupStorage {
	return &S3BackupStorage{
		bucket: bucket,
		region: region,
		prefix: prefix,
	}
}

// Store saves backup data to S3
func (s *S3BackupStorage) Store(ctx context.Context, backupID string, data io.Reader) (string, error) {
	// Placeholder - in real implementation, use AWS SDK
	return "", fmt.Errorf("S3 backup storage not implemented")
}

// Retrieve gets backup data from S3
func (s *S3BackupStorage) Retrieve(ctx context.Context, backupID string) (io.ReadCloser, error) {
	// Placeholder - in real implementation, use AWS SDK
	return nil, fmt.Errorf("S3 backup storage not implemented")
}

// Delete removes backup data from S3
func (s *S3BackupStorage) Delete(ctx context.Context, backupID string) error {
	// Placeholder - in real implementation, use AWS SDK
	return fmt.Errorf("S3 backup storage not implemented")
}

// Exists checks if backup data exists in S3
func (s *S3BackupStorage) Exists(ctx context.Context, backupID string) (bool, error) {
	// Placeholder - in real implementation, use AWS SDK
	return false, fmt.Errorf("S3 backup storage not implemented")
}

// GetSize returns the size of backup data in S3
func (s *S3BackupStorage) GetSize(ctx context.Context, backupID string) (int64, error) {
	// Placeholder - in real implementation, use AWS SDK
	return 0, fmt.Errorf("S3 backup storage not implemented")
}

// ListBackups returns a list of backup IDs in S3
func (s *S3BackupStorage) ListBackups(ctx context.Context) ([]string, error) {
	// Placeholder - in real implementation, use AWS SDK
	return nil, fmt.Errorf("S3 backup storage not implemented")
}

// GCSBackupStorage implements BackupStorage using Google Cloud Storage (placeholder)
type GCSBackupStorage struct {
	bucket string
	prefix string
}

// NewGCSBackupStorage creates a new GCS backup storage (placeholder implementation)
func NewGCSBackupStorage(bucket, prefix string) *GCSBackupStorage {
	return &GCSBackupStorage{
		bucket: bucket,
		prefix: prefix,
	}
}

// Store saves backup data to GCS
func (g *GCSBackupStorage) Store(ctx context.Context, backupID string, data io.Reader) (string, error) {
	// Placeholder - in real implementation, use Google Cloud SDK
	return "", fmt.Errorf("GCS backup storage not implemented")
}

// Retrieve gets backup data from GCS
func (g *GCSBackupStorage) Retrieve(ctx context.Context, backupID string) (io.ReadCloser, error) {
	// Placeholder - in real implementation, use Google Cloud SDK
	return nil, fmt.Errorf("GCS backup storage not implemented")
}

// Delete removes backup data from GCS
func (g *GCSBackupStorage) Delete(ctx context.Context, backupID string) error {
	// Placeholder - in real implementation, use Google Cloud SDK
	return fmt.Errorf("GCS backup storage not implemented")
}

// Exists checks if backup data exists in GCS
func (g *GCSBackupStorage) Exists(ctx context.Context, backupID string) (bool, error) {
	// Placeholder - in real implementation, use Google Cloud SDK
	return false, fmt.Errorf("GCS backup storage not implemented")
}

// GetSize returns the size of backup data in GCS
func (g *GCSBackupStorage) GetSize(ctx context.Context, backupID string) (int64, error) {
	// Placeholder - in real implementation, use Google Cloud SDK
	return 0, fmt.Errorf("GCS backup storage not implemented")
}

// ListBackups returns a list of backup IDs in GCS
func (g *GCSBackupStorage) ListBackups(ctx context.Context) ([]string, error) {
	// Placeholder - in real implementation, use Google Cloud SDK
	return nil, fmt.Errorf("GCS backup storage not implemented")
}