package utils

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"doc-to-text/pkg/constants"
	"doc-to-text/pkg/types"
)

// NormalizePath standardizes file paths
func NormalizePath(path string) string {
	return filepath.Clean(path)
}

// EnsureDir creates directory if it doesn't exist
func EnsureDir(dirPath string) error {
	if dirPath == "" {
		return fmt.Errorf("directory path cannot be empty")
	}
	return os.MkdirAll(dirPath, constants.DefaultDirPermission)
}

// IsCommandAvailable checks if a command is available in PATH
func IsCommandAvailable(command string) bool {
	var cmd string
	if constants.IsWindows() {
		cmd = "where"
	} else {
		cmd = "which"
	}

	err := exec.Command(cmd, command).Run()
	return err == nil
}

// ValidatePath validates file path
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if len(path) > 255 {
		return fmt.Errorf("path too long (max 255 characters)")
	}
	if runtime.GOOS == "windows" {
		invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
		for _, char := range invalidChars {
			if strings.Contains(path, char) {
				return fmt.Errorf("path contains invalid character: %s", char)
			}
		}
	}
	return nil
}

// SanitizeFileName cleans filename for cross-platform compatibility
func SanitizeFileName(filename string) string {
	if runtime.GOOS == "windows" {
		re := regexp.MustCompile(`[<>:"/\\|?*]`)
		filename = re.ReplaceAllString(filename, "_")
	} else {
		re := regexp.MustCompile(`[/\x00]`)
		filename = re.ReplaceAllString(filename, "_")
	}
	filename = strings.TrimSpace(filename)
	if len(filename) > 250 {
		filename = filename[:250]
	}
	return filename
}

// GetFileInfo gets comprehensive file information
func GetFileInfo(filePath string) (*types.FileInfo, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}

	md5Hash, err := CalculateFileMD5(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}

	extension := strings.ToLower(filepath.Ext(filePath))
	if extension != "" && extension[0] == '.' {
		extension = extension[1:]
	}

	mimeType, err := getMimeType(filePath)
	if err != nil {
		mimeType = "application/octet-stream"
	}

	return &types.FileInfo{
		MD5Hash:   md5Hash,
		Extension: extension,
		MimeType:  mimeType,
		Size:      stat.Size(),
		MediaType: determineMediaType(extension, mimeType),
	}, nil
}

// CalculateFileMD5 calculates MD5 hash of file
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

// getMimeType detects MIME type from file content
func getMimeType(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	return http.DetectContentType(buffer[:n]), nil
}

// IsTextFile checks if file is a text file
func IsTextFile(extension, mimeType string) bool {
	ext := strings.ToLower(extension)
	for _, textExt := range constants.TextExtensions {
		if ext == textExt {
			return true
		}
	}
	return strings.HasPrefix(mimeType, "text/") ||
		strings.Contains(mimeType, "json") ||
		strings.Contains(mimeType, "xml")
}

// IsEbookFile checks if file is an e-book
func IsEbookFile(extension, mimeType string) bool {
	ext := strings.ToLower(extension)
	for _, ebookExt := range constants.EbookExtensions {
		if ext == ebookExt {
			return true
		}
	}
	return strings.Contains(mimeType, "epub") || strings.Contains(mimeType, "mobi")
}

// IsImageFile checks if file is an image
func IsImageFile(extension string) bool {
	ext := strings.ToLower(extension)
	for _, imageExt := range constants.ImageExtensions {
		if ext == imageExt {
			return true
		}
	}
	return false
}

// determineMediaType determines media type from extension and MIME type
func determineMediaType(extension, mimeType string) types.MediaType {
	if IsImageFile(extension) || strings.HasPrefix(mimeType, "image/") {
		return types.ImageMediaType
	}
	if IsTextFile(extension, mimeType) || IsEbookFile(extension, mimeType) {
		return types.DocumentMediaType
	}
	if strings.HasPrefix(mimeType, "video/") {
		return types.VideoMediaType
	}
	if strings.HasPrefix(mimeType, "audio/") {
		return types.AudioMediaType
	}
	return types.DocumentMediaType
}
