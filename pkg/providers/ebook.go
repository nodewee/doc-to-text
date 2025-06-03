package providers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"doc-to-text/pkg/config"
	"doc-to-text/pkg/interfaces"
	"doc-to-text/pkg/logger"
	"doc-to-text/pkg/types"
	"doc-to-text/pkg/utils"
)

// EbookExtractor handles e-book files using Calibre
type EbookExtractor struct {
	name        string
	config      *config.Config
	logger      *logger.Logger
	tempManager interfaces.TempFileManager
}

// NewEbookExtractor creates a new e-book extractor
func NewEbookExtractor(cfg *config.Config, log *logger.Logger) interfaces.Extractor {
	return &EbookExtractor{
		name:   "ebook",
		config: cfg,
		logger: log,
	}
}

// Extract extracts text from e-book files
func (e *EbookExtractor) Extract(ctx context.Context, inputFile string) (string, error) {
	e.logger.Progress("📚", "Processing e-book with Calibre: %s", filepath.Base(inputFile))
	e.logger.Debug("E-book extraction started for: %s", inputFile)

	// Get file info for MD5 hash to initialize temp manager
	fileInfo, err := utils.GetFileInfo(inputFile)
	if err != nil {
		return "", fmt.Errorf("failed to get file info for temp manager: %w", err)
	}

	// Initialize temp manager if not already done
	if e.tempManager == nil {
		e.tempManager = e.config.CreateTempFileManager(inputFile, fileInfo.MD5Hash, e.logger)
	}

	// Create temporary output file
	outputFile, err := e.tempManager.CreateTempFile("ebook", ".txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary output file: %w", err)
	}

	e.logger.Debug("Created temporary output file: %s", outputFile)

	// Run calibre's ebook-convert
	e.logger.Debug("Running calibre ebook-convert: %s -> %s", inputFile, outputFile)
	cmd := exec.CommandContext(ctx, e.config.CalibrePath, inputFile, outputFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.logger.Error("ebook-convert failed: %s", string(output))
		return "", fmt.Errorf("ebook-convert failed: %w", err)
	}

	e.logger.Debug("Calibre conversion completed successfully")

	// Read the converted text
	content, err := os.ReadFile(outputFile)
	if err != nil {
		e.logger.Error("Failed to read converted ebook file: %v", err)
		return "", fmt.Errorf("error reading converted ebook: %w", err)
	}

	textLen := len(content)
	e.logger.Debug("Read %d bytes from converted file", textLen)
	e.logger.Progress("✅", "E-book conversion completed successfully (%d characters)", textLen)

	return string(content), nil
}

// SupportsFile checks if this extractor supports the given file type
func (e *EbookExtractor) SupportsFile(fileInfo *types.FileInfo) bool {
	return utils.IsEbookFile(fileInfo.Extension, fileInfo.MimeType)
}

// Name returns the name of the extractor
func (e *EbookExtractor) Name() string {
	return e.name
}

// GetCacheKey returns a unique cache key for the file
func (e *EbookExtractor) GetCacheKey(fileInfo *types.FileInfo) string {
	return fmt.Sprintf("ebook-%s", fileInfo.MD5Hash)
}
