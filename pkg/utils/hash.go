package utils

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
)

// CalculateFileMD5 计算文件内容的MD5哈希值
func CalculateFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for MD5 calculation: %w", err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate MD5 hash: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// CalculatePathMD5 计算文件路径的MD5哈希值（用于创建一致的临时目录名）
func CalculatePathMD5(filePath string) string {
	hash := md5.New()
	hash.Write([]byte(filePath))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
