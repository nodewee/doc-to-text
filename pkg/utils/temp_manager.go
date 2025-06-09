package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/nodewee/doc-to-text/pkg/interfaces"
	"github.com/nodewee/doc-to-text/pkg/logger"
)

// SimpleTempManager manages temporary files that are cleaned up after processing
// Directory structure: {input_file_dir}/{md5_hash}/temp/
type SimpleTempManager struct {
	inputFile  string
	md5Hash    string
	baseDir    string
	tempFiles  []string
	tempDirs   []string
	mu         sync.RWMutex
	logger     *logger.Logger
	cleanupFns []func() error
}

// Ensure SimpleTempManager implements TempFileManager interface
var _ interfaces.TempFileManager = (*SimpleTempManager)(nil)

// NewSimpleTempManager creates a new temporary file manager
func NewSimpleTempManager(inputFile, md5Hash string, log *logger.Logger) *SimpleTempManager {
	inputDir := filepath.Dir(inputFile)
	baseDir := filepath.Join(inputDir, md5Hash, "temp")

	return &SimpleTempManager{
		inputFile: NormalizePath(inputFile),
		md5Hash:   md5Hash,
		baseDir:   NormalizePath(baseDir),
		logger:    log,
	}
}

// EnsureBaseDir ensures the base directory exists
func (tm *SimpleTempManager) EnsureBaseDir() error {
	return EnsureDir(tm.baseDir)
}

// GetBasePath returns the base path for file operations
func (tm *SimpleTempManager) GetBasePath() string {
	return tm.baseDir
}

// GetPath returns a path under the base directory
func (tm *SimpleTempManager) GetPath(relativePath string) string {
	return NormalizePath(filepath.Join(tm.baseDir, relativePath))
}

// CreateTempDir creates a temporary directory
func (tm *SimpleTempManager) CreateTempDir(prefix string) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if err := tm.EnsureBaseDir(); err != nil {
		return "", fmt.Errorf("failed to ensure base directory: %w", err)
	}

	// Sanitize prefix for the current platform
	sanitizedPrefix := SanitizeFileName(prefix)
	if sanitizedPrefix == "" {
		sanitizedPrefix = "temp"
	}

	// Use platform-specific temp directory creation
	tempDir, err := DefaultPathUtils.CreateTempDir(sanitizedPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	tm.tempDirs = append(tm.tempDirs, tempDir)
	tm.logger.Debug("Created temp directory: %s", tempDir)
	return tempDir, nil
}

// CreateTempFile creates a temporary file
func (tm *SimpleTempManager) CreateTempFile(prefix, suffix string) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if err := tm.EnsureBaseDir(); err != nil {
		return "", fmt.Errorf("failed to ensure base directory: %w", err)
	}

	// Sanitize prefix and suffix for the current platform
	sanitizedPrefix := SanitizeFileName(prefix)
	sanitizedSuffix := suffix
	if sanitizedPrefix == "" {
		sanitizedPrefix = "temp"
	}

	// Use platform-specific temp file creation
	tempFile, err := DefaultPathUtils.CreateTempFile(tm.baseDir, sanitizedPrefix, sanitizedSuffix)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	tm.tempFiles = append(tm.tempFiles, tempFile)
	tm.logger.Debug("Created temp file: %s", tempFile)
	return tempFile, nil
}

// RegisterCleanupFunc registers a cleanup function
func (tm *SimpleTempManager) RegisterCleanupFunc(fn func() error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.cleanupFns = append(tm.cleanupFns, fn)
}

// WithCleanup executes a function with automatic cleanup
func (tm *SimpleTempManager) WithCleanup(fn func() error) error {
	defer func() {
		if err := tm.Cleanup(); err != nil {
			tm.logger.Error("Temporary file cleanup failed: %v", err)
		}
	}()
	return fn()
}

// Cleanup cleans up temporary resources
func (tm *SimpleTempManager) Cleanup() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	var errors []error

	// Run custom cleanup functions first
	for _, fn := range tm.cleanupFns {
		if err := fn(); err != nil {
			errors = append(errors, err)
			tm.logger.Warn("Cleanup function failed: %v", err)
		}
	}

	// Clean up temporary files
	for _, file := range tm.tempFiles {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("failed to remove temp file %s: %w", file, err))
			tm.logger.Warn("Failed to remove temporary file: %s, error: %v", file, err)
		} else {
			tm.logger.Debug("Removed temporary file: %s", file)
		}
	}

	// Clean up temporary directories
	for _, dir := range tm.tempDirs {
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("failed to remove temp dir %s: %w", dir, err))
			tm.logger.Warn("Failed to remove temporary directory: %s, error: %v", dir, err)
		} else {
			tm.logger.Debug("Removed temporary directory: %s", dir)
		}
	}

	// Clear the slices
	tm.tempFiles = tm.tempFiles[:0]
	tm.tempDirs = tm.tempDirs[:0]
	tm.cleanupFns = tm.cleanupFns[:0]

	if len(errors) > 0 {
		return fmt.Errorf("cleanup failed with %d errors: %v", len(errors), errors)
	}

	return nil
}
