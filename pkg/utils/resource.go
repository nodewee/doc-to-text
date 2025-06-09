package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nodewee/doc-to-text/pkg/logger"
)

// ResourceManager manages temporary resources and provides cleanup capabilities
type ResourceManager struct {
	tempDirs   []string
	tempFiles  []string
	mu         sync.RWMutex
	logger     *logger.Logger
	baseDir    string
	cleanupFns []func() error
}

// NewResourceManager creates a new resource manager
func NewResourceManager(baseDir string, log *logger.Logger) *ResourceManager {
	if baseDir == "" {
		baseDir = os.TempDir()
	}

	return &ResourceManager{
		tempDirs:   make([]string, 0),
		tempFiles:  make([]string, 0),
		logger:     log,
		baseDir:    baseDir,
		cleanupFns: make([]func() error, 0),
	}
}

// CreateTempDir creates a temporary directory and tracks it for cleanup
func (rm *ResourceManager) CreateTempDir(prefix string) (string, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	tempDir := filepath.Join(rm.baseDir, fmt.Sprintf("%s_%d_%d", prefix, os.Getpid(), time.Now().UnixNano()))
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	rm.tempDirs = append(rm.tempDirs, tempDir)
	rm.logger.Debug("Created temporary directory: %s", tempDir)
	return tempDir, nil
}

// CreateTempFile creates a temporary file and tracks it for cleanup
func (rm *ResourceManager) CreateTempFile(prefix, suffix string) (string, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	tempFile := filepath.Join(rm.baseDir, fmt.Sprintf("%s_%d_%d%s", prefix, os.Getpid(), time.Now().UnixNano(), suffix))
	file, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	file.Close()

	rm.tempFiles = append(rm.tempFiles, tempFile)
	rm.logger.Debug("Created temporary file: %s", tempFile)
	return tempFile, nil
}

// GetTempPath returns a path under the base directory
func (rm *ResourceManager) GetTempPath(relativePath string) string {
	return filepath.Join(rm.baseDir, relativePath)
}

// EnsureTempDir ensures the base directory exists
func (rm *ResourceManager) EnsureTempDir() error {
	return os.MkdirAll(rm.baseDir, 0755)
}

// RegisterCleanupFunc registers a cleanup function to be called on cleanup
func (rm *ResourceManager) RegisterCleanupFunc(fn func() error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.cleanupFns = append(rm.cleanupFns, fn)
}

// GetBaseTempDir returns the base temporary directory path
func (rm *ResourceManager) GetBaseTempDir() string {
	return rm.baseDir
}

// Cleanup cleans up all tracked resources
func (rm *ResourceManager) Cleanup() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	var errors []error

	// Run custom cleanup functions first
	for _, fn := range rm.cleanupFns {
		if err := fn(); err != nil {
			errors = append(errors, err)
			rm.logger.Warn("Cleanup function failed: %v", err)
		}
	}

	// Clean up temporary files
	for _, file := range rm.tempFiles {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("failed to remove temp file %s: %w", file, err))
			rm.logger.Warn("Failed to remove temporary file: %s, error: %v", file, err)
		} else {
			rm.logger.Debug("Removed temporary file: %s", file)
		}
	}

	// Clean up temporary directories
	for _, dir := range rm.tempDirs {
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("failed to remove temp dir %s: %w", dir, err))
			rm.logger.Warn("Failed to remove temporary directory: %s, error: %v", dir, err)
		} else {
			rm.logger.Debug("Removed temporary directory: %s", dir)
		}
	}

	// Clear the slices
	rm.tempDirs = rm.tempDirs[:0]
	rm.tempFiles = rm.tempFiles[:0]
	rm.cleanupFns = rm.cleanupFns[:0]

	if len(errors) > 0 {
		return fmt.Errorf("cleanup failed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// WithCleanup executes a function with automatic cleanup
func (rm *ResourceManager) WithCleanup(fn func() error) error {
	defer func() {
		if err := rm.Cleanup(); err != nil {
			rm.logger.Error("Resource cleanup failed: %v", err)
		}
	}()
	return fn()
}

// WithContext creates a context that will trigger cleanup when cancelled
func (rm *ResourceManager) WithContext(ctx context.Context) context.Context {
	newCtx, cancel := context.WithCancel(ctx)

	// Register cleanup to be called when context is done
	go func() {
		<-newCtx.Done()
		if err := rm.Cleanup(); err != nil {
			rm.logger.Error("Context cleanup failed: %v", err)
		}
	}()

	// Override cancel to ensure cleanup
	originalCancel := cancel
	cancel = func() {
		originalCancel()
		if err := rm.Cleanup(); err != nil {
			rm.logger.Error("Cancel cleanup failed: %v", err)
		}
	}

	return newCtx
}
