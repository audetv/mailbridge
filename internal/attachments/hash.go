// Package attachments предоставляет утилиты для работы с файлами.
package attachments

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ComputeHash вычисляет SHA-256 файла.
func ComputeHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// HashToPath преобразует hash в путь: {hash[0:2]}/{hash[2:4]}/{hash}
func HashToPath(hash string) string {
	if len(hash) < 4 {
		return hash
	}
	return filepath.Join(hash[0:2], hash[2:4], hash)
}

// StoragePathForHash возвращает полный путь для хранения файла с hash.
func StoragePathForHash(hash string) string {
	return HashToPath(hash)
}
