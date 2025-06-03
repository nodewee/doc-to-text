package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"doc-to-text/pkg/constants"
)

// PathUtils provides cross-platform path utilities
type PathUtils struct{}

// NewPathUtils creates a new PathUtils instance
func NewPathUtils() *PathUtils {
	return &PathUtils{}
}

// JoinPath safely joins path components using the platform-appropriate separator
func (p *PathUtils) JoinPath(elements ...string) string {
	return filepath.Join(elements...)
}

// NormalizePath normalizes a path for the current platform
func (p *PathUtils) NormalizePath(path string) string {
	// Clean the path and convert to platform-appropriate separators
	cleaned := filepath.Clean(path)

	// On Windows, ensure proper drive letter formatting
	if constants.IsWindows() && len(cleaned) >= 2 && cleaned[1] == ':' {
		// Ensure drive letter is uppercase
		if cleaned[0] >= 'a' && cleaned[0] <= 'z' {
			cleaned = strings.ToUpper(string(cleaned[0])) + cleaned[1:]
		}
	}

	return cleaned
}

// GetAbsolutePath returns the absolute path, handling cross-platform differences
func (p *PathUtils) GetAbsolutePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	return p.NormalizePath(absPath), nil
}

// EnsureDir creates a directory if it doesn't exist, with appropriate permissions
func (p *PathUtils) EnsureDir(dirPath string) error {
	normalizedPath := p.NormalizePath(dirPath)

	// Use platform-appropriate permissions
	var perm os.FileMode
	if constants.IsWindows() {
		// Windows doesn't use Unix-style permissions in the same way
		perm = 0755
	} else {
		perm = 0755
	}

	return os.MkdirAll(normalizedPath, perm)
}

// GetTempDir returns a platform-appropriate temporary directory
func (p *PathUtils) GetTempDir() string {
	tempDir := os.TempDir()
	if tempDir == "" {
		return constants.GetDefaultTempDir()
	}
	return p.NormalizePath(tempDir)
}

// CreateTempDir creates a temporary directory with a platform-appropriate name
func (p *PathUtils) CreateTempDir(prefix string) (string, error) {
	tempDir := p.GetTempDir()
	fullPrefix := prefix
	if !strings.HasSuffix(fullPrefix, "-") {
		fullPrefix += "-"
	}

	dir, err := os.MkdirTemp(tempDir, fullPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	return p.NormalizePath(dir), nil
}

// CreateTempFile creates a temporary file with appropriate naming
func (p *PathUtils) CreateTempFile(dir, prefix, suffix string) (string, error) {
	if dir == "" {
		dir = p.GetTempDir()
	}

	// Ensure directory exists
	if err := p.EnsureDir(dir); err != nil {
		return "", fmt.Errorf("failed to ensure temp directory: %w", err)
	}

	// Create temp file
	file, err := os.CreateTemp(dir, prefix+"*"+suffix)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	return p.NormalizePath(file.Name()), nil
}

// IsExecutable checks if a file is executable on the current platform
func (p *PathUtils) IsExecutable(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	if constants.IsWindows() {
		// On Windows, check file extension
		ext := strings.ToLower(filepath.Ext(filePath))
		return ext == ".exe" || ext == ".bat" || ext == ".cmd"
	} else {
		// On Unix-like systems, check execute permission
		return info.Mode()&0111 != 0
	}
}

// GetExecutableName returns the platform-appropriate executable name
func (p *PathUtils) GetExecutableName(baseName string) string {
	if constants.IsWindows() && !strings.HasSuffix(strings.ToLower(baseName), ".exe") {
		return baseName + ".exe"
	}
	return baseName
}

// ExpandPath expands environment variables and user home directory in path
func (p *PathUtils) ExpandPath(path string) (string, error) {
	// Expand environment variables
	expanded := os.ExpandEnv(path)

	// Handle home directory expansion
	if strings.HasPrefix(expanded, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}

		if expanded == "~" {
			expanded = homeDir
		} else if strings.HasPrefix(expanded, "~/") {
			expanded = filepath.Join(homeDir, expanded[2:])
		}
	}

	return p.NormalizePath(expanded), nil
}

// ValidatePath validates that a path is safe and accessible
func (p *PathUtils) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Normalize the path
	normalizedPath := p.NormalizePath(path)

	// Check for invalid characters (platform-specific)
	if constants.IsWindows() {
		invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
		baseName := filepath.Base(normalizedPath)
		for _, char := range invalidChars {
			if strings.Contains(baseName, char) {
				return fmt.Errorf("path contains invalid character '%s': %s", char, normalizedPath)
			}
		}

		// Check for reserved names on Windows
		reservedNames := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
		baseNameUpper := strings.ToUpper(strings.TrimSuffix(baseName, filepath.Ext(baseName)))
		for _, reserved := range reservedNames {
			if baseNameUpper == reserved {
				return fmt.Errorf("path uses reserved name '%s': %s", reserved, normalizedPath)
			}
		}
	}

	return nil
}

// GetRelativePath returns the relative path from base to target
func (p *PathUtils) GetRelativePath(basePath, targetPath string) (string, error) {
	normalizedBase := p.NormalizePath(basePath)
	normalizedTarget := p.NormalizePath(targetPath)

	relPath, err := filepath.Rel(normalizedBase, normalizedTarget)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	return p.NormalizePath(relPath), nil
}

// SanitizeFileName sanitizes a filename for the current platform
func (p *PathUtils) SanitizeFileName(filename string) string {
	sanitized := filename

	if constants.IsWindows() {
		// Replace invalid characters for Windows
		invalidChars := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
		for _, char := range invalidChars {
			sanitized = strings.ReplaceAll(sanitized, char, "_")
		}

		// Remove trailing dots and spaces (invalid on Windows)
		sanitized = strings.TrimRight(sanitized, ". ")
	} else {
		// For Unix-like systems, mainly avoid / and null characters
		sanitized = strings.ReplaceAll(sanitized, "/", "_")
		sanitized = strings.ReplaceAll(sanitized, "\x00", "_")
	}

	// Ensure filename is not empty
	if strings.TrimSpace(sanitized) == "" {
		sanitized = "unnamed_file"
	}

	return sanitized
}

// Global instance for easy access
var DefaultPathUtils = NewPathUtils()

// Convenience functions that use the default instance
func JoinPath(elements ...string) string {
	return DefaultPathUtils.JoinPath(elements...)
}

func NormalizePath(path string) string {
	return DefaultPathUtils.NormalizePath(path)
}

func GetAbsolutePath(path string) (string, error) {
	return DefaultPathUtils.GetAbsolutePath(path)
}

func EnsureDir(dirPath string) error {
	return DefaultPathUtils.EnsureDir(dirPath)
}

func ValidatePath(path string) error {
	return DefaultPathUtils.ValidatePath(path)
}

func SanitizeFileName(filename string) string {
	return DefaultPathUtils.SanitizeFileName(filename)
}
