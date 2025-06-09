package providers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nodewee/doc-to-text/pkg/config"
	"github.com/nodewee/doc-to-text/pkg/interfaces"
	"github.com/nodewee/doc-to-text/pkg/logger"
	"github.com/nodewee/doc-to-text/pkg/types"
	"github.com/nodewee/doc-to-text/pkg/utils"
)

// CalibreFallbackExtractor handles various document types using Calibre as fallback
type CalibreFallbackExtractor struct {
	name        string
	config      *config.Config
	logger      *logger.Logger
	tempManager interfaces.TempFileManager
}

// NewCalibreFallbackExtractor creates a new calibre fallback extractor
func NewCalibreFallbackExtractor(cfg *config.Config, log *logger.Logger) interfaces.Extractor {
	return &CalibreFallbackExtractor{
		name:   "calibre",
		config: cfg,
		logger: log,
	}
}

// Extract extracts text from various document types using Calibre
func (e *CalibreFallbackExtractor) Extract(ctx context.Context, inputFile string) (string, error) {
	e.logger.Progress("📖", "Attempting Calibre fallback extraction for: %s", inputFile)

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
	outputFile, err := e.tempManager.CreateTempFile("calibre_fallback", ".txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary output file: %w", err)
	}
	defer os.Remove(outputFile) // Clean up temporary file

	// Run calibre's ebook-convert with timeout
	cmd := exec.CommandContext(ctx, e.config.CalibrePath, inputFile, outputFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.logger.Error("Calibre fallback failed: %s", string(output))
		return "", fmt.Errorf("calibre fallback extraction failed: %w", err)
	}

	// Read the converted text
	content, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("error reading calibre-converted file: %w", err)
	}

	// Check if the output is meaningful (not just empty or error messages)
	text := string(content)
	if len(strings.TrimSpace(text)) < 10 {
		return "", fmt.Errorf("calibre fallback produced insufficient text content")
	}

	e.logger.Progress("✅", "Calibre fallback extraction completed successfully")
	return text, nil
}

// SupportsFile checks if this extractor can handle the file as fallback
func (e *CalibreFallbackExtractor) SupportsFile(fileInfo *types.FileInfo) bool {
	ext := strings.ToLower(fileInfo.Extension)

	// Support common document formats that Calibre can handle
	supportedExtensions := map[string]bool{
		"pdf":  true,
		"doc":  true,
		"docx": true,
		"rtf":  true,
		"odt":  true,
		"html": true,
		"htm":  true,
		"epub": true,
		"mobi": true,
		"txt":  true,
	}

	return supportedExtensions[ext]
}

// Name returns the name of the extractor
func (e *CalibreFallbackExtractor) Name() string {
	return e.name
}

// GetCacheKey returns a unique cache key for the file
func (e *CalibreFallbackExtractor) GetCacheKey(fileInfo *types.FileInfo) string {
	return fmt.Sprintf("calibre-%s", fileInfo.MD5Hash)
}
