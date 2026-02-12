package backup

import (
	"fmt"
	"os"
	"path/filepath"
)

// BackupDir builds the backup directory path: backup/<resourceGroup>/<serviceName>[/<productID>]
func BackupDir(resourceGroup, serviceName, productID string) string {
	dir := filepath.Join("backup", resourceGroup, serviceName)
	if productID != "" {
		dir = filepath.Join(dir, productID)
	}
	return dir
}

// EnsureBackupDir creates the backup directory structure and returns the path.
func EnsureBackupDir(resourceGroup, serviceName, productID string) (string, error) {
	dir := BackupDir(resourceGroup, serviceName, productID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory %s: %w", dir, err)
	}
	return dir, nil
}