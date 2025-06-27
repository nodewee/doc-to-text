package providers

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"doc-to-text/pkg/config"
	"doc-to-text/pkg/constants"
	"doc-to-text/pkg/interfaces"
	"doc-to-text/pkg/logger"
	"doc-to-text/pkg/types"
	"doc-to-text/pkg/utils"
)

// EbookExtractor extracts text from e-books using Calibre
type EbookExtractor struct {
	name        string
	config      *config.Config
	logger      *logger.Logger
	fileManager *utils.FileManager
}

// NewEbookExtractor creates a new e-book extractor
func NewEbookExtractor(cfg *config.Config, log *logger.Logger) interfaces.Extractor {
	return &EbookExtractor{
		name:   "ebook",
		config: cfg,
		logger: log,
	}
}

// findCalibrePath attempts to find the Calibre ebook-convert command
func (e *EbookExtractor) findCalibrePath() (string, error) {
	// Try to find ebook-convert using shell detection
	if utils.IsCommandAvailable("ebook-convert") {
		e.logger.Debug("Found ebook-convert in PATH")
		return "ebook-convert", nil
	}

	// Common installation paths based on platform
	platformConfig := constants.GetPlatformConfig()
	for _, path := range platformConfig.CalibrePaths {
		if utils.IsCommandAvailable(path) {
			e.logger.Debug("Found ebook-convert at common path: %s", path)
			return path, nil
		}
	}

	return "", fmt.Errorf("Calibre not found:\n" +
		"Please install Calibre from https://calibre-ebook.com/ or ensure it's in your PATH")
}

// Extract implements interfaces.Extractor
func (e *EbookExtractor) Extract(ctx context.Context, inputFile string) (string, error) {
	e.logger.ProgressAlways("📖", "Extracting e-book: %s", inputFile)

	// Initialize file manager
	fileInfo, err := utils.GetFileInfo(inputFile)
	if err != nil {
		return "", utils.WrapError(err, utils.ErrorTypeIO, "failed to get file info")
	}

	e.fileManager = utils.NewFileManager(inputFile, fileInfo.MD5Hash, e.logger)
	if err := e.fileManager.EnsureBaseDir(); err != nil {
		return "", utils.WrapError(err, utils.ErrorTypeIO, "failed to create base directory")
	}

	// Find Calibre
	calibrePath, err := e.findCalibrePath()
	if err != nil {
		return "", utils.WrapError(err, utils.ErrorTypeSystem, "Calibre not found")
	}

	// Create temporary output file
	tempOutputFile, err := e.fileManager.CreateTempFile("ebook_output_", ".txt")
	if err != nil {
		return "", utils.WrapError(err, utils.ErrorTypeIO, "failed to create temporary file")
	}

	// Build and execute Calibre command
	cmd := exec.CommandContext(ctx, calibrePath, inputFile, tempOutputFile)
	e.logger.Debug("Running Calibre command: %s", cmd.String())

	// Execute command
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(outputBytes)
		e.logger.Debug("Calibre command failed: %v, output: %s", err, outputStr)
		return "", utils.WrapError(err, utils.ErrorTypeConversion, "Calibre conversion failed")
	}

	// Read extracted text
	textBytes, err := os.ReadFile(tempOutputFile)
	if err != nil {
		return "", utils.WrapError(err, utils.ErrorTypeIO, "failed to read Calibre output")
	}

	text := string(textBytes)

	// Validate text length
	if len(text) < e.config.MinTextThreshold {
		return "", utils.NewValidationError(fmt.Sprintf("extracted text too short: %d characters (minimum: %d)", len(text), e.config.MinTextThreshold), nil)
	}

	e.logger.Progress("✅", "E-book extraction successful: %d characters", len(text))
	return text, nil
}

// SupportsFile checks if this extractor supports the given file type
func (e *EbookExtractor) SupportsFile(fileInfo *types.FileInfo) bool {
	return utils.IsEbookFile(fileInfo.Extension, fileInfo.MimeType)
}

// Name returns the name of the extractor
func (e *EbookExtractor) Name() string {
	return e.name
}
