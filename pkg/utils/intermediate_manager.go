package utils

import (
	"fmt"
	"path/filepath"
	"sync"

	"doc-to-text/pkg/interfaces"
	"doc-to-text/pkg/logger"
)

// IntermediateManager manages intermediate files that persist between runs
// Directory structure: {input_file_dir}/{md5_hash}/intermediate/
type IntermediateManager struct {
	inputFile string
	md5Hash   string
	baseDir   string
	mu        sync.RWMutex
	logger    *logger.Logger
}

// Ensure IntermediateManager implements IntermediateFileManager interface
var _ interfaces.IntermediateFileManager = (*IntermediateManager)(nil)

// NewIntermediateManager creates a new intermediate file manager
func NewIntermediateManager(inputFile, md5Hash string, log *logger.Logger) *IntermediateManager {
	inputDir := filepath.Dir(inputFile)
	baseDir := filepath.Join(inputDir, md5Hash)

	return &IntermediateManager{
		inputFile: NormalizePath(inputFile),
		md5Hash:   md5Hash,
		baseDir:   NormalizePath(baseDir),
		logger:    log,
	}
}

// EnsureBaseDir ensures the base directory exists
func (im *IntermediateManager) EnsureBaseDir() error {
	return EnsureDir(im.baseDir)
}

// GetBasePath returns the base path for file operations
func (im *IntermediateManager) GetBasePath() string {
	return im.baseDir
}

// GetPath returns a path under the base directory
func (im *IntermediateManager) GetPath(relativePath string) string {
	return NormalizePath(filepath.Join(im.baseDir, relativePath))
}

// CreateIntermediateDir creates a directory for intermediate files
func (im *IntermediateManager) CreateIntermediateDir(name string) (string, error) {
	im.mu.Lock()
	defer im.mu.Unlock()

	// Sanitize the directory name for the current platform
	sanitizedName := SanitizeFileName(name)
	dirPath := im.GetPath(sanitizedName)

	if err := EnsureDir(dirPath); err != nil {
		return "", fmt.Errorf("failed to create intermediate directory: %w", err)
	}

	im.logger.Debug("Created intermediate directory: %s", dirPath)
	return dirPath, nil
}

// GetIntermediatePath returns a path for intermediate files
func (im *IntermediateManager) GetIntermediatePath(relativePath string) string {
	return im.GetPath(relativePath)
}

// GetPagesDir returns the directory for PDF page files
func (im *IntermediateManager) GetPagesDir() string {
	return im.GetPath("pages")
}

// GetPagePDFPath returns the path for a specific page PDF
func (im *IntermediateManager) GetPagePDFPath(pageNum int) string {
	return im.GetPath(filepath.Join("pages", fmt.Sprintf("page_%d.pdf", pageNum)))
}

// GetPageTextPath returns the path for a specific page text
func (im *IntermediateManager) GetPageTextPath(pageNum int) string {
	return im.GetPath(filepath.Join("pages", fmt.Sprintf("page_%d.txt", pageNum)))
}

// GetPageImagePath returns the path for a specific page image
func (im *IntermediateManager) GetPageImagePath(pageNum int) string {
	return im.GetPath(filepath.Join("pages", fmt.Sprintf("page_%d.png", pageNum)))
}

// GetTextFilePath returns the final text output file path
func (im *IntermediateManager) GetTextFilePath() string {
	return im.GetPath("text.txt")
}

// GetOCRDataPath returns the path for OCR result data
func (im *IntermediateManager) GetOCRDataPath() string {
	return im.GetPath("ocr_data.json")
}

// Cleanup performs cleanup operations (for intermediate files, this is optional)
func (im *IntermediateManager) Cleanup() error {
	// For intermediate files, cleanup is generally not performed
	// as these files are meant to persist for caching and resume capability
	im.logger.Debug("Intermediate file cleanup skipped (files preserved for caching)")
	return nil
}
